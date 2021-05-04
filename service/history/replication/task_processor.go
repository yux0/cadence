// Copyright (c) 2020 Uber Technologies, Inc.
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

//go:generate mockgen -package $GOPACKAGE -source $GOFILE -destination task_processor_mock.go -self_package github.com/uber/cadence/service/history/replication

package replication

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync/atomic"
	"time"

	"go.uber.org/yarpc/yarpcerrors"

	h "github.com/uber/cadence/.gen/go/history"
	r "github.com/uber/cadence/.gen/go/replicator"
	"github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/backoff"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/quotas"
	"github.com/uber/cadence/service/history/config"
	"github.com/uber/cadence/service/history/engine"
	"github.com/uber/cadence/service/history/execution"
	"github.com/uber/cadence/service/history/shard"
)

const (
	dropSyncShardTaskTimeThreshold = 10 * time.Minute
	replicationTimeout             = 30 * time.Second
	dlqErrorRetryWait              = time.Second
	dlqMetricsEmitTimerInterval    = 5 * time.Minute
	dlqMetricsEmitTimerCoefficient = 0.05
)

var (
	// ErrUnknownReplicationTask is the error to indicate unknown replication task type
	ErrUnknownReplicationTask = &shared.BadRequestError{Message: "unknown replication task"}
)

type (
	// TaskProcessor is responsible for processing replication tasks for a shard.
	TaskProcessor interface {
		common.Daemon
	}

	// taskProcessorImpl is responsible for processing replication tasks for a shard.
	taskProcessorImpl struct {
		currentCluster    string
		sourceCluster     string
		status            int32
		shard             shard.Context
		historyEngine     engine.Engine
		historySerializer persistence.PayloadSerializer
		config            *config.Config
		metricsClient     metrics.Client
		logger            log.Logger
		taskExecutor      TaskExecutor
		hostRateLimiter   *quotas.DynamicRateLimiter
		shardRateLimiter  *quotas.DynamicRateLimiter

		taskRetryPolicy backoff.RetryPolicy
		dlqRetryPolicy  backoff.RetryPolicy
		noTaskRetrier   backoff.Retrier

		lastProcessedMessageID int64
		lastRetrievedMessageID int64

		requestChan   chan<- *request
		syncShardChan chan *r.SyncShardStatus
		done          chan struct{}
	}

	request struct {
		token    *r.ReplicationToken
		respChan chan<- *r.ReplicationMessages
	}
)

var _ TaskProcessor = (*taskProcessorImpl)(nil)

// NewTaskProcessor creates a new replication task processor.
func NewTaskProcessor(
	shard shard.Context,
	historyEngine engine.Engine,
	config *config.Config,
	metricsClient metrics.Client,
	taskFetcher TaskFetcher,
	taskExecutor TaskExecutor,
) TaskProcessor {
	shardID := shard.GetShardID()
	firstRetryPolicy := backoff.NewExponentialRetryPolicy(config.ReplicationTaskProcessorErrorRetryWait(shardID))
	firstRetryPolicy.SetMaximumAttempts(config.ReplicationTaskProcessorErrorRetryMaxAttempts(shardID))
	secondRetryPolicy := backoff.NewExponentialRetryPolicy(config.ReplicationTaskProcessorErrorSecondRetryWait(shardID))
	secondRetryPolicy.SetMaximumInterval(config.ReplicationTaskProcessorErrorSecondRetryMaxWait(shardID))
	secondRetryPolicy.SetExpirationInterval(config.ReplicationTaskProcessorErrorSecondRetryExpiration(shardID))
	taskRetryPolicy := backoff.NewMultiPhasesRetryPolicy(firstRetryPolicy, secondRetryPolicy)

	dlqRetryPolicy := backoff.NewExponentialRetryPolicy(dlqErrorRetryWait)
	dlqRetryPolicy.SetExpirationInterval(backoff.NoInterval)

	noTaskBackoffPolicy := backoff.NewExponentialRetryPolicy(config.ReplicationTaskProcessorNoTaskRetryWait(shardID))
	noTaskBackoffPolicy.SetBackoffCoefficient(1)
	noTaskBackoffPolicy.SetExpirationInterval(backoff.NoInterval)
	noTaskRetrier := backoff.NewRetrier(noTaskBackoffPolicy, backoff.SystemClock)
	return &taskProcessorImpl{
		currentCluster:    shard.GetClusterMetadata().GetCurrentClusterName(),
		sourceCluster:     taskFetcher.GetSourceCluster(),
		status:            common.DaemonStatusInitialized,
		shard:             shard,
		historyEngine:     historyEngine,
		historySerializer: persistence.NewPayloadSerializer(),
		config:            config,
		metricsClient:     metricsClient,
		logger:            shard.GetLogger(),
		taskExecutor:      taskExecutor,
		hostRateLimiter:   taskFetcher.GetRateLimiter(),
		shardRateLimiter: quotas.NewDynamicRateLimiter(func() float64 {
			return config.ReplicationTaskProcessorShardQPS()
		}),
		taskRetryPolicy:        taskRetryPolicy,
		dlqRetryPolicy:         dlqRetryPolicy,
		noTaskRetrier:          noTaskRetrier,
		requestChan:            taskFetcher.GetRequestChan(),
		syncShardChan:          make(chan *r.SyncShardStatus, 1),
		done:                   make(chan struct{}),
		lastProcessedMessageID: common.EmptyMessageID,
		lastRetrievedMessageID: common.EmptyMessageID,
	}
}

// Start starts the processor
func (p *taskProcessorImpl) Start() {
	if !atomic.CompareAndSwapInt32(&p.status, common.DaemonStatusInitialized, common.DaemonStatusStarted) {
		return
	}

	go p.processorLoop()
	go p.syncShardStatusLoop()
	go p.cleanupReplicationTaskLoop()
	go p.emitDLQSizeMetricsLoop()
	p.logger.Info("ReplicationTaskProcessor started.")
}

// Stop stops the processor
func (p *taskProcessorImpl) Stop() {
	if !atomic.CompareAndSwapInt32(&p.status, common.DaemonStatusStarted, common.DaemonStatusStopped) {
		return
	}

	p.logger.Debug("ReplicationTaskProcessor shutting down.")
	close(p.done)
}

func (p *taskProcessorImpl) processorLoop() {
	defer func() {
		p.logger.Debug("Closing replication task processor.", tag.ReadLevel(p.lastRetrievedMessageID))
	}()

Loop:
	for {
		// for each iteration, do close check first
		select {
		case <-p.done:
			p.logger.Debug("ReplicationTaskProcessor shutting down.")
			return
		default:
		}

		respChan := p.sendFetchMessageRequest()

		select {
		case response, ok := <-respChan:
			if !ok {
				p.logger.Debug("Fetch replication messages chan closed.")
				continue Loop
			}

			p.logger.Debug("Got fetch replication messages response.",
				tag.ReadLevel(response.GetLastRetrievedMessageId()),
				tag.Bool(response.GetHasMore()),
				tag.Counter(len(response.GetReplicationTasks())),
			)

			p.taskProcessingStartWait()
			p.processResponse(response)
		case <-p.done:
			return
		}
	}
}

func (p *taskProcessorImpl) cleanupReplicationTaskLoop() {

	shardID := p.shard.GetShardID()
	timer := time.NewTimer(backoff.JitDuration(
		p.config.ReplicationTaskProcessorCleanupInterval(shardID),
		p.config.ReplicationTaskProcessorCleanupJitterCoefficient(shardID),
	))
	for {
		select {
		case <-p.done:
			timer.Stop()
			return
		case <-timer.C:
			if err := p.cleanupAckedReplicationTasks(); err != nil {
				p.logger.Error("Failed to clean up replication messages.", tag.Error(err))
				p.metricsClient.Scope(metrics.ReplicationTaskCleanupScope).IncCounter(metrics.ReplicationTaskCleanupFailure)
			}
			timer.Reset(backoff.JitDuration(
				p.config.ReplicationTaskProcessorCleanupInterval(shardID),
				p.config.ReplicationTaskProcessorCleanupJitterCoefficient(shardID),
			))
		}
	}
}

func (p *taskProcessorImpl) cleanupAckedReplicationTasks() error {

	clusterMetadata := p.shard.GetClusterMetadata()
	currentCluster := clusterMetadata.GetCurrentClusterName()
	minAckLevel := int64(math.MaxInt64)
	for clusterName, clusterInfo := range clusterMetadata.GetAllClusterInfo() {
		if !clusterInfo.Enabled {
			continue
		}

		if clusterName != currentCluster {
			ackLevel := p.shard.GetClusterReplicationLevel(clusterName)
			if ackLevel < minAckLevel {
				minAckLevel = ackLevel
			}
		}
	}
	p.logger.Debug("Cleaning up replication task queue.", tag.ReadLevel(minAckLevel))
	p.metricsClient.Scope(metrics.ReplicationTaskCleanupScope).IncCounter(metrics.ReplicationTaskCleanupCount)
	p.metricsClient.Scope(metrics.ReplicationTaskFetcherScope,
		metrics.TargetClusterTag(p.currentCluster),
	).RecordTimer(
		metrics.ReplicationTasksLag,
		time.Duration(p.shard.GetTransferMaxReadLevel()-minAckLevel),
	)
	return p.shard.GetExecutionManager().RangeCompleteReplicationTask(
		context.Background(),
		&persistence.RangeCompleteReplicationTaskRequest{
			InclusiveEndTaskID: minAckLevel,
		},
	)
}

func (p *taskProcessorImpl) sendFetchMessageRequest() <-chan *r.ReplicationMessages {
	respChan := make(chan *r.ReplicationMessages, 1)
	// TODO: when we support prefetching, LastRetrievedMessageId can be different than LastProcessedMessageId
	p.requestChan <- &request{
		token: &r.ReplicationToken{
			ShardID:                common.Int32Ptr(int32(p.shard.GetShardID())),
			LastRetrievedMessageId: common.Int64Ptr(p.lastRetrievedMessageID),
			LastProcessedMessageId: common.Int64Ptr(p.lastProcessedMessageID),
		},
		respChan: respChan,
	}
	return respChan
}

func (p *taskProcessorImpl) processResponse(response *r.ReplicationMessages) {

	select {
	case p.syncShardChan <- response.GetSyncShardStatus():
	default:
	}

	scope := p.metricsClient.Scope(metrics.ReplicationTaskFetcherScope, metrics.TargetClusterTag(p.sourceCluster))
	batchRequestStartTime := time.Now()
	ctx := context.Background()
	for _, replicationTask := range response.ReplicationTasks {
		// TODO: move to MultiStageRateLimiter
		_ = p.hostRateLimiter.Wait(ctx)
		_ = p.shardRateLimiter.Wait(ctx)
		err := p.processSingleTask(replicationTask)
		if err != nil {
			// Encounter error and skip updating ack levels
			return
		}
	}

	// Note here we check replication tasks instead of hasMore. The expectation is that in a steady state
	// we will receive replication tasks but hasMore is false (meaning that we are always catching up).
	// So hasMore might not be a good indicator for additional wait.
	if len(response.ReplicationTasks) == 0 {
		backoffDuration := p.noTaskRetrier.NextBackOff()
		time.Sleep(backoffDuration)
	} else {
		scope.RecordTimer(metrics.ReplicationTasksAppliedLatency, time.Now().Sub(batchRequestStartTime))
	}

	p.lastProcessedMessageID = response.GetLastRetrievedMessageId()
	p.lastRetrievedMessageID = response.GetLastRetrievedMessageId()
	scope.UpdateGauge(metrics.LastRetrievedMessageID, float64(p.lastRetrievedMessageID))
	p.noTaskRetrier.Reset()
}

func (p *taskProcessorImpl) syncShardStatusLoop() {

	timer := time.NewTimer(backoff.JitDuration(
		p.config.ShardSyncMinInterval(),
		p.config.ShardSyncTimerJitterCoefficient(),
	))
	var syncShardTask *r.SyncShardStatus
	for {
		select {
		case syncShardRequest := <-p.syncShardChan:
			syncShardTask = syncShardRequest
		case <-timer.C:
			if err := p.handleSyncShardStatus(
				syncShardTask,
			); err != nil {
				p.logger.Error("failed to sync shard status", tag.Error(err))
				p.metricsClient.Scope(metrics.HistorySyncShardStatusScope).IncCounter(metrics.SyncShardFromRemoteFailure)
			}
			timer.Reset(backoff.JitDuration(
				p.config.ShardSyncMinInterval(),
				p.config.ShardSyncTimerJitterCoefficient(),
			))
		case <-p.done:
			timer.Stop()
			return
		}
	}
}

func (p *taskProcessorImpl) handleSyncShardStatus(
	status *r.SyncShardStatus,
) error {

	if status == nil ||
		p.shard.GetTimeSource().Now().Sub(
			time.Unix(0, status.GetTimestamp())) > dropSyncShardTaskTimeThreshold {
		return nil
	}
	p.metricsClient.Scope(metrics.HistorySyncShardStatusScope).IncCounter(metrics.SyncShardFromRemoteCounter)
	ctx, cancel := context.WithTimeout(context.Background(), replicationTimeout)
	defer cancel()
	return p.historyEngine.SyncShardStatus(ctx, &h.SyncShardStatusRequest{
		SourceCluster: common.StringPtr(p.sourceCluster),
		ShardId:       common.Int64Ptr(int64(p.shard.GetShardID())),
		Timestamp:     status.Timestamp,
	})
}

func (p *taskProcessorImpl) processSingleTask(replicationTask *r.ReplicationTask) error {
	retryTransientError := func() error {
		return backoff.Retry(
			func() error {
				select {
				case <-p.done:
					// if the processor is stopping, skip the task
					// the ack level will not update and the new shard owner will retry the task.
					return nil
				default:
					return p.processTaskOnce(replicationTask)
				}
			},
			p.taskRetryPolicy,
			isTransientRetryableError)
	}

	//Handle service busy error
	err := backoff.Retry(
		retryTransientError,
		common.CreateReplicationServiceBusyRetryPolicy(),
		common.IsServiceBusyError,
	)

	switch {
	case err == nil:
		return nil
	case common.IsServiceBusyError(err):
		return err
	case err == execution.ErrMissingVersionHistories:
		// skip the workflow without version histories
		p.logger.Warn("Encounter workflow withour version histories")
		return nil
	default:
		//handle error
	}

	// handle error to DLQ
	select {
	case <-p.done:
		p.logger.Warn("Skip adding new messages to DLQ.", tag.Error(err))
		return err
	default:
		p.logger.Error(
			"Failed to apply replication task after retry. Putting task into DLQ.",
			tag.TaskID(replicationTask.GetSourceTaskId()),
			tag.Error(err),
		)
		return p.putReplicationTaskToDLQ(replicationTask)
	}
}

func (p *taskProcessorImpl) processTaskOnce(replicationTask *r.ReplicationTask) error {
	ts := p.shard.GetTimeSource()
	startTime := ts.Now()
	scope, err := p.taskExecutor.execute(
		replicationTask,
		false)

	if err != nil {
		p.updateFailureMetric(scope, err)
	} else {
		now := ts.Now()
		mScope := p.metricsClient.Scope(scope, metrics.TargetClusterTag(p.sourceCluster))
		// emit the number of replication tasks
		mScope.IncCounter(metrics.ReplicationTasksApplied)
		// emit single task processing latency
		mScope.RecordTimer(metrics.TaskProcessingLatency, now.Sub(startTime))
		// emit latency from task generated to task received
		mScope.RecordTimer(
			metrics.ReplicationTaskLatency,
			now.Sub(time.Unix(0, replicationTask.GetCreationTime())),
		)
	}

	return err
}

func (p *taskProcessorImpl) putReplicationTaskToDLQ(replicationTask *r.ReplicationTask) error {
	request, err := p.generateDLQRequest(replicationTask)
	if err != nil {
		p.logger.Error("Failed to generate DLQ replication task.", tag.Error(err))
		// We cannot deserialize the task. Dropping it.
		return nil
	}
	p.logger.Info("Put history replication to DLQ",
		tag.WorkflowDomainID(request.TaskInfo.GetDomainID()),
		tag.WorkflowID(request.TaskInfo.GetWorkflowID()),
		tag.WorkflowRunID(request.TaskInfo.GetRunID()),
		tag.TaskID(request.TaskInfo.GetTaskID()),
		tag.ShardID(p.shard.GetShardID()),
	)

	p.metricsClient.Scope(
		metrics.ReplicationDLQStatsScope,
		metrics.TargetClusterTag(p.sourceCluster),
		metrics.InstanceTag(strconv.Itoa(p.shard.GetShardID())),
	).UpdateGauge(
		metrics.ReplicationDLQMaxLevelGauge,
		float64(request.TaskInfo.GetTaskID()),
	)
	// The following is guaranteed to success or retry forever until processor is shutdown.
	return backoff.Retry(func() error {
		err := p.shard.GetExecutionManager().PutReplicationTaskToDLQ(context.Background(), request)
		if err != nil {
			p.logger.Error("Failed to put replication task to DLQ.", tag.Error(err))
			p.metricsClient.IncCounter(metrics.ReplicationTaskFetcherScope, metrics.ReplicationDLQFailed)
		}
		return err
	}, p.dlqRetryPolicy, p.shouldRetryDLQ)
}

func (p *taskProcessorImpl) generateDLQRequest(
	replicationTask *r.ReplicationTask,
) (*persistence.PutReplicationTaskToDLQRequest, error) {
	switch *replicationTask.TaskType {
	case r.ReplicationTaskTypeSyncActivity:
		taskAttributes := replicationTask.GetSyncActivityTaskAttributes()
		return &persistence.PutReplicationTaskToDLQRequest{
			SourceClusterName: p.sourceCluster,
			TaskInfo: &persistence.ReplicationTaskInfo{
				DomainID:    taskAttributes.GetDomainId(),
				WorkflowID:  taskAttributes.GetWorkflowId(),
				RunID:       taskAttributes.GetRunId(),
				TaskID:      replicationTask.GetSourceTaskId(),
				TaskType:    persistence.ReplicationTaskTypeSyncActivity,
				ScheduledID: taskAttributes.GetScheduledId(),
			},
		}, nil

	case r.ReplicationTaskTypeHistoryV2:
		taskAttributes := replicationTask.GetHistoryTaskV2Attributes()
		eventsDataBlob := persistence.NewDataBlobFromThrift(taskAttributes.GetEvents())
		events, err := p.historySerializer.DeserializeBatchEvents(eventsDataBlob)
		if err != nil {
			return nil, err
		}

		if len(events) == 0 {
			p.logger.Error("Empty events in a batch")
			return nil, fmt.Errorf("corrupted history event batch, empty events")
		}

		return &persistence.PutReplicationTaskToDLQRequest{
			SourceClusterName: p.sourceCluster,
			TaskInfo: &persistence.ReplicationTaskInfo{
				DomainID:     taskAttributes.GetDomainId(),
				WorkflowID:   taskAttributes.GetWorkflowId(),
				RunID:        taskAttributes.GetRunId(),
				TaskID:       replicationTask.GetSourceTaskId(),
				TaskType:     persistence.ReplicationTaskTypeHistory,
				FirstEventID: events[0].GetEventId(),
				NextEventID:  events[len(events)-1].GetEventId() + 1,
				Version:      events[0].GetVersion(),
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown replication task type")
	}
}

func (p *taskProcessorImpl) emitDLQSizeMetricsLoop() {
	timer := time.NewTimer(backoff.JitDuration(
		dlqMetricsEmitTimerInterval,
		dlqMetricsEmitTimerCoefficient,
	))
	staticRequest := &persistence.GetReplicationDLQSizeRequest{
		SourceClusterName: p.sourceCluster,
	}
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			resp, err := p.shard.GetExecutionManager().GetReplicationDLQSize(context.Background(), staticRequest)
			timer.Reset(backoff.JitDuration(
				dlqMetricsEmitTimerInterval,
				dlqMetricsEmitTimerCoefficient,
			))
			if err != nil {
				p.logger.Error("failed to get replication DLQ size", tag.Error(err))
				p.metricsClient.Scope(metrics.ReplicationDLQStatsScope).IncCounter(metrics.ReplicationDLQProbeFailed)
			} else {
				p.metricsClient.Scope(
					metrics.ReplicationDLQStatsScope,
					metrics.InstanceTag(strconv.Itoa(p.shard.GetShardID())),
				).UpdateGauge(metrics.ReplicationDLQSize, float64(resp.Size))
			}
		case <-p.done:
			return
		}
	}
}

func isTransientRetryableError(err error) bool {
	switch err.(type) {
	case *shared.BadRequestError:
		return false
	case *shared.ServiceBusyError:
		return false
	default:
		return true
	}
}

func (p *taskProcessorImpl) shouldRetryDLQ(err error) bool {
	if err == nil {
		return false
	}

	select {
	case <-p.done:
		p.logger.Debug("ReplicationTaskProcessor shutting down.")
		return false
	default:
		return true
	}
}

func (p *taskProcessorImpl) updateFailureMetric(scope int, err error) {
	// Always update failure counter for all replicator errors
	p.metricsClient.IncCounter(scope, metrics.ReplicatorFailures)

	// Also update counter to distinguish between type of failures
	switch err := err.(type) {
	case *h.ShardOwnershipLostError:
		p.metricsClient.IncCounter(scope, metrics.CadenceErrShardOwnershipLostCounter)
	case *shared.BadRequestError:
		p.metricsClient.IncCounter(scope, metrics.CadenceErrBadRequestCounter)
	case *shared.DomainNotActiveError:
		p.metricsClient.IncCounter(scope, metrics.CadenceErrDomainNotActiveCounter)
	case *shared.WorkflowExecutionAlreadyStartedError:
		p.metricsClient.IncCounter(scope, metrics.CadenceErrExecutionAlreadyStartedCounter)
	case *shared.EntityNotExistsError:
		p.metricsClient.IncCounter(scope, metrics.CadenceErrEntityNotExistsCounter)
	case *shared.LimitExceededError:
		p.metricsClient.IncCounter(scope, metrics.CadenceErrLimitExceededCounter)
	case *yarpcerrors.Status:
		if err.Code() == yarpcerrors.CodeDeadlineExceeded {
			p.metricsClient.IncCounter(scope, metrics.CadenceErrContextTimeoutCounter)
		}
	}
}

func (p *taskProcessorImpl) taskProcessingStartWait() {
	shardID := p.shard.GetShardID()
	time.Sleep(backoff.JitDuration(
		p.config.ReplicationTaskProcessorStartWait(shardID),
		p.config.ReplicationTaskProcessorStartWaitJitterCoefficient(shardID),
	))
}
