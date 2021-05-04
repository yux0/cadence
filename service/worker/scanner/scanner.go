// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/uber-go/tally"
	"go.uber.org/zap"

	"go.uber.org/cadence/.gen/go/shared"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/worker"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/backoff"
	"github.com/uber/cadence/common/cluster"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/resource"
	"github.com/uber/cadence/common/service/config"
	"github.com/uber/cadence/common/service/dynamicconfig"
	"github.com/uber/cadence/service/worker/scanner/shardscanner"
)

const (
	// scannerStartUpDelay is to let services warm up
	scannerStartUpDelay = time.Second * 4
)

type (
	contextKey string

	// Config defines the configuration for scanner
	Config struct {
		// ScannerPersistenceMaxQPS the max rate of calls to persistence
		// Right now is being used by historyScanner to determine the rate of persistence API calls
		ScannerPersistenceMaxQPS dynamicconfig.IntPropertyFn
		// Persistence contains the persistence configuration
		Persistence *config.Persistence
		// ClusterMetadata contains the metadata for this cluster
		ClusterMetadata cluster.Metadata
		// TaskListScannerEnabled indicates if taskList scanner should be started as part of scanner
		TaskListScannerEnabled dynamicconfig.BoolPropertyFn
		// HistoryScannerEnabled indicates if history scanner should be started as part of scanner
		HistoryScannerEnabled dynamicconfig.BoolPropertyFn
		// ShardScanners is a list of shard scanner configs
		ShardScanners              []*shardscanner.ScannerConfig
		MaxWorkflowRetentionInDays dynamicconfig.IntPropertyFn
	}

	// BootstrapParams contains the set of params needed to bootstrap
	// the scanner sub-system
	BootstrapParams struct {
		// Config contains the configuration for scanner
		Config Config
		// TallyScope is an instance of tally metrics scope
		TallyScope tally.Scope
	}

	// scannerContext is the context object that get's
	// passed around within the scanner workflows / activities
	scannerContext struct {
		resource resource.Resource
		cfg      Config
	}

	// Scanner is the background sub-system that does full scans
	// of database tables to cleanup resources, monitor anomalies
	// and emit stats for analytics
	Scanner struct {
		context    scannerContext
		tallyScope tally.Scope
		zapLogger  *zap.Logger
	}
)

// New returns a new instance of scanner daemon
// Scanner is the background sub-system that does full
// scans of database tables in an attempt to cleanup
// resources, monitor system anamolies and emit stats
// for analysis and alerting
func New(
	resource resource.Resource,
	params *BootstrapParams,
) *Scanner {

	zapLogger, err := zap.NewProduction()
	if err != nil {
		resource.GetLogger().Fatal("failed to initialize zap logger", tag.Error(err))
	}
	return &Scanner{
		context: scannerContext{
			resource: resource,
			cfg:      params.Config,
		},
		tallyScope: params.TallyScope,
		zapLogger:  zapLogger.Named("scanner"),
	}

}

// Start starts the scanner
func (s *Scanner) Start() error {
	ctx := context.Background()
	var workerTaskListNames []string
	var wtl []string

	for _, sc := range s.context.cfg.ShardScanners {
		ctx, wtl = s.startShardScanner(ctx, sc)
		workerTaskListNames = append(workerTaskListNames, wtl...)
	}

	switch s.context.cfg.Persistence.DefaultStoreType() {
	case config.StoreTypeSQL:
		if s.context.cfg.TaskListScannerEnabled() {
			ctx = s.startScanner(
				ctx,
				tlScannerWFStartOptions,
				tlScannerWFTypeName)
			workerTaskListNames = append(workerTaskListNames, tlScannerTaskListName)
		}
	case config.StoreTypeCassandra:
		if s.context.cfg.HistoryScannerEnabled() {
			ctx = s.startScanner(
				ctx,
				historyScannerWFStartOptions,
				historyScannerWFTypeName)
			workerTaskListNames = append(workerTaskListNames, historyScannerTaskListName)
		}
	}

	workerOpts := worker.Options{
		Logger:                                 s.zapLogger,
		MetricsScope:                           s.tallyScope,
		MaxConcurrentActivityExecutionSize:     maxConcurrentActivityExecutionSize,
		MaxConcurrentDecisionTaskExecutionSize: maxConcurrentDecisionTaskExecutionSize,
		BackgroundActivityContext:              ctx,
	}

	for _, tl := range workerTaskListNames {
		if err := worker.New(s.context.resource.GetSDKClient(), common.SystemLocalDomainName, tl, workerOpts).Start(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scanner) startScanner(ctx context.Context, options client.StartWorkflowOptions, workflowName string) context.Context {
	go s.startWorkflowWithRetry(options, workflowName, nil)
	return NewScannerContext(ctx, workflowName, s.context)
}

func (s *Scanner) startShardScanner(
	ctx context.Context,
	config *shardscanner.ScannerConfig,
) (context.Context, []string) {
	workerTaskListNames := []string{}
	if config.DynamicParams.ScannerEnabled() {
		ctx = shardscanner.NewScannerContext(
			ctx,
			config.ScannerWFTypeName,
			shardscanner.NewShardScannerContext(s.context.resource, config),
		)
		go s.startWorkflowWithRetry(
			config.StartWorkflowOptions,
			config.ScannerWFTypeName,
			shardscanner.ScannerWorkflowParams{
				Shards: shardscanner.Shards{
					Range: &shardscanner.ShardRange{
						Min: 0,
						Max: s.context.cfg.Persistence.NumHistoryShards,
					},
				},
			})

		workerTaskListNames = append(workerTaskListNames, config.StartWorkflowOptions.TaskList)
	}

	if config.DynamicParams.FixerEnabled() {
		ctx = shardscanner.NewFixerContext(
			ctx,
			config.FixerWFTypeName,
			shardscanner.NewShardFixerContext(s.context.resource, config),
		)
		go s.startWorkflowWithRetry(
			config.StartFixerOptions,
			config.FixerWFTypeName,
			shardscanner.FixerWorkflowParams{
				ScannerWorkflowWorkflowID: config.StartWorkflowOptions.ID,
			},
		)

		workerTaskListNames = append(workerTaskListNames, config.StartFixerOptions.TaskList)
	}

	return ctx, workerTaskListNames
}

func (s *Scanner) startWorkflowWithRetry(
	options client.StartWorkflowOptions,
	workflowType string,
	workflowArg interface{},
) {
	// let history / matching service warm up
	time.Sleep(scannerStartUpDelay)
	res := s.context.resource
	sdkClient := client.NewClient(
		res.GetSDKClient(),
		common.SystemLocalDomainName,
		nil, /* &client.Options{} */
	)
	policy := backoff.NewExponentialRetryPolicy(time.Second)
	policy.SetMaximumInterval(time.Minute)
	policy.SetExpirationInterval(backoff.NoInterval)
	err := backoff.Retry(func() error {
		return s.startWorkflow(sdkClient, options, workflowType, workflowArg)
	}, policy, func(err error) bool {
		return true
	})
	if err != nil {
		res.GetLogger().Fatal("unable to start scanner", tag.WorkflowType(workflowType), tag.Error(err))
	} else {
		res.GetLogger().Info("starting scanner", tag.WorkflowType(workflowType))
	}
}

func (s *Scanner) startWorkflow(
	client client.Client,
	options client.StartWorkflowOptions,
	workflowType string,
	workflowArg interface{},
) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	var err error
	if workflowArg != nil {
		_, err = client.StartWorkflow(ctx, options, workflowType, workflowArg)
	} else {
		_, err = client.StartWorkflow(ctx, options, workflowType)
	}

	cancel()

	switch err.(type) {
	case *shared.WorkflowExecutionAlreadyStartedError, nil:
		return nil
	default:
		return err
	}
}

// NewScannerContext provides context to be used as background activity context
// it uses typed, private key to reduce access scope
func NewScannerContext(ctx context.Context, workflowName string, scannerContext scannerContext) context.Context {
	return context.WithValue(ctx, contextKey(workflowName), scannerContext)
}

// GetScannerContext extracts scanner context from activity context
// it uses typed, private key to reduce access scope
func GetScannerContext(ctx context.Context) (scannerContext, error) {
	info := activity.GetInfo(ctx)
	if info.WorkflowType == nil {
		return scannerContext{}, fmt.Errorf("workflowType is nil")
	}
	val, ok := ctx.Value(contextKey(info.WorkflowType.Name)).(scannerContext)
	if !ok {
		return scannerContext{}, fmt.Errorf("context type is not %T for a key %q", val, info.WorkflowType.Name)
	}
	return val, nil
}
