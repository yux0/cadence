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

package task

import (
	"time"

	workflow "github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/ndc"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/service/history/config"
	"github.com/uber/cadence/service/history/execution"
	"github.com/uber/cadence/service/history/shard"
	"github.com/uber/cadence/service/worker/archiver"
)

type (
	transferStandbyTaskExecutor struct {
		*transferTaskExecutorBase

		clusterName     string
		historyResender ndc.HistoryResender
	}
)

// NewTransferStandbyTaskExecutor creates a new task executor for standby transfer task
func NewTransferStandbyTaskExecutor(
	shard shard.Context,
	archiverClient archiver.Client,
	executionCache *execution.Cache,
	historyResender ndc.HistoryResender,
	logger log.Logger,
	metricsClient metrics.Client,
	clusterName string,
	config *config.Config,
) Executor {
	return &transferStandbyTaskExecutor{
		transferTaskExecutorBase: newTransferTaskExecutorBase(
			shard,
			archiverClient,
			executionCache,
			logger,
			metricsClient,
			config,
		),
		clusterName:     clusterName,
		historyResender: historyResender,
	}
}

func (t *transferStandbyTaskExecutor) Execute(
	taskInfo Info,
	shouldProcessTask bool,
) error {

	transferTask, ok := taskInfo.(*persistence.TransferTaskInfo)
	if !ok {
		return errUnexpectedTask
	}

	if !shouldProcessTask &&
		transferTask.TaskType != persistence.TransferTaskTypeCloseExecution {
		// guarantee the processing of workflow execution close
		return nil
	}

	switch transferTask.TaskType {
	case persistence.TransferTaskTypeActivityTask:
		return t.processActivityTask(transferTask)
	case persistence.TransferTaskTypeDecisionTask:
		return t.processDecisionTask(transferTask)
	case persistence.TransferTaskTypeCloseExecution:
		return t.processCloseExecution(transferTask)
	case persistence.TransferTaskTypeCancelExecution:
		return t.processCancelExecution(transferTask)
	case persistence.TransferTaskTypeSignalExecution:
		return t.processSignalExecution(transferTask)
	case persistence.TransferTaskTypeStartChildExecution:
		return t.processStartChildExecution(transferTask)
	case persistence.TransferTaskTypeRecordWorkflowStarted:
		return t.processRecordWorkflowStarted(transferTask)
	case persistence.TransferTaskTypeResetWorkflow:
		// no reset needed for standby
		// TODO: add error logs
		return nil
	case persistence.TransferTaskTypeUpsertWorkflowSearchAttributes:
		return t.processUpsertWorkflowSearchAttributes(transferTask)
	default:
		return errUnknownTransferTask
	}
}

func (t *transferStandbyTaskExecutor) processActivityTask(
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	actionFn := func(context execution.Context, mutableState execution.MutableState) (interface{}, error) {

		activityInfo, ok := mutableState.GetActivityInfo(transferTask.ScheduleID)
		if !ok {
			return nil, nil
		}

		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, activityInfo.Version, transferTask.Version, transferTask)
		if err != nil || !ok {
			return nil, err
		}

		if activityInfo.StartedID == common.EmptyEventID {
			return newPushActivityToMatchingInfo(
				activityInfo.ScheduleToStartTimeout,
			), nil
		}

		return nil, nil
	}

	return t.processTransfer(
		processTaskIfClosed,
		transferTask,
		actionFn,
		getStandbyPostActionFn(
			transferTask,
			t.getCurrentTime,
			t.config.StandbyTaskMissingEventsResendDelay(),
			t.config.StandbyTaskMissingEventsDiscardDelay(),
			t.pushActivity,
			t.pushActivity,
		),
	)
}

func (t *transferStandbyTaskExecutor) processDecisionTask(
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	actionFn := func(context execution.Context, mutableState execution.MutableState) (interface{}, error) {

		decisionInfo, ok := mutableState.GetDecisionInfo(transferTask.ScheduleID)
		if !ok {
			return nil, nil
		}

		executionInfo := mutableState.GetExecutionInfo()
		workflowTimeout := executionInfo.WorkflowTimeout
		decisionTimeout := common.MinInt32(workflowTimeout, common.MaxTaskTimeout)

		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, decisionInfo.Version, transferTask.Version, transferTask)
		if err != nil || !ok {
			return nil, err
		}

		if decisionInfo.StartedID == common.EmptyEventID {
			return newPushDecisionToMatchingInfo(
				decisionTimeout,
				workflow.TaskList{Name: &transferTask.TaskList},
			), nil
		}

		return nil, nil
	}

	return t.processTransfer(
		processTaskIfClosed,
		transferTask,
		actionFn,
		getStandbyPostActionFn(
			transferTask,
			t.getCurrentTime,
			t.config.StandbyTaskMissingEventsResendDelay(),
			t.config.StandbyTaskMissingEventsDiscardDelay(),
			t.pushDecision,
			t.pushDecision,
		),
	)
}

func (t *transferStandbyTaskExecutor) processCloseExecution(
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := true
	actionFn := func(context execution.Context, mutableState execution.MutableState) (interface{}, error) {

		if mutableState.IsWorkflowExecutionRunning() {
			// this can happen if workflow is reset.
			return nil, nil
		}

		completionEvent, err := mutableState.GetCompletionEvent()
		if err != nil {
			return nil, err
		}
		wfCloseTime := completionEvent.GetTimestamp()

		executionInfo := mutableState.GetExecutionInfo()
		workflowTypeName := executionInfo.WorkflowTypeName
		workflowCloseTimestamp := wfCloseTime
		workflowCloseStatus := persistence.ToThriftWorkflowExecutionCloseStatus(executionInfo.CloseStatus)
		workflowHistoryLength := mutableState.GetNextEventID() - 1
		startEvent, err := mutableState.GetStartEvent()
		if err != nil {
			return nil, err
		}
		workflowStartTimestamp := startEvent.GetTimestamp()
		workflowExecutionTimestamp := getWorkflowExecutionTimestamp(mutableState, startEvent)
		visibilityMemo := getWorkflowMemo(executionInfo.Memo)
		searchAttr := executionInfo.SearchAttributes

		lastWriteVersion, err := mutableState.GetLastWriteVersion()
		if err != nil {
			return nil, err
		}
		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, lastWriteVersion, transferTask.Version, transferTask)
		if err != nil || !ok {
			return nil, err
		}

		// DO NOT REPLY TO PARENT
		// since event replication should be done by active cluster
		return nil, t.recordWorkflowClosed(
			transferTask.DomainID,
			transferTask.WorkflowID,
			transferTask.RunID,
			workflowTypeName,
			workflowStartTimestamp,
			workflowExecutionTimestamp.UnixNano(),
			workflowCloseTimestamp,
			workflowCloseStatus,
			workflowHistoryLength,
			transferTask.GetTaskID(),
			visibilityMemo,
			executionInfo.TaskList,
			searchAttr,
		)
	}

	return t.processTransfer(
		processTaskIfClosed,
		transferTask,
		actionFn,
		standbyTaskPostActionNoOp,
	) // no op post action, since the entire workflow is finished
}

func (t *transferStandbyTaskExecutor) processCancelExecution(
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	actionFn := func(context execution.Context, mutableState execution.MutableState) (interface{}, error) {

		requestCancelInfo, ok := mutableState.GetRequestCancelInfo(transferTask.ScheduleID)
		if !ok {
			return nil, nil
		}

		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, requestCancelInfo.Version, transferTask.Version, transferTask)
		if err != nil || !ok {
			return nil, err
		}

		return getHistoryResendInfo(mutableState)
	}

	return t.processTransfer(
		processTaskIfClosed,
		transferTask,
		actionFn,
		getStandbyPostActionFn(
			transferTask,
			t.getCurrentTime,
			t.config.StandbyTaskMissingEventsResendDelay(),
			t.config.StandbyTaskMissingEventsDiscardDelay(),
			t.fetchHistoryFromRemote,
			standbyTransferTaskPostActionTaskDiscarded,
		),
	)
}

func (t *transferStandbyTaskExecutor) processSignalExecution(
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	actionFn := func(context execution.Context, mutableState execution.MutableState) (interface{}, error) {

		signalInfo, ok := mutableState.GetSignalInfo(transferTask.ScheduleID)
		if !ok {
			return nil, nil
		}

		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, signalInfo.Version, transferTask.Version, transferTask)
		if err != nil || !ok {
			return nil, err
		}

		return getHistoryResendInfo(mutableState)
	}

	return t.processTransfer(
		processTaskIfClosed,
		transferTask,
		actionFn,
		getStandbyPostActionFn(
			transferTask,
			t.getCurrentTime,
			t.config.StandbyTaskMissingEventsResendDelay(),
			t.config.StandbyTaskMissingEventsDiscardDelay(),
			t.fetchHistoryFromRemote,
			standbyTransferTaskPostActionTaskDiscarded,
		),
	)
}

func (t *transferStandbyTaskExecutor) processStartChildExecution(
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	actionFn := func(context execution.Context, mutableState execution.MutableState) (interface{}, error) {

		childWorkflowInfo, ok := mutableState.GetChildExecutionInfo(transferTask.ScheduleID)
		if !ok {
			return nil, nil
		}

		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, childWorkflowInfo.Version, transferTask.Version, transferTask)
		if err != nil || !ok {
			return nil, err
		}

		if childWorkflowInfo.StartedID != common.EmptyEventID {
			return nil, nil
		}

		return getHistoryResendInfo(mutableState)
	}

	return t.processTransfer(
		processTaskIfClosed,
		transferTask,
		actionFn,
		getStandbyPostActionFn(
			transferTask,
			t.getCurrentTime,
			t.config.StandbyTaskMissingEventsResendDelay(),
			t.config.StandbyTaskMissingEventsDiscardDelay(),
			t.fetchHistoryFromRemote,
			standbyTransferTaskPostActionTaskDiscarded,
		),
	)
}

func (t *transferStandbyTaskExecutor) processRecordWorkflowStarted(
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	return t.processTransfer(
		processTaskIfClosed,
		transferTask,
		func(context execution.Context, mutableState execution.MutableState) (interface{}, error) {
			return nil, t.processRecordWorkflowStartedOrUpsertHelper(transferTask, mutableState, true)
		},
		standbyTaskPostActionNoOp,
	)
}

func (t *transferStandbyTaskExecutor) processUpsertWorkflowSearchAttributes(
	transferTask *persistence.TransferTaskInfo,
) error {

	processTaskIfClosed := false
	return t.processTransfer(
		processTaskIfClosed,
		transferTask,
		func(context execution.Context, mutableState execution.MutableState) (interface{}, error) {
			return nil, t.processRecordWorkflowStartedOrUpsertHelper(transferTask, mutableState, false)
		},
		standbyTaskPostActionNoOp,
	)
}

func (t *transferStandbyTaskExecutor) processRecordWorkflowStartedOrUpsertHelper(
	transferTask *persistence.TransferTaskInfo,
	mutableState execution.MutableState,
	isRecordStart bool,
) error {

	// verify task version for RecordWorkflowStarted.
	// upsert doesn't require verifyTask, because it is just a sync of mutableState.
	if isRecordStart {
		startVersion, err := mutableState.GetStartVersion()
		if err != nil {
			return err
		}
		ok, err := verifyTaskVersion(t.shard, t.logger, transferTask.DomainID, startVersion, transferTask.Version, transferTask)
		if err != nil || !ok {
			return err
		}
	}

	executionInfo := mutableState.GetExecutionInfo()
	workflowTimeout := executionInfo.WorkflowTimeout
	wfTypeName := executionInfo.WorkflowTypeName
	startEvent, err := mutableState.GetStartEvent()
	if err != nil {
		return err
	}
	startTimestamp := startEvent.GetTimestamp()
	executionTimestamp := getWorkflowExecutionTimestamp(mutableState, startEvent)
	visibilityMemo := getWorkflowMemo(executionInfo.Memo)
	searchAttr := copySearchAttributes(executionInfo.SearchAttributes)

	if isRecordStart {
		return t.recordWorkflowStarted(
			transferTask.DomainID,
			transferTask.WorkflowID,
			transferTask.RunID,
			wfTypeName,
			startTimestamp,
			executionTimestamp.UnixNano(),
			workflowTimeout,
			transferTask.GetTaskID(),
			executionInfo.TaskList,
			visibilityMemo,
			searchAttr,
		)
	}
	return t.upsertWorkflowExecution(
		transferTask.DomainID,
		transferTask.WorkflowID,
		transferTask.RunID,
		wfTypeName,
		startTimestamp,
		executionTimestamp.UnixNano(),
		workflowTimeout,
		transferTask.GetTaskID(),
		executionInfo.TaskList,
		visibilityMemo,
		searchAttr,
	)

}

func (t *transferStandbyTaskExecutor) processTransfer(
	processTaskIfClosed bool,
	taskInfo Info,
	actionFn standbyActionFn,
	postActionFn standbyPostActionFn,
) (retError error) {

	transferTask := taskInfo.(*persistence.TransferTaskInfo)
	context, release, err := t.executionCache.GetOrCreateWorkflowExecutionWithTimeout(
		transferTask.DomainID,
		getWorkflowExecution(transferTask),
		taskDefaultTimeout,
	)
	if err != nil {
		return err
	}
	defer func() {
		if retError == ErrTaskRedispatch {
			release(nil)
		} else {
			release(retError)
		}
	}()

	mutableState, err := loadMutableStateForTransferTask(context, transferTask, t.metricsClient, t.logger)
	if err != nil || mutableState == nil {
		return err
	}

	if !mutableState.IsWorkflowExecutionRunning() && !processTaskIfClosed {
		// workflow already finished, no need to process the timer
		return nil
	}

	historyResendInfo, err := actionFn(context, mutableState)
	if err != nil {
		return err
	}

	release(nil)
	return postActionFn(taskInfo, historyResendInfo, t.logger)
}

func (t *transferStandbyTaskExecutor) pushActivity(
	task Info,
	postActionInfo interface{},
	logger log.Logger,
) error {

	if postActionInfo == nil {
		return nil
	}

	pushActivityInfo := postActionInfo.(*pushActivityToMatchingInfo)
	timeout := common.MinInt32(pushActivityInfo.activityScheduleToStartTimeout, common.MaxTaskTimeout)
	return t.transferTaskExecutorBase.pushActivity(
		task.(*persistence.TransferTaskInfo),
		timeout,
	)
}

func (t *transferStandbyTaskExecutor) pushDecision(
	task Info,
	postActionInfo interface{},
	logger log.Logger,
) error {

	if postActionInfo == nil {
		return nil
	}

	pushDecisionInfo := postActionInfo.(*pushDecisionToMatchingInfo)
	timeout := common.MinInt32(pushDecisionInfo.decisionScheduleToStartTimeout, common.MaxTaskTimeout)
	return t.transferTaskExecutorBase.pushDecision(
		task.(*persistence.TransferTaskInfo),
		&pushDecisionInfo.tasklist,
		timeout,
	)
}

func (t *transferStandbyTaskExecutor) fetchHistoryFromRemote(
	taskInfo Info,
	postActionInfo interface{},
	log log.Logger,
) error {

	if postActionInfo == nil {
		return nil
	}

	transferTask := taskInfo.(*persistence.TransferTaskInfo)
	resendInfo := postActionInfo.(*historyResendInfo)

	t.metricsClient.IncCounter(metrics.HistoryRereplicationByTransferTaskScope, metrics.CadenceClientRequests)
	stopwatch := t.metricsClient.StartTimer(metrics.HistoryRereplicationByTransferTaskScope, metrics.CadenceClientLatency)
	defer stopwatch.Stop()

	var err error
	if resendInfo.lastEventID != nil && resendInfo.lastEventVersion != nil {
		err = t.historyResender.SendSingleWorkflowHistory(
			transferTask.DomainID,
			transferTask.WorkflowID,
			transferTask.RunID,
			resendInfo.lastEventID,
			resendInfo.lastEventVersion,
			nil,
			nil,
		)
	} else {
		err = &workflow.InternalServiceError{
			Message: "transferQueueStandbyProcessor encounter empty historyResendInfo",
		}
	}

	if err != nil {
		t.logger.Error("Error re-replicating history from remote.",
			tag.ShardID(t.shard.GetShardID()),
			tag.WorkflowDomainID(transferTask.DomainID),
			tag.WorkflowID(transferTask.WorkflowID),
			tag.WorkflowRunID(transferTask.RunID),
			tag.SourceCluster(t.clusterName))
	}

	// return error so task processing logic will retry
	return ErrTaskRedispatch
}

func (t *transferStandbyTaskExecutor) getCurrentTime() time.Time {
	return t.shard.GetCurrentTime(t.clusterName)
}
