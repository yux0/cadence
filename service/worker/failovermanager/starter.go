// Copyright (c) 2017-2020 Uber Technologies Inc.
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

package failovermanager

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/worker"
	"go.uber.org/cadence/workflow"

	"github.com/uber/cadence/client"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/cluster"
	"github.com/uber/cadence/common/dynamicconfig"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/metrics"
)

type (

	// Config defines the configuration for failover
	Config struct {
		AdminOperationToken dynamicconfig.StringPropertyFn
		// ClusterMetadata contains the metadata for this cluster
		ClusterMetadata cluster.Metadata
	}

	// BootstrapParams contains the set of params needed to bootstrap
	// failover manager
	BootstrapParams struct {
		// Config contains the configuration for scanner
		Config Config
		// ServiceClient is an instance of cadence service client
		ServiceClient workflowserviceclient.Interface
		// MetricsClient is an instance of metrics object for emitting stats
		MetricsClient metrics.Client
		Logger        log.Logger
		// TallyScope is an instance of tally metrics scope
		TallyScope tally.Scope
		// ClientBean is an instance of client.Bean for a collection of clients
		ClientBean client.Bean
	}

	// FailoverManager of cadence worker service
	FailoverManager struct {
		cfg           Config
		svcClient     workflowserviceclient.Interface
		clientBean    client.Bean
		metricsClient metrics.Client
		tallyScope    tally.Scope
		logger        log.Logger
		worker        worker.Worker
	}
)

// New returns a new instance of FailoverManager
func New(params *BootstrapParams) *FailoverManager {
	return &FailoverManager{
		cfg:           params.Config,
		svcClient:     params.ServiceClient,
		metricsClient: params.MetricsClient,
		tallyScope:    params.TallyScope,
		logger:        params.Logger.WithTags(tag.ComponentBatcher),
		clientBean:    params.ClientBean,
	}
}

// Start starts the worker
func (s *FailoverManager) Start() error {
	ctx := context.WithValue(context.Background(), failoverManagerContextKey, s)
	workerOpts := worker.Options{
		MetricsScope:              s.tallyScope,
		BackgroundActivityContext: ctx,
		Tracer:                    opentracing.GlobalTracer(),
	}
	failoverWorker := worker.New(s.svcClient, common.SystemLocalDomainName, TaskListName, workerOpts)
	failoverWorker.RegisterWorkflowWithOptions(FailoverWorkflow, workflow.RegisterOptions{Name: WorkflowTypeName})
	failoverWorker.RegisterActivityWithOptions(FailoverActivity, activity.RegisterOptions{Name: failoverActivityName})
	failoverWorker.RegisterActivityWithOptions(GetDomainsActivity, activity.RegisterOptions{Name: getDomainsActivityName})
	s.worker = failoverWorker
	return failoverWorker.Start()
}

// Stop stops the worker
func (s *FailoverManager) Stop() {
	s.worker.Stop()
}
