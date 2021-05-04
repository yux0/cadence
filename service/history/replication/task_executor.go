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

//go:generate mockgen -package $GOPACKAGE -source $GOFILE -destination task_executor_mock.go

package replication

import (
	"context"

	"github.com/uber/cadence/.gen/go/history"
	r "github.com/uber/cadence/.gen/go/replicator"
	"github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/ndc"
	"github.com/uber/cadence/service/history/engine"
	"github.com/uber/cadence/service/history/shard"
)

type (
	// TaskExecutor is the executor for replication task
	TaskExecutor interface {
		execute(replicationTask *r.ReplicationTask, forceApply bool) (int, error)
	}

	taskExecutorImpl struct {
		currentCluster  string
		shard           shard.Context
		domainCache     cache.DomainCache
		historyResender ndc.HistoryResender
		historyEngine   engine.Engine

		metricsClient metrics.Client
		logger        log.Logger
	}
)

var _ TaskExecutor = (*taskExecutorImpl)(nil)

// NewTaskExecutor creates an replication task executor
// The executor uses by 1) DLQ replication task handler 2) history replication task processor
func NewTaskExecutor(
	shard shard.Context,
	domainCache cache.DomainCache,
	historyResender ndc.HistoryResender,
	historyEngine engine.Engine,
	metricsClient metrics.Client,
	logger log.Logger,
) TaskExecutor {
	return &taskExecutorImpl{
		currentCluster:  shard.GetClusterMetadata().GetCurrentClusterName(),
		shard:           shard,
		domainCache:     domainCache,
		historyResender: historyResender,
		historyEngine:   historyEngine,
		metricsClient:   metricsClient,
		logger:          logger,
	}
}

func (e *taskExecutorImpl) execute(
	replicationTask *r.ReplicationTask,
	forceApply bool,
) (int, error) {

	var err error
	var scope int
	switch replicationTask.GetTaskType() {
	case r.ReplicationTaskTypeSyncActivity:
		scope = metrics.SyncActivityTaskScope
		err = e.handleActivityTask(replicationTask, forceApply)
	case r.ReplicationTaskTypeHistoryV2:
		scope = metrics.HistoryReplicationV2TaskScope
		err = e.handleHistoryReplicationTaskV2(replicationTask, forceApply)
	case r.ReplicationTaskTypeFailoverMarker:
		scope = metrics.FailoverMarkerScope
		err = e.handleFailoverReplicationTask(replicationTask)
	default:
		e.logger.Error("Unknown task type.")
		scope = metrics.ReplicatorScope
		err = ErrUnknownReplicationTask
	}

	return scope, err
}

func (e *taskExecutorImpl) handleActivityTask(
	task *r.ReplicationTask,
	forceApply bool,
) error {

	attr := task.SyncActivityTaskAttributes
	doContinue, err := e.filterTask(attr.GetDomainId(), forceApply)
	if err != nil || !doContinue {
		return err
	}

	replicationStopWatch := e.metricsClient.StartTimer(metrics.SyncActivityTaskScope, metrics.CadenceLatency)
	defer replicationStopWatch.Stop()
	request := &history.SyncActivityRequest{
		DomainId:           attr.DomainId,
		WorkflowId:         attr.WorkflowId,
		RunId:              attr.RunId,
		Version:            attr.Version,
		ScheduledId:        attr.ScheduledId,
		ScheduledTime:      attr.ScheduledTime,
		StartedId:          attr.StartedId,
		StartedTime:        attr.StartedTime,
		LastHeartbeatTime:  attr.LastHeartbeatTime,
		Details:            attr.Details,
		Attempt:            attr.Attempt,
		LastFailureReason:  attr.LastFailureReason,
		LastWorkerIdentity: attr.LastWorkerIdentity,
		VersionHistory:     attr.GetVersionHistory(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), replicationTimeout)
	defer cancel()
	err = e.historyEngine.SyncActivity(ctx, request)
	// Handle resend error
	if retryErr, ok := e.convertRetryTaskV2Error(err); ok {
		e.metricsClient.IncCounter(metrics.HistoryRereplicationByActivityReplicationScope, metrics.CadenceClientRequests)
		stopwatch := e.metricsClient.StartTimer(metrics.HistoryRereplicationByActivityReplicationScope, metrics.CadenceClientLatency)
		defer stopwatch.Stop()

		resendErr := e.historyResender.SendSingleWorkflowHistory(
			retryErr.GetDomainId(),
			retryErr.GetWorkflowId(),
			retryErr.GetRunId(),
			retryErr.StartEventId,
			retryErr.StartEventVersion,
			retryErr.EndEventId,
			retryErr.EndEventVersion,
		)
		switch {
		case resendErr == nil:
			break
		case resendErr == ndc.ErrSkipTask:
			e.logger.Error(
				"skip replication sync activity task",
				tag.WorkflowDomainID(retryErr.GetDomainId()),
				tag.WorkflowID(retryErr.GetWorkflowId()),
				tag.WorkflowRunID(retryErr.GetRunId()),
			)
			return nil
		default:
			e.logger.Error(
				"error resend history for sync activity",
				tag.WorkflowDomainID(retryErr.GetDomainId()),
				tag.WorkflowID(retryErr.GetWorkflowId()),
				tag.WorkflowRunID(retryErr.GetRunId()),
				tag.Error(resendErr),
			)
			// should return the replication error, not the resending error
			return err
		}
	}
	// should try again after back fill the history
	return e.historyEngine.SyncActivity(ctx, request)
}

func (e *taskExecutorImpl) handleHistoryReplicationTaskV2(
	task *r.ReplicationTask,
	forceApply bool,
) error {

	attr := task.HistoryTaskV2Attributes
	doContinue, err := e.filterTask(attr.GetDomainId(), forceApply)
	if err != nil || !doContinue {
		return err
	}

	replicationStopWatch := e.metricsClient.StartTimer(metrics.HistoryReplicationV2TaskScope, metrics.CadenceLatency)
	defer replicationStopWatch.Stop()
	request := &history.ReplicateEventsV2Request{
		DomainUUID: attr.DomainId,
		WorkflowExecution: &shared.WorkflowExecution{
			WorkflowId: attr.WorkflowId,
			RunId:      attr.RunId,
		},
		VersionHistoryItems: attr.VersionHistoryItems,
		Events:              attr.Events,
		// new run events does not need version history since there is no prior events
		NewRunEvents: attr.NewRunEvents,
	}
	ctx, cancel := context.WithTimeout(context.Background(), replicationTimeout)
	defer cancel()

	err = e.historyEngine.ReplicateEventsV2(ctx, request)
	retryErr, ok := e.convertRetryTaskV2Error(err)
	if !ok {
		return err
	}
	e.metricsClient.IncCounter(metrics.HistoryRereplicationByHistoryReplicationScope, metrics.CadenceClientRequests)
	resendStopWatch := e.metricsClient.StartTimer(metrics.HistoryRereplicationByHistoryReplicationScope, metrics.CadenceClientLatency)
	defer resendStopWatch.Stop()

	resendErr := e.historyResender.SendSingleWorkflowHistory(
		retryErr.GetDomainId(),
		retryErr.GetWorkflowId(),
		retryErr.GetRunId(),
		retryErr.StartEventId,
		retryErr.StartEventVersion,
		retryErr.EndEventId,
		retryErr.EndEventVersion,
	)
	switch {
	case resendErr == nil:
		break
	case resendErr == ndc.ErrSkipTask:
		e.logger.Error(
			"skip replication history task",
			tag.WorkflowDomainID(retryErr.GetDomainId()),
			tag.WorkflowID(retryErr.GetWorkflowId()),
			tag.WorkflowRunID(retryErr.GetRunId()),
		)
		return nil
	default:
		e.logger.Error(
			"error resend history for history event v2",
			tag.WorkflowDomainID(retryErr.GetDomainId()),
			tag.WorkflowID(retryErr.GetWorkflowId()),
			tag.WorkflowRunID(retryErr.GetRunId()),
			tag.Error(resendErr),
		)
		// should return the replication error, not the resending error
		return err
	}

	return e.historyEngine.ReplicateEventsV2(ctx, request)
}

func (e *taskExecutorImpl) handleFailoverReplicationTask(
	task *r.ReplicationTask,
) error {
	failoverAttributes := task.GetFailoverMarkerAttributes()
	failoverAttributes.CreationTime = task.CreationTime
	return e.shard.AddingPendingFailoverMarker(failoverAttributes)
}

func (e *taskExecutorImpl) filterTask(
	domainID string,
	forceApply bool,
) (bool, error) {

	if forceApply {
		return true, nil
	}

	domainEntry, err := e.domainCache.GetDomainByID(domainID)
	if err != nil {
		return false, err
	}

	shouldProcessTask := false
FilterLoop:
	for _, targetCluster := range domainEntry.GetReplicationConfig().Clusters {
		if e.currentCluster == targetCluster.ClusterName {
			shouldProcessTask = true
			break FilterLoop
		}
	}
	return shouldProcessTask, nil
}

func (e *taskExecutorImpl) convertRetryTaskV2Error(
	err error,
) (*shared.RetryTaskV2Error, bool) {

	retError, ok := err.(*shared.RetryTaskV2Error)
	return retError, ok
}
