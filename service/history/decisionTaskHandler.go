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

package history

import (
	"context"
	"fmt"

	"github.com/pborman/uuid"

	workflow "github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/backoff"
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/service/history/config"
	"github.com/uber/cadence/service/history/execution"
)

type (
	decisionAttrValidationFn func() error

	decisionTaskHandlerImpl struct {
		identity                string
		decisionTaskCompletedID int64
		domainEntry             *cache.DomainCacheEntry

		// internal state
		hasUnhandledEventsBeforeDecisions bool
		failDecision                      bool
		failDecisionCause                 *workflow.DecisionTaskFailedCause
		failMessage                       *string
		activityNotStartedCancelled       bool
		continueAsNewBuilder              execution.MutableState
		stopProcessing                    bool // should stop processing any more decisions
		mutableState                      execution.MutableState

		// validation
		attrValidator    *decisionAttrValidator
		sizeLimitChecker *workflowSizeChecker

		tokenSerializer common.TaskTokenSerializer

		logger        log.Logger
		domainCache   cache.DomainCache
		metricsClient metrics.Client
		config        *config.Config
	}

	decisionResult struct {
		activityDispatchInfo *workflow.ActivityLocalDispatchInfo
	}
)

func newDecisionTaskHandler(
	identity string,
	decisionTaskCompletedID int64,
	domainEntry *cache.DomainCacheEntry,
	mutableState execution.MutableState,
	attrValidator *decisionAttrValidator,
	sizeLimitChecker *workflowSizeChecker,
	tokenSerializer common.TaskTokenSerializer,
	logger log.Logger,
	domainCache cache.DomainCache,
	metricsClient metrics.Client,
	config *config.Config,
) *decisionTaskHandlerImpl {

	return &decisionTaskHandlerImpl{
		identity:                identity,
		decisionTaskCompletedID: decisionTaskCompletedID,
		domainEntry:             domainEntry,

		// internal state
		hasUnhandledEventsBeforeDecisions: mutableState.HasBufferedEvents(),
		failDecision:                      false,
		failDecisionCause:                 nil,
		failMessage:                       nil,
		activityNotStartedCancelled:       false,
		continueAsNewBuilder:              nil,
		stopProcessing:                    false,
		mutableState:                      mutableState,

		// validation
		attrValidator:    attrValidator,
		sizeLimitChecker: sizeLimitChecker,

		tokenSerializer: tokenSerializer,

		logger:        logger,
		domainCache:   domainCache,
		metricsClient: metricsClient,
		config:        config,
	}
}

func (handler *decisionTaskHandlerImpl) handleDecisions(
	ctx context.Context,
	executionContext []byte,
	decisions []*workflow.Decision,
) ([]*decisionResult, error) {

	// overall workflow size / count check
	failWorkflow, err := handler.sizeLimitChecker.failWorkflowSizeExceedsLimit()
	if err != nil || failWorkflow {
		return nil, err
	}

	var results []*decisionResult
	for _, decision := range decisions {

		result, err := handler.handleDecisionWithResult(ctx, decision)
		if err != nil || handler.stopProcessing {
			return nil, err
		} else if result != nil {
			results = append(results, result)
		}

	}
	handler.mutableState.GetExecutionInfo().ExecutionContext = executionContext
	return results, nil
}

func (handler *decisionTaskHandlerImpl) handleDecisionWithResult(
	ctx context.Context,
	decision *workflow.Decision,
) (*decisionResult, error) {
	switch decision.GetDecisionType() {
	case workflow.DecisionTypeScheduleActivityTask:
		return handler.handleDecisionScheduleActivity(ctx, decision.ScheduleActivityTaskDecisionAttributes)
	default:
		return nil, handler.handleDecision(ctx, decision)
	}
}

func (handler *decisionTaskHandlerImpl) handleDecision(
	ctx context.Context,
	decision *workflow.Decision,
) error {
	switch decision.GetDecisionType() {

	case workflow.DecisionTypeCompleteWorkflowExecution:
		return handler.handleDecisionCompleteWorkflow(ctx, decision.CompleteWorkflowExecutionDecisionAttributes)

	case workflow.DecisionTypeFailWorkflowExecution:
		return handler.handleDecisionFailWorkflow(ctx, decision.FailWorkflowExecutionDecisionAttributes)

	case workflow.DecisionTypeCancelWorkflowExecution:
		return handler.handleDecisionCancelWorkflow(ctx, decision.CancelWorkflowExecutionDecisionAttributes)

	case workflow.DecisionTypeStartTimer:
		return handler.handleDecisionStartTimer(ctx, decision.StartTimerDecisionAttributes)

	case workflow.DecisionTypeRequestCancelActivityTask:
		return handler.handleDecisionRequestCancelActivity(ctx, decision.RequestCancelActivityTaskDecisionAttributes)

	case workflow.DecisionTypeCancelTimer:
		return handler.handleDecisionCancelTimer(ctx, decision.CancelTimerDecisionAttributes)

	case workflow.DecisionTypeRecordMarker:
		return handler.handleDecisionRecordMarker(ctx, decision.RecordMarkerDecisionAttributes)

	case workflow.DecisionTypeRequestCancelExternalWorkflowExecution:
		return handler.handleDecisionRequestCancelExternalWorkflow(ctx, decision.RequestCancelExternalWorkflowExecutionDecisionAttributes)

	case workflow.DecisionTypeSignalExternalWorkflowExecution:
		return handler.handleDecisionSignalExternalWorkflow(ctx, decision.SignalExternalWorkflowExecutionDecisionAttributes)

	case workflow.DecisionTypeContinueAsNewWorkflowExecution:
		return handler.handleDecisionContinueAsNewWorkflow(ctx, decision.ContinueAsNewWorkflowExecutionDecisionAttributes)

	case workflow.DecisionTypeStartChildWorkflowExecution:
		return handler.handleDecisionStartChildWorkflow(ctx, decision.StartChildWorkflowExecutionDecisionAttributes)

	case workflow.DecisionTypeUpsertWorkflowSearchAttributes:
		return handler.handleDecisionUpsertWorkflowSearchAttributes(ctx, decision.UpsertWorkflowSearchAttributesDecisionAttributes)

	default:
		return &workflow.BadRequestError{Message: fmt.Sprintf("Unknown decision type: %v", decision.GetDecisionType())}
	}
}

func (handler *decisionTaskHandlerImpl) handleDecisionScheduleActivity(
	ctx context.Context,
	attr *workflow.ScheduleActivityTaskDecisionAttributes,
) (*decisionResult, error) {

	handler.metricsClient.IncCounter(
		metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeScheduleActivityCounter,
	)

	executionInfo := handler.mutableState.GetExecutionInfo()
	domainID := executionInfo.DomainID
	targetDomainID := domainID
	if attr.GetDomain() != "" {
		targetDomainEntry, err := handler.domainCache.GetDomain(attr.GetDomain())
		if err != nil {
			return nil, &workflow.InternalServiceError{
				Message: fmt.Sprintf("Unable to schedule activity across domain %v.", attr.GetDomain()),
			}
		}
		targetDomainID = targetDomainEntry.GetInfo().ID
	}

	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateActivityScheduleAttributes(
				domainID,
				targetDomainID,
				attr,
				executionInfo.WorkflowTimeout,
			)
		},
		workflow.DecisionTaskFailedCauseBadScheduleActivityAttributes,
	); err != nil || handler.stopProcessing {
		return nil, err
	}

	failWorkflow, err := handler.sizeLimitChecker.failWorkflowIfBlobSizeExceedsLimit(
		metrics.DecisionTypeTag(workflow.DecisionTypeScheduleActivityTask.String()),
		attr.Input,
		"ScheduleActivityTaskDecisionAttributes.Input exceeds size limit.",
	)
	if err != nil || failWorkflow {
		handler.stopProcessing = true
		return nil, err
	}

	event, ai, activityDispatchInfo, err := handler.mutableState.AddActivityTaskScheduledEvent(handler.decisionTaskCompletedID, attr)
	switch err.(type) {
	case nil:
		if activityDispatchInfo != nil {
			if _, err1 := handler.mutableState.AddActivityTaskStartedEvent(ai, event.GetEventId(), uuid.New(), handler.identity); err1 != nil {
				return nil, err1
			}
			token := &common.TaskToken{
				DomainID:        executionInfo.DomainID,
				WorkflowID:      executionInfo.WorkflowID,
				WorkflowType:    executionInfo.WorkflowTypeName,
				RunID:           executionInfo.RunID,
				ScheduleID:      ai.ScheduleID,
				ScheduleAttempt: 0,
				ActivityID:      ai.ActivityID,
				ActivityType:    attr.ActivityType.GetName(),
			}
			activityDispatchInfo.TaskToken, err = handler.tokenSerializer.Serialize(token)
			if err != nil {
				return nil, ErrSerializingToken
			}
			activityDispatchInfo.ScheduledTimestamp = common.Int64Ptr(ai.ScheduledTime.UnixNano())
			activityDispatchInfo.ScheduledTimestampOfThisAttempt = common.Int64Ptr(ai.ScheduledTime.UnixNano())
			activityDispatchInfo.StartedTimestamp = common.Int64Ptr(ai.StartedTime.UnixNano())
			return &decisionResult{activityDispatchInfo: activityDispatchInfo}, nil
		}
		return nil, nil
	case *workflow.BadRequestError:
		return nil, handler.handlerFailDecision(
			workflow.DecisionTaskFailedCauseScheduleActivityDuplicateID, "",
		)
	default:
		return nil, err
	}
}

func (handler *decisionTaskHandlerImpl) handleDecisionRequestCancelActivity(
	ctx context.Context,
	attr *workflow.RequestCancelActivityTaskDecisionAttributes,
) error {

	handler.metricsClient.IncCounter(
		metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeCancelActivityCounter,
	)

	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateActivityCancelAttributes(attr)
		},
		workflow.DecisionTaskFailedCauseBadRequestCancelActivityAttributes,
	); err != nil || handler.stopProcessing {
		return err
	}

	activityID := attr.GetActivityId()
	actCancelReqEvent, ai, err := handler.mutableState.AddActivityTaskCancelRequestedEvent(
		handler.decisionTaskCompletedID,
		activityID,
		handler.identity,
	)
	switch err.(type) {
	case nil:
		if ai.StartedID == common.EmptyEventID {
			// We haven't started the activity yet, we can cancel the activity right away and
			// schedule a decision task to ensure the workflow makes progress.
			_, err = handler.mutableState.AddActivityTaskCanceledEvent(
				ai.ScheduleID,
				ai.StartedID,
				actCancelReqEvent.GetEventId(),
				[]byte(activityCancellationMsgActivityNotStarted),
				handler.identity,
			)
			if err != nil {
				return err
			}
			handler.activityNotStartedCancelled = true
		}
		return nil
	case *workflow.BadRequestError:
		_, err = handler.mutableState.AddRequestCancelActivityTaskFailedEvent(
			handler.decisionTaskCompletedID,
			activityID,
			activityCancellationMsgActivityIDUnknown,
		)
		return err
	default:
		return err
	}
}

func (handler *decisionTaskHandlerImpl) handleDecisionStartTimer(
	ctx context.Context,
	attr *workflow.StartTimerDecisionAttributes,
) error {

	handler.metricsClient.IncCounter(
		metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeStartTimerCounter,
	)

	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateTimerScheduleAttributes(attr)
		},
		workflow.DecisionTaskFailedCauseBadStartTimerAttributes,
	); err != nil || handler.stopProcessing {
		return err
	}

	_, _, err := handler.mutableState.AddTimerStartedEvent(handler.decisionTaskCompletedID, attr)
	switch err.(type) {
	case nil:
		return nil
	case *workflow.BadRequestError:
		return handler.handlerFailDecision(
			workflow.DecisionTaskFailedCauseStartTimerDuplicateID, "",
		)
	default:
		return err
	}
}

func (handler *decisionTaskHandlerImpl) handleDecisionCompleteWorkflow(
	ctx context.Context,
	attr *workflow.CompleteWorkflowExecutionDecisionAttributes,
) error {

	handler.metricsClient.IncCounter(
		metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeCompleteWorkflowCounter,
	)

	if handler.hasUnhandledEventsBeforeDecisions {
		return handler.handlerFailDecision(workflow.DecisionTaskFailedCauseUnhandledDecision, "")
	}

	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateCompleteWorkflowExecutionAttributes(attr)
		},
		workflow.DecisionTaskFailedCauseBadCompleteWorkflowExecutionAttributes,
	); err != nil || handler.stopProcessing {
		return err
	}

	failWorkflow, err := handler.sizeLimitChecker.failWorkflowIfBlobSizeExceedsLimit(
		metrics.DecisionTypeTag(workflow.DecisionTypeCompleteWorkflowExecution.String()),
		attr.Result,
		"CompleteWorkflowExecutionDecisionAttributes.Result exceeds size limit.",
	)
	if err != nil || failWorkflow {
		handler.stopProcessing = true
		return err
	}

	// If the decision has more than one completion event than just pick the first one
	if !handler.mutableState.IsWorkflowExecutionRunning() {
		handler.metricsClient.IncCounter(
			metrics.HistoryRespondDecisionTaskCompletedScope,
			metrics.MultipleCompletionDecisionsCounter,
		)
		handler.logger.Warn(
			"Multiple completion decisions",
			tag.WorkflowDecisionType(int64(workflow.DecisionTypeCompleteWorkflowExecution)),
			tag.ErrorTypeMultipleCompletionDecisions,
		)
		return nil
	}

	// check if this is a cron workflow
	cronBackoff, err := handler.mutableState.GetCronBackoffDuration(ctx)
	if err != nil {
		handler.stopProcessing = true
		return err
	}
	if cronBackoff == backoff.NoBackoff {
		// not cron, so complete this workflow execution
		if _, err := handler.mutableState.AddCompletedWorkflowEvent(handler.decisionTaskCompletedID, attr); err != nil {
			return &workflow.InternalServiceError{Message: "Unable to add complete workflow event."}
		}
		return nil
	}

	// this is a cron workflow
	startEvent, err := handler.mutableState.GetStartEvent(ctx)
	if err != nil {
		return err
	}
	startAttributes := startEvent.WorkflowExecutionStartedEventAttributes
	return handler.retryCronContinueAsNew(
		ctx,
		startAttributes,
		int32(cronBackoff.Seconds()),
		workflow.ContinueAsNewInitiatorCronSchedule.Ptr(),
		nil,
		nil,
		attr.Result,
	)
}

func (handler *decisionTaskHandlerImpl) handleDecisionFailWorkflow(
	ctx context.Context,
	attr *workflow.FailWorkflowExecutionDecisionAttributes,
) error {

	handler.metricsClient.IncCounter(
		metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeFailWorkflowCounter,
	)

	if handler.hasUnhandledEventsBeforeDecisions {
		return handler.handlerFailDecision(workflow.DecisionTaskFailedCauseUnhandledDecision, "")
	}

	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateFailWorkflowExecutionAttributes(attr)
		},
		workflow.DecisionTaskFailedCauseBadFailWorkflowExecutionAttributes,
	); err != nil || handler.stopProcessing {
		return err
	}

	failWorkflow, err := handler.sizeLimitChecker.failWorkflowIfBlobSizeExceedsLimit(
		metrics.DecisionTypeTag(workflow.DecisionTypeFailWorkflowExecution.String()),
		attr.Details,
		"FailWorkflowExecutionDecisionAttributes.Details exceeds size limit.",
	)
	if err != nil || failWorkflow {
		handler.stopProcessing = true
		return err
	}

	// If the decision has more than one completion event than just pick the first one
	if !handler.mutableState.IsWorkflowExecutionRunning() {
		handler.metricsClient.IncCounter(
			metrics.HistoryRespondDecisionTaskCompletedScope,
			metrics.MultipleCompletionDecisionsCounter,
		)
		handler.logger.Warn(
			"Multiple completion decisions",
			tag.WorkflowDecisionType(int64(workflow.DecisionTypeFailWorkflowExecution)),
			tag.ErrorTypeMultipleCompletionDecisions,
		)
		return nil
	}

	// below will check whether to do continue as new based on backoff & backoff or cron
	backoffInterval := handler.mutableState.GetRetryBackoffDuration(attr.GetReason())
	continueAsNewInitiator := workflow.ContinueAsNewInitiatorRetryPolicy
	// first check the backoff retry
	if backoffInterval == backoff.NoBackoff {
		// if no backoff retry, set the backoffInterval using cron schedule
		backoffInterval, err = handler.mutableState.GetCronBackoffDuration(ctx)
		if err != nil {
			handler.stopProcessing = true
			return err
		}
		continueAsNewInitiator = workflow.ContinueAsNewInitiatorCronSchedule
	}
	// second check the backoff / cron schedule
	if backoffInterval == backoff.NoBackoff {
		// no retry or cron
		if _, err := handler.mutableState.AddFailWorkflowEvent(handler.decisionTaskCompletedID, attr); err != nil {
			return err
		}
		return nil
	}

	// this is a cron / backoff workflow
	startEvent, err := handler.mutableState.GetStartEvent(ctx)
	if err != nil {
		return err
	}
	startAttributes := startEvent.WorkflowExecutionStartedEventAttributes
	return handler.retryCronContinueAsNew(
		ctx,
		startAttributes,
		int32(backoffInterval.Seconds()),
		continueAsNewInitiator.Ptr(),
		attr.Reason,
		attr.Details,
		startAttributes.LastCompletionResult,
	)
}

func (handler *decisionTaskHandlerImpl) handleDecisionCancelTimer(
	ctx context.Context,
	attr *workflow.CancelTimerDecisionAttributes,
) error {

	handler.metricsClient.IncCounter(
		metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeCancelTimerCounter,
	)

	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateTimerCancelAttributes(attr)
		},
		workflow.DecisionTaskFailedCauseBadCancelTimerAttributes,
	); err != nil || handler.stopProcessing {
		return err
	}

	_, err := handler.mutableState.AddTimerCanceledEvent(
		handler.decisionTaskCompletedID,
		attr,
		handler.identity)
	switch err.(type) {
	case nil:
		// timer deletion is a success, we may have deleted a fired timer in
		// which case we should reset hasBufferedEvents
		// TODO deletion of timer fired event refreshing hasUnhandledEventsBeforeDecisions
		//  is not entirely correct, since during these decisions processing, new event may appear
		handler.hasUnhandledEventsBeforeDecisions = handler.mutableState.HasBufferedEvents()
		return nil
	case *workflow.BadRequestError:
		_, err = handler.mutableState.AddCancelTimerFailedEvent(
			handler.decisionTaskCompletedID,
			attr,
			handler.identity,
		)
		return err
	default:
		return err
	}
}

func (handler *decisionTaskHandlerImpl) handleDecisionCancelWorkflow(
	ctx context.Context,
	attr *workflow.CancelWorkflowExecutionDecisionAttributes,
) error {

	handler.metricsClient.IncCounter(metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeCancelWorkflowCounter)

	if handler.hasUnhandledEventsBeforeDecisions {
		return handler.handlerFailDecision(workflow.DecisionTaskFailedCauseUnhandledDecision, "")
	}

	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateCancelWorkflowExecutionAttributes(attr)
		},
		workflow.DecisionTaskFailedCauseBadCancelWorkflowExecutionAttributes,
	); err != nil || handler.stopProcessing {
		return err
	}

	// If the decision has more than one completion event than just pick the first one
	if !handler.mutableState.IsWorkflowExecutionRunning() {
		handler.metricsClient.IncCounter(
			metrics.HistoryRespondDecisionTaskCompletedScope,
			metrics.MultipleCompletionDecisionsCounter,
		)
		handler.logger.Warn(
			"Multiple completion decisions",
			tag.WorkflowDecisionType(int64(workflow.DecisionTypeCancelWorkflowExecution)),
			tag.ErrorTypeMultipleCompletionDecisions,
		)
		return nil
	}

	_, err := handler.mutableState.AddWorkflowExecutionCanceledEvent(handler.decisionTaskCompletedID, attr)
	return err
}

func (handler *decisionTaskHandlerImpl) handleDecisionRequestCancelExternalWorkflow(
	ctx context.Context,
	attr *workflow.RequestCancelExternalWorkflowExecutionDecisionAttributes,
) error {

	handler.metricsClient.IncCounter(
		metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeCancelExternalWorkflowCounter,
	)

	executionInfo := handler.mutableState.GetExecutionInfo()
	domainID := executionInfo.DomainID
	targetDomainID := domainID
	if attr.GetDomain() != "" {
		targetDomainEntry, err := handler.domainCache.GetDomain(attr.GetDomain())
		if err != nil {
			return &workflow.InternalServiceError{
				Message: fmt.Sprintf("Unable to cancel workflow across domain: %v.", attr.GetDomain()),
			}
		}
		targetDomainID = targetDomainEntry.GetInfo().ID
	}

	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateCancelExternalWorkflowExecutionAttributes(
				domainID,
				targetDomainID,
				attr,
			)
		},
		workflow.DecisionTaskFailedCauseBadRequestCancelExternalWorkflowExecutionAttributes,
	); err != nil || handler.stopProcessing {
		return err
	}

	cancelRequestID := uuid.New()
	_, _, err := handler.mutableState.AddRequestCancelExternalWorkflowExecutionInitiatedEvent(
		handler.decisionTaskCompletedID, cancelRequestID, attr,
	)
	return err
}

func (handler *decisionTaskHandlerImpl) handleDecisionRecordMarker(
	ctx context.Context,
	attr *workflow.RecordMarkerDecisionAttributes,
) error {

	handler.metricsClient.IncCounter(
		metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeRecordMarkerCounter,
	)

	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateRecordMarkerAttributes(attr)
		},
		workflow.DecisionTaskFailedCauseBadRecordMarkerAttributes,
	); err != nil || handler.stopProcessing {
		return err
	}

	failWorkflow, err := handler.sizeLimitChecker.failWorkflowIfBlobSizeExceedsLimit(
		metrics.DecisionTypeTag(workflow.DecisionTypeRecordMarker.String()),
		attr.Details,
		"RecordMarkerDecisionAttributes.Details exceeds size limit.",
	)
	if err != nil || failWorkflow {
		handler.stopProcessing = true
		return err
	}

	_, err = handler.mutableState.AddRecordMarkerEvent(handler.decisionTaskCompletedID, attr)
	return err
}

func (handler *decisionTaskHandlerImpl) handleDecisionContinueAsNewWorkflow(
	ctx context.Context,
	attr *workflow.ContinueAsNewWorkflowExecutionDecisionAttributes,
) error {

	handler.metricsClient.IncCounter(
		metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeContinueAsNewCounter,
	)

	if handler.hasUnhandledEventsBeforeDecisions {
		return handler.handlerFailDecision(workflow.DecisionTaskFailedCauseUnhandledDecision, "")
	}

	executionInfo := handler.mutableState.GetExecutionInfo()

	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateContinueAsNewWorkflowExecutionAttributes(
				attr,
				executionInfo,
			)
		},
		workflow.DecisionTaskFailedCauseBadContinueAsNewAttributes,
	); err != nil || handler.stopProcessing {
		return err
	}

	failWorkflow, err := handler.sizeLimitChecker.failWorkflowIfBlobSizeExceedsLimit(
		metrics.DecisionTypeTag(workflow.DecisionTypeContinueAsNewWorkflowExecution.String()),
		attr.Input,
		"ContinueAsNewWorkflowExecutionDecisionAttributes. Input exceeds size limit.",
	)
	if err != nil || failWorkflow {
		handler.stopProcessing = true
		return err
	}

	// If the decision has more than one completion event than just pick the first one
	if !handler.mutableState.IsWorkflowExecutionRunning() {
		handler.metricsClient.IncCounter(
			metrics.HistoryRespondDecisionTaskCompletedScope,
			metrics.MultipleCompletionDecisionsCounter,
		)
		handler.logger.Warn(
			"Multiple completion decisions",
			tag.WorkflowDecisionType(int64(workflow.DecisionTypeContinueAsNewWorkflowExecution)),
			tag.ErrorTypeMultipleCompletionDecisions,
		)
		return nil
	}

	// Extract parentDomainName so it can be passed down to next run of workflow execution
	var parentDomainName string
	if handler.mutableState.HasParentExecution() {
		parentDomainID := executionInfo.ParentDomainID
		parentDomainEntry, err := handler.domainCache.GetDomainByID(parentDomainID)
		if err != nil {
			return err
		}
		parentDomainName = parentDomainEntry.GetInfo().Name
	}

	_, newStateBuilder, err := handler.mutableState.AddContinueAsNewEvent(
		ctx,
		handler.decisionTaskCompletedID,
		handler.decisionTaskCompletedID,
		parentDomainName,
		attr,
	)
	if err != nil {
		return err
	}

	handler.continueAsNewBuilder = newStateBuilder
	return nil
}

func (handler *decisionTaskHandlerImpl) handleDecisionStartChildWorkflow(
	ctx context.Context,
	attr *workflow.StartChildWorkflowExecutionDecisionAttributes,
) error {

	handler.metricsClient.IncCounter(
		metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeChildWorkflowCounter,
	)

	executionInfo := handler.mutableState.GetExecutionInfo()
	domainID := executionInfo.DomainID
	targetDomainID := domainID
	if attr.GetDomain() != "" {
		targetDomainEntry, err := handler.domainCache.GetDomain(attr.GetDomain())
		if err != nil {
			return &workflow.InternalServiceError{
				Message: fmt.Sprintf("Unable to schedule child execution across domain %v.", attr.GetDomain()),
			}
		}
		targetDomainID = targetDomainEntry.GetInfo().ID
	}

	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateStartChildExecutionAttributes(
				domainID,
				targetDomainID,
				attr,
				executionInfo,
			)
		},
		workflow.DecisionTaskFailedCauseBadStartChildExecutionAttributes,
	); err != nil || handler.stopProcessing {
		return err
	}

	failWorkflow, err := handler.sizeLimitChecker.failWorkflowIfBlobSizeExceedsLimit(
		metrics.DecisionTypeTag(workflow.DecisionTypeStartChildWorkflowExecution.String()),
		attr.Input,
		"StartChildWorkflowExecutionDecisionAttributes.Input exceeds size limit.",
	)
	if err != nil || failWorkflow {
		handler.stopProcessing = true
		return err
	}

	enabled := handler.config.EnableParentClosePolicy(handler.domainEntry.GetInfo().Name)
	if attr.ParentClosePolicy == nil {
		// for old clients, this field is empty. If they enable the feature, make default as terminate
		if enabled {
			attr.ParentClosePolicy = common.ParentClosePolicyPtr(workflow.ParentClosePolicyTerminate)
		} else {
			attr.ParentClosePolicy = common.ParentClosePolicyPtr(workflow.ParentClosePolicyAbandon)
		}
	} else {
		// for domains that haven't enabled the feature yet, need to use Abandon for backward-compatibility
		if !enabled {
			attr.ParentClosePolicy = common.ParentClosePolicyPtr(workflow.ParentClosePolicyAbandon)
		}
	}

	requestID := uuid.New()
	_, _, err = handler.mutableState.AddStartChildWorkflowExecutionInitiatedEvent(
		handler.decisionTaskCompletedID, requestID, attr,
	)
	return err
}

func (handler *decisionTaskHandlerImpl) handleDecisionSignalExternalWorkflow(
	ctx context.Context,
	attr *workflow.SignalExternalWorkflowExecutionDecisionAttributes,
) error {

	handler.metricsClient.IncCounter(
		metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeSignalExternalWorkflowCounter,
	)

	executionInfo := handler.mutableState.GetExecutionInfo()
	domainID := executionInfo.DomainID
	targetDomainID := domainID
	if attr.GetDomain() != "" {
		targetDomainEntry, err := handler.domainCache.GetDomain(attr.GetDomain())
		if err != nil {
			return &workflow.InternalServiceError{
				Message: fmt.Sprintf("Unable to signal workflow across domain: %v.", attr.GetDomain()),
			}
		}
		targetDomainID = targetDomainEntry.GetInfo().ID
	}

	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateSignalExternalWorkflowExecutionAttributes(
				domainID,
				targetDomainID,
				attr,
			)
		},
		workflow.DecisionTaskFailedCauseBadSignalWorkflowExecutionAttributes,
	); err != nil || handler.stopProcessing {
		return err
	}

	failWorkflow, err := handler.sizeLimitChecker.failWorkflowIfBlobSizeExceedsLimit(
		metrics.DecisionTypeTag(workflow.DecisionTypeSignalExternalWorkflowExecution.String()),
		attr.Input,
		"SignalExternalWorkflowExecutionDecisionAttributes.Input exceeds size limit.",
	)
	if err != nil || failWorkflow {
		handler.stopProcessing = true
		return err
	}

	signalRequestID := uuid.New() // for deduplicate
	_, _, err = handler.mutableState.AddSignalExternalWorkflowExecutionInitiatedEvent(
		handler.decisionTaskCompletedID, signalRequestID, attr,
	)
	return err
}

func (handler *decisionTaskHandlerImpl) handleDecisionUpsertWorkflowSearchAttributes(
	ctx context.Context,
	attr *workflow.UpsertWorkflowSearchAttributesDecisionAttributes,
) error {

	handler.metricsClient.IncCounter(
		metrics.HistoryRespondDecisionTaskCompletedScope,
		metrics.DecisionTypeUpsertWorkflowSearchAttributesCounter,
	)

	// get domain name
	executionInfo := handler.mutableState.GetExecutionInfo()
	domainID := executionInfo.DomainID
	domainEntry, err := handler.domainCache.GetDomainByID(domainID)
	if err != nil {
		return &workflow.InternalServiceError{
			Message: fmt.Sprintf("Unable to get domain for domainID: %v.", domainID),
		}
	}
	domainName := domainEntry.GetInfo().Name

	// valid search attributes for upsert
	if err := handler.validateDecisionAttr(
		func() error {
			return handler.attrValidator.validateUpsertWorkflowSearchAttributes(
				domainName,
				attr,
			)
		},
		workflow.DecisionTaskFailedCauseBadSearchAttributes,
	); err != nil || handler.stopProcessing {
		return err
	}

	// blob size limit check
	failWorkflow, err := handler.sizeLimitChecker.failWorkflowIfBlobSizeExceedsLimit(
		metrics.DecisionTypeTag(workflow.DecisionTypeUpsertWorkflowSearchAttributes.String()),
		convertSearchAttributesToByteArray(attr.GetSearchAttributes().GetIndexedFields()),
		"UpsertWorkflowSearchAttributesDecisionAttributes exceeds size limit.",
	)
	if err != nil || failWorkflow {
		handler.stopProcessing = true
		return err
	}

	_, err = handler.mutableState.AddUpsertWorkflowSearchAttributesEvent(
		handler.decisionTaskCompletedID, attr,
	)
	return err
}

func convertSearchAttributesToByteArray(fields map[string][]byte) []byte {
	result := make([]byte, 0)

	for k, v := range fields {
		result = append(result, []byte(k)...)
		result = append(result, v...)
	}
	return result
}

func (handler *decisionTaskHandlerImpl) retryCronContinueAsNew(
	ctx context.Context,
	attr *workflow.WorkflowExecutionStartedEventAttributes,
	backoffInterval int32,
	continueAsNewIter *workflow.ContinueAsNewInitiator,
	failureReason *string,
	failureDetails []byte,
	lastCompletionResult []byte,
) error {

	continueAsNewAttributes := &workflow.ContinueAsNewWorkflowExecutionDecisionAttributes{
		WorkflowType:                        attr.WorkflowType,
		TaskList:                            attr.TaskList,
		RetryPolicy:                         attr.RetryPolicy,
		Input:                               attr.Input,
		ExecutionStartToCloseTimeoutSeconds: attr.ExecutionStartToCloseTimeoutSeconds,
		TaskStartToCloseTimeoutSeconds:      attr.TaskStartToCloseTimeoutSeconds,
		CronSchedule:                        attr.CronSchedule,
		BackoffStartIntervalInSeconds:       common.Int32Ptr(backoffInterval),
		Initiator:                           continueAsNewIter,
		FailureReason:                       failureReason,
		FailureDetails:                      failureDetails,
		LastCompletionResult:                lastCompletionResult,
		Header:                              attr.Header,
		Memo:                                attr.Memo,
		SearchAttributes:                    attr.SearchAttributes,
	}

	_, newStateBuilder, err := handler.mutableState.AddContinueAsNewEvent(
		ctx,
		handler.decisionTaskCompletedID,
		handler.decisionTaskCompletedID,
		attr.GetParentWorkflowDomain(),
		continueAsNewAttributes,
	)
	if err != nil {
		return err
	}

	handler.continueAsNewBuilder = newStateBuilder
	return nil
}

func (handler *decisionTaskHandlerImpl) validateDecisionAttr(
	validationFn decisionAttrValidationFn,
	failedCause workflow.DecisionTaskFailedCause,
) error {

	if err := validationFn(); err != nil {
		if _, ok := err.(*workflow.BadRequestError); ok {
			return handler.handlerFailDecision(failedCause, err.Error())
		}
		return err
	}

	return nil
}

func (handler *decisionTaskHandlerImpl) handlerFailDecision(
	failedCause workflow.DecisionTaskFailedCause,
	failMessage string,
) error {
	handler.failDecision = true
	handler.failDecisionCause = failedCause.Ptr()
	handler.failMessage = common.StringPtr(failMessage)
	handler.stopProcessing = true
	return nil
}
