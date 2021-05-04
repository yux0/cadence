// Copyright (c) 2017-2020 Uber Technologies, Inc.
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

package frontend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pborman/uuid"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/yarpcerrors"

	"github.com/uber/cadence/.gen/go/health"
	h "github.com/uber/cadence/.gen/go/history"
	m "github.com/uber/cadence/.gen/go/matching"
	gen "github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/client/frontend"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/archiver"
	"github.com/uber/cadence/common/backoff"
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/client"
	"github.com/uber/cadence/common/domain"
	"github.com/uber/cadence/common/elasticsearch/validator"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/messaging"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/quotas"
	"github.com/uber/cadence/common/resource"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/common/types/mapper/thrift"
)

const (
	getDomainReplicationMessageBatchSize = 100
	defaultLastMessageID                 = int64(-1)
)

const (
	// HealthStatusOK is used when this node is healthy and rpc requests are allowed
	HealthStatusOK HealthStatus = iota + 1
	// HealthStatusWarmingUp is used when the rpc handler is warming up
	HealthStatusWarmingUp
	// HealthStatusShuttingDown is used when the rpc handler is shutting down
	HealthStatusShuttingDown
)

var _ Handler = (*WorkflowHandler)(nil)

type (
	// WorkflowHandler - Thrift handler interface for workflow service
	WorkflowHandler struct {
		resource.Resource

		shuttingDown              int32
		healthStatus              int32
		tokenSerializer           common.TaskTokenSerializer
		rateLimiter               quotas.Policy
		config                    *Config
		versionChecker            client.VersionChecker
		domainHandler             domain.Handler
		visibilityQueryValidator  *validator.VisibilityQueryValidator
		searchAttributesValidator *validator.SearchAttributesValidator
	}

	getHistoryContinuationToken struct {
		RunID             string
		FirstEventID      int64
		NextEventID       int64
		IsWorkflowRunning bool
		PersistenceToken  []byte
		TransientDecision *gen.TransientDecisionInfo
		BranchToken       []byte
	}

	domainGetter interface {
		GetDomain() string
	}

	// HealthStatus is an enum that refers to the rpc handler health status
	HealthStatus int32
)

var (
	errDomainNotSet                               = &gen.BadRequestError{Message: "Domain not set on request."}
	errTaskTokenNotSet                            = &gen.BadRequestError{Message: "Task token not set on request."}
	errInvalidTaskToken                           = &gen.BadRequestError{Message: "Invalid TaskToken."}
	errTaskListNotSet                             = &gen.BadRequestError{Message: "TaskList is not set on request."}
	errTaskListTypeNotSet                         = &gen.BadRequestError{Message: "TaskListType is not set on request."}
	errExecutionNotSet                            = &gen.BadRequestError{Message: "Execution is not set on request."}
	errWorkflowIDNotSet                           = &gen.BadRequestError{Message: "WorkflowId is not set on request."}
	errActivityIDNotSet                           = &gen.BadRequestError{Message: "ActivityID is not set on request."}
	errInvalidRunID                               = &gen.BadRequestError{Message: "Invalid RunId."}
	errInvalidNextPageToken                       = &gen.BadRequestError{Message: "Invalid NextPageToken."}
	errNextPageTokenRunIDMismatch                 = &gen.BadRequestError{Message: "RunID in the request does not match the NextPageToken."}
	errQueryNotSet                                = &gen.BadRequestError{Message: "WorkflowQuery is not set on request."}
	errQueryTypeNotSet                            = &gen.BadRequestError{Message: "QueryType is not set on request."}
	errRequestNotSet                              = &gen.BadRequestError{Message: "Request is nil."}
	errNoPermission                               = &gen.BadRequestError{Message: "No permission to do this operation."}
	errRequestIDNotSet                            = &gen.BadRequestError{Message: "RequestId is not set on request."}
	errWorkflowTypeNotSet                         = &gen.BadRequestError{Message: "WorkflowType is not set on request."}
	errInvalidRetention                           = &gen.BadRequestError{Message: "RetentionDays is invalid."}
	errInvalidExecutionStartToCloseTimeoutSeconds = &gen.BadRequestError{Message: "A valid ExecutionStartToCloseTimeoutSeconds is not set on request."}
	errInvalidTaskStartToCloseTimeoutSeconds      = &gen.BadRequestError{Message: "A valid TaskStartToCloseTimeoutSeconds is not set on request."}
	errQueryDisallowedForDomain                   = &gen.BadRequestError{Message: "Domain is not allowed to query, please contact cadence team to re-enable queries."}
	errClusterNameNotSet                          = &gen.BadRequestError{Message: "Cluster name is not set."}
	errEmptyReplicationInfo                       = &gen.BadRequestError{Message: "Replication task info is not set."}
	errEmptyQueueType                             = &gen.BadRequestError{Message: "Queue type is not set."}
	errShuttingDown                               = &gen.InternalServiceError{Message: "Shutting down"}

	// err for archival
	errHistoryNotFound = &gen.BadRequestError{Message: "Requested workflow history not found, may have passed retention period."}

	// err for string too long
	errDomainTooLong       = &gen.BadRequestError{Message: "Domain length exceeds limit."}
	errWorkflowTypeTooLong = &gen.BadRequestError{Message: "WorkflowType length exceeds limit."}
	errWorkflowIDTooLong   = &gen.BadRequestError{Message: "WorkflowID length exceeds limit."}
	errSignalNameTooLong   = &gen.BadRequestError{Message: "SignalName length exceeds limit."}
	errTaskListTooLong     = &gen.BadRequestError{Message: "TaskList length exceeds limit."}
	errRawTaskListTooLong  = &gen.BadRequestError{Message: "Raw TaskList length exceeds limit."}
	errRequestIDTooLong    = &gen.BadRequestError{Message: "RequestID length exceeds limit."}
	errIdentityTooLong     = &gen.BadRequestError{Message: "Identity length exceeds limit."}

	frontendServiceRetryPolicy = common.CreateFrontendServiceRetryPolicy()
)

// NewWorkflowHandler creates a thrift handler for the cadence service
func NewWorkflowHandler(
	resource resource.Resource,
	config *Config,
	replicationMessageSink messaging.Producer,
	versionChecker client.VersionChecker,
) *WorkflowHandler {
	return &WorkflowHandler{
		Resource:        resource,
		config:          config,
		healthStatus:    int32(HealthStatusWarmingUp),
		tokenSerializer: common.NewJSONTaskTokenSerializer(),
		rateLimiter: quotas.NewMultiStageRateLimiter(
			func() float64 {
				return float64(config.RPS())
			},
			func(domain string) float64 {
				if monitor := resource.GetMembershipMonitor(); monitor != nil && config.GlobalDomainRPS(domain) > 0 {
					ringSize, err := monitor.GetMemberCount(common.FrontendServiceName)
					if err == nil && ringSize > 0 {
						avgQuota := common.MaxInt(config.GlobalDomainRPS(domain)/ringSize, 1)
						return float64(common.MinInt(avgQuota, config.MaxDomainRPSPerInstance(domain)))
					}
				}
				return float64(config.MaxDomainRPSPerInstance(domain))
			},
		),
		versionChecker: versionChecker,
		domainHandler: domain.NewHandler(
			config.domainConfig,
			resource.GetLogger(),
			resource.GetMetadataManager(),
			resource.GetClusterMetadata(),
			domain.NewDomainReplicator(replicationMessageSink, resource.GetLogger()),
			resource.GetArchivalMetadata(),
			resource.GetArchiverProvider(),
			resource.GetTimeSource(),
		),
		visibilityQueryValidator: validator.NewQueryValidator(config.ValidSearchAttributes),
		searchAttributesValidator: validator.NewSearchAttributesValidator(
			resource.GetLogger(),
			config.ValidSearchAttributes,
			config.SearchAttributesNumberOfKeysLimit,
			config.SearchAttributesSizeOfValueLimit,
			config.SearchAttributesTotalSizeLimit,
		),
	}
}

// Start starts the handler
func (wh *WorkflowHandler) Start() {
	// TODO: Get warmup duration from config. Even better, run proactive checks such as probing downstream connections.
	const warmUpDuration = 30 * time.Second

	warmupTimer := time.NewTimer(warmUpDuration)
	go func() {
		<-warmupTimer.C
		wh.GetLogger().Warn("Service warmup duration has elapsed.")
		if atomic.CompareAndSwapInt32(&wh.healthStatus, int32(HealthStatusWarmingUp), int32(HealthStatusOK)) {
			wh.GetLogger().Warn("Warmup time has elapsed. Service is healthy.")
		} else {
			status := HealthStatus(atomic.LoadInt32(&wh.healthStatus))
			wh.GetLogger().Warn(fmt.Sprintf("Warmup time has elapsed. Service status is: %v", status.String()))
		}
	}()
}

// Stop stops the handler
func (wh *WorkflowHandler) Stop() {
	atomic.StoreInt32(&wh.shuttingDown, 1)
}

// UpdateHealthStatus sets the health status for this rpc handler.
// This health status will be used within the rpc health check handler
func (wh *WorkflowHandler) UpdateHealthStatus(status HealthStatus) {
	atomic.StoreInt32(&wh.healthStatus, int32(status))
}

func (wh *WorkflowHandler) isShuttingDown() bool {
	return atomic.LoadInt32(&wh.shuttingDown) != 0
}

// Health is for health check
func (wh *WorkflowHandler) Health(ctx context.Context) (*health.HealthStatus, error) {
	status := HealthStatus(atomic.LoadInt32(&wh.healthStatus))
	msg := status.String()

	if status != HealthStatusOK {
		wh.GetLogger().Warn(fmt.Sprintf("Service status is: %v", msg))
	}

	return &health.HealthStatus{
		Ok:  status == HealthStatusOK,
		Msg: &msg,
	}, nil
}

// RegisterDomain creates a new domain which can be used as a container for all resources.  Domain is a top level
// entity within Cadence, used as a container for all resources like workflow executions, tasklists, etc.  Domain
// acts as a sandbox and provides isolation for all resources within the domain.  All resources belongs to exactly one
// domain.
func (wh *WorkflowHandler) RegisterDomain(ctx context.Context, registerRequest *gen.RegisterDomainRequest) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfile(metrics.FrontendRegisterDomainScope)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope)
	}

	if registerRequest == nil {
		return errRequestNotSet
	}

	if registerRequest.GetWorkflowExecutionRetentionPeriodInDays() > common.MaxWorkflowRetentionPeriodInDays {
		return errInvalidRetention
	}

	if err := checkPermission(wh.config, registerRequest.SecurityToken); err != nil {
		return err
	}

	if registerRequest.GetName() == "" {
		return errDomainNotSet
	}

	err := wh.domainHandler.RegisterDomain(ctx, registerRequest)
	if err != nil {
		return wh.error(err, scope)
	}
	return nil
}

// ListDomains returns the information and configuration for a registered domain.
func (wh *WorkflowHandler) ListDomains(
	ctx context.Context,
	listRequest *gen.ListDomainsRequest,
) (response *gen.ListDomainsResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfile(metrics.FrontendListDomainsScope)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if listRequest == nil {
		return nil, errRequestNotSet
	}

	resp, err := wh.domainHandler.ListDomains(ctx, listRequest)
	if err != nil {
		return resp, wh.error(err, scope)
	}
	return resp, err
}

// DescribeDomain returns the information and configuration for a registered domain.
func (wh *WorkflowHandler) DescribeDomain(
	ctx context.Context,
	describeRequest *gen.DescribeDomainRequest,
) (response *gen.DescribeDomainResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfile(metrics.FrontendDescribeDomainScope)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if describeRequest == nil {
		return nil, errRequestNotSet
	}

	if describeRequest.GetName() == "" && describeRequest.GetUUID() == "" {
		return nil, errDomainNotSet
	}

	resp, err := wh.domainHandler.DescribeDomain(ctx, describeRequest)
	if err != nil {
		return resp, wh.error(err, scope)
	}
	return resp, err
}

// UpdateDomain is used to update the information and configuration for a registered domain.
func (wh *WorkflowHandler) UpdateDomain(
	ctx context.Context,
	updateRequest *gen.UpdateDomainRequest,
) (resp *gen.UpdateDomainResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfile(metrics.FrontendUpdateDomainScope)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if updateRequest == nil {
		return nil, errRequestNotSet
	}

	// don't require permission for failover request
	if !isFailoverRequest(updateRequest) {
		if err := checkPermission(wh.config, updateRequest.SecurityToken); err != nil {
			return nil, err
		}
	}

	if isGraceFailoverRequest(updateRequest) {
		if err := wh.checkOngoingFailover(
			ctx,
			updateRequest.Name,
		); err != nil {
			return nil, err
		}
	}

	if updateRequest.GetName() == "" {
		return nil, errDomainNotSet
	}
	// TODO: call remote clusters to verify domain data
	resp, err := wh.domainHandler.UpdateDomain(ctx, updateRequest)
	if err != nil {
		return resp, wh.error(err, scope)
	}
	return resp, err
}

// DeprecateDomain us used to update status of a registered domain to DEPRECATED. Once the domain is deprecated
// it cannot be used to start new workflow executions.  Existing workflow executions will continue to run on
// deprecated domains.
func (wh *WorkflowHandler) DeprecateDomain(ctx context.Context, deprecateRequest *gen.DeprecateDomainRequest) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfile(metrics.FrontendDeprecateDomainScope)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope)
	}

	if deprecateRequest == nil {
		return errRequestNotSet
	}

	if err := checkPermission(wh.config, deprecateRequest.SecurityToken); err != nil {
		return err
	}

	if deprecateRequest.GetName() == "" {
		return errDomainNotSet
	}

	err := wh.domainHandler.DeprecateDomain(ctx, deprecateRequest)
	if err != nil {
		return wh.error(err, scope)
	}
	return err
}

// PollForActivityTask - Poll for an activity task.
func (wh *WorkflowHandler) PollForActivityTask(
	ctx context.Context,
	pollRequest *gen.PollForActivityTaskRequest,
) (resp *gen.PollForActivityTaskResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	callTime := time.Now()

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendPollForActivityTaskScope, pollRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if pollRequest == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	wh.GetLogger().Debug("Received PollForActivityTask")
	if err := common.ValidateLongPollContextTimeout(
		ctx,
		"PollForActivityTask",
		wh.GetThrottledLogger(),
	); err != nil {
		return nil, wh.error(err, scope)
	}

	if pollRequest.Domain == nil || pollRequest.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}

	if !wh.validIDLength(pollRequest.GetDomain(), scope, pollRequest.GetDomain()) {
		return nil, wh.error(errDomainTooLong, scope)
	}

	if err := wh.validateTaskList(pollRequest.TaskList, scope, pollRequest.GetDomain()); err != nil {
		return nil, err
	}
	if !wh.validIDLength(pollRequest.GetIdentity(), scope, pollRequest.GetDomain()) {
		return nil, wh.error(errIdentityTooLong, scope)
	}

	domainID, err := wh.GetDomainCache().GetDomainID(pollRequest.GetDomain())
	if err != nil {
		return nil, wh.error(err, scope)
	}

	pollerID := uuid.New()
	op := func() error {
		clientResp, err := wh.GetMatchingClient().PollForActivityTask(ctx, &types.MatchingPollForActivityTaskRequest{
			DomainUUID:  common.StringPtr(domainID),
			PollerID:    common.StringPtr(pollerID),
			PollRequest: thrift.ToPollForActivityTaskRequest(pollRequest),
		})
		resp = thrift.FromPollForActivityTaskResponse(clientResp)
		return thrift.FromError(err)
	}

	err = backoff.Retry(op, frontendServiceRetryPolicy, common.IsServiceTransientError)
	if err != nil {
		err = wh.cancelOutstandingPoll(ctx, err, domainID, persistence.TaskListTypeActivity, pollRequest.TaskList, pollerID)
		if err != nil {
			// For all other errors log an error and return it back to client.
			ctxTimeout := "not-set"
			ctxDeadline, ok := ctx.Deadline()
			if ok {
				ctxTimeout = ctxDeadline.Sub(callTime).String()
			}
			wh.GetLogger().Error("PollForActivityTask failed.",
				tag.WorkflowTaskListName(pollRequest.GetTaskList().GetName()),
				tag.Value(ctxTimeout),
				tag.Error(err))
			return nil, wh.error(err, scope)
		}
	}
	return resp, nil
}

// PollForDecisionTask - Poll for a decision task.
func (wh *WorkflowHandler) PollForDecisionTask(
	ctx context.Context,
	pollRequest *gen.PollForDecisionTaskRequest,
) (resp *gen.PollForDecisionTaskResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	tagsForErrorLog := []tag.Tag{tag.WorkflowDomainName(pollRequest.GetDomain())}
	callTime := time.Now()

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendPollForDecisionTaskScope, pollRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope, tagsForErrorLog...)
	}

	if pollRequest == nil {
		return nil, wh.error(errRequestNotSet, scope, tagsForErrorLog...)
	}

	wh.GetLogger().Debug("Received PollForDecisionTask")
	if err := common.ValidateLongPollContextTimeout(
		ctx,
		"PollForDecisionTask",
		wh.GetThrottledLogger(),
	); err != nil {
		return nil, wh.error(err, scope, tagsForErrorLog...)
	}

	if pollRequest.Domain == nil || pollRequest.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope, tagsForErrorLog...)
	}
	if !wh.validIDLength(pollRequest.GetDomain(), scope, pollRequest.GetDomain()) {
		return nil, wh.error(errDomainTooLong, scope, tagsForErrorLog...)
	}

	if !wh.validIDLength(pollRequest.GetIdentity(), scope, pollRequest.GetDomain()) {
		return nil, wh.error(errIdentityTooLong, scope, tagsForErrorLog...)
	}

	if err := wh.validateTaskList(pollRequest.TaskList, scope, pollRequest.GetDomain()); err != nil {
		return nil, err
	}

	domainName := pollRequest.GetDomain()
	domainEntry, err := wh.GetDomainCache().GetDomain(domainName)
	if err != nil {
		return nil, wh.error(err, scope, tagsForErrorLog...)
	}
	domainID := domainEntry.GetInfo().ID

	wh.GetLogger().Debug("Poll for decision.", tag.WorkflowDomainName(domainName), tag.WorkflowDomainID(domainID))
	if err := wh.checkBadBinary(domainEntry, pollRequest.GetBinaryChecksum()); err != nil {
		return nil, wh.error(err, scope, tagsForErrorLog...)
	}

	pollerID := uuid.New()
	var matchingResp *m.PollForDecisionTaskResponse
	op := func() error {
		clientResp, err := wh.GetMatchingClient().PollForDecisionTask(ctx, &types.MatchingPollForDecisionTaskRequest{
			DomainUUID:  common.StringPtr(domainID),
			PollerID:    common.StringPtr(pollerID),
			PollRequest: thrift.ToPollForDecisionTaskRequest(pollRequest),
		})
		matchingResp = thrift.FromMatchingPollForDecisionTaskResponse(clientResp)
		return thrift.FromError(err)
	}

	err = backoff.Retry(op, frontendServiceRetryPolicy, common.IsServiceTransientError)
	if err != nil {
		err = wh.cancelOutstandingPoll(ctx, err, domainID, persistence.TaskListTypeDecision, pollRequest.TaskList, pollerID)
		if err != nil {
			// For all other errors log an error and return it back to client.
			ctxTimeout := "not-set"
			ctxDeadline, ok := ctx.Deadline()
			if ok {
				ctxTimeout = ctxDeadline.Sub(callTime).String()
			}
			wh.GetLogger().Error("PollForDecisionTask failed.",
				tag.WorkflowTaskListName(pollRequest.GetTaskList().GetName()),
				tag.Value(ctxTimeout),
				tag.Error(err))
			return nil, wh.error(err, scope)
		}

		// Must be cancellation error.  Does'nt matter what we return here.  Client already went away.
		return nil, nil
	}

	tagsForErrorLog = append(tagsForErrorLog, []tag.Tag{tag.WorkflowID(
		matchingResp.GetWorkflowExecution().GetWorkflowId()),
		tag.WorkflowRunID(matchingResp.GetWorkflowExecution().GetRunId())}...)
	resp, err = wh.createPollForDecisionTaskResponse(ctx, scope, domainID, matchingResp, matchingResp.GetBranchToken())
	if err != nil {
		return nil, wh.error(err, scope, tagsForErrorLog...)
	}
	return resp, nil
}

func (wh *WorkflowHandler) checkBadBinary(domainEntry *cache.DomainCacheEntry, binaryChecksum string) error {
	if domainEntry.GetConfig().BadBinaries.Binaries != nil {
		badBinaries := domainEntry.GetConfig().BadBinaries.Binaries
		_, ok := badBinaries[binaryChecksum]
		if ok {
			wh.GetMetricsClient().IncCounter(metrics.FrontendPollForDecisionTaskScope, metrics.CadenceErrBadBinaryCounter)
			return &gen.BadRequestError{
				Message: fmt.Sprintf("binary %v already marked as bad deployment", binaryChecksum),
			}
		}
	}
	return nil
}

func (wh *WorkflowHandler) cancelOutstandingPoll(ctx context.Context, err error, domainID string, taskListType int32,
	taskList *gen.TaskList, pollerID string) error {
	// First check if this err is due to context cancellation.  This means client connection to frontend is closed.
	if ctx.Err() == context.Canceled {
		// Our rpc stack does not propagates context cancellation to the other service.  Lets make an explicit
		// call to matching to notify this poller is gone to prevent any tasks being dispatched to zombie pollers.
		err = wh.GetMatchingClient().CancelOutstandingPoll(context.Background(), &types.CancelOutstandingPollRequest{
			DomainUUID:   common.StringPtr(domainID),
			TaskListType: common.Int32Ptr(taskListType),
			TaskList:     thrift.ToTaskList(taskList),
			PollerID:     common.StringPtr(pollerID),
		})
		err = thrift.FromError(err)
		// We can not do much if this call fails.  Just log the error and move on
		if err != nil {
			wh.GetLogger().Warn("Failed to cancel outstanding poller.",
				tag.WorkflowTaskListName(taskList.GetName()), tag.Error(err))
		}

		// Clear error as we don't want to report context cancellation error to count against our SLA
		return nil
	}

	return err
}

// RecordActivityTaskHeartbeat - Record Activity Task Heart beat.
func (wh *WorkflowHandler) RecordActivityTaskHeartbeat(
	ctx context.Context,
	heartbeatRequest *gen.RecordActivityTaskHeartbeatRequest,
) (resp *gen.RecordActivityTaskHeartbeatResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope := wh.getDefaultScope(metrics.FrontendRecordActivityTaskHeartbeatScope)

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if heartbeatRequest == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	// Count the request in the RPS, but we still accept it even if RPS is exceeded
	wh.allow(nil)

	wh.GetLogger().Debug("Received RecordActivityTaskHeartbeat")
	if heartbeatRequest.TaskToken == nil {
		return nil, wh.error(errTaskTokenNotSet, scope)
	}
	taskToken, err := wh.tokenSerializer.Deserialize(heartbeatRequest.TaskToken)
	if err != nil {
		return nil, wh.error(err, scope)
	}
	if taskToken.DomainID == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}

	domainEntry, err := wh.GetDomainCache().GetDomainByID(taskToken.DomainID)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	scope, sw := wh.startRequestProfileWithDomain(
		metrics.FrontendRecordActivityTaskHeartbeatScope,
		domainWrapper{
			domain: domainEntry.GetInfo().Name,
		},
	)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	sizeLimitError := wh.config.BlobSizeLimitError(domainEntry.GetInfo().Name)
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(domainEntry.GetInfo().Name)

	if err := common.CheckEventBlobSizeLimit(
		len(heartbeatRequest.Details),
		sizeLimitWarn,
		sizeLimitError,
		taskToken.DomainID,
		taskToken.WorkflowID,
		taskToken.RunID,
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("RecordActivityTaskHeartbeat"),
	); err != nil {
		// heartbeat details exceed size limit, we would fail the activity immediately with explicit error reason
		failRequest := &types.RespondActivityTaskFailedRequest{
			TaskToken: heartbeatRequest.TaskToken,
			Reason:    common.StringPtr(common.FailureReasonHeartbeatExceedsLimit),
			Details:   heartbeatRequest.Details[0:sizeLimitError],
			Identity:  heartbeatRequest.Identity,
		}
		err = wh.GetHistoryClient().RespondActivityTaskFailed(ctx, &types.HistoryRespondActivityTaskFailedRequest{
			DomainUUID:    common.StringPtr(taskToken.DomainID),
			FailedRequest: failRequest,
		})
		err = thrift.FromError(err)
		if err != nil {
			return nil, wh.error(err, scope)
		}
		resp = &gen.RecordActivityTaskHeartbeatResponse{CancelRequested: common.BoolPtr(true)}
	} else {
		clientResp, err := wh.GetHistoryClient().RecordActivityTaskHeartbeat(ctx, &types.HistoryRecordActivityTaskHeartbeatRequest{
			DomainUUID:       common.StringPtr(taskToken.DomainID),
			HeartbeatRequest: thrift.ToRecordActivityTaskHeartbeatRequest(heartbeatRequest),
		})
		resp = thrift.FromRecordActivityTaskHeartbeatResponse(clientResp)
		err = thrift.FromError(err)
		if err != nil {
			return nil, wh.error(err, scope)
		}
	}

	return resp, nil
}

// RecordActivityTaskHeartbeatByID - Record Activity Task Heart beat.
func (wh *WorkflowHandler) RecordActivityTaskHeartbeatByID(
	ctx context.Context,
	heartbeatRequest *gen.RecordActivityTaskHeartbeatByIDRequest,
) (resp *gen.RecordActivityTaskHeartbeatResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendRecordActivityTaskHeartbeatByIDScope, heartbeatRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if heartbeatRequest == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	// Count the request in the RPS, but we still accept it even if RPS is exceeded
	wh.allow(nil)

	wh.GetLogger().Debug("Received RecordActivityTaskHeartbeatByID")
	domainID, err := wh.GetDomainCache().GetDomainID(heartbeatRequest.GetDomain())
	if err != nil {
		return nil, wh.error(err, scope)
	}
	workflowID := heartbeatRequest.GetWorkflowID()
	runID := heartbeatRequest.GetRunID() // runID is optional so can be empty
	activityID := heartbeatRequest.GetActivityID()

	if domainID == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}
	if workflowID == "" {
		return nil, wh.error(errWorkflowIDNotSet, scope)
	}
	if activityID == "" {
		return nil, wh.error(errActivityIDNotSet, scope)
	}

	taskToken := &common.TaskToken{
		DomainID:   domainID,
		RunID:      runID,
		WorkflowID: workflowID,
		ScheduleID: common.EmptyEventID,
		ActivityID: activityID,
	}
	token, err := wh.tokenSerializer.Serialize(taskToken)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	domainEntry, err := wh.GetDomainCache().GetDomainByID(taskToken.DomainID)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	// add domain tag to scope, so further metrics will have the domain tag
	scope = scope.Tagged(metrics.DomainTag(domainEntry.GetInfo().Name))

	sizeLimitError := wh.config.BlobSizeLimitError(domainEntry.GetInfo().Name)
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(domainEntry.GetInfo().Name)

	if err := common.CheckEventBlobSizeLimit(
		len(heartbeatRequest.Details),
		sizeLimitWarn,
		sizeLimitError,
		taskToken.DomainID,
		taskToken.WorkflowID,
		taskToken.RunID,
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("RecordActivityTaskHeartbeatByID"),
	); err != nil {
		// heartbeat details exceed size limit, we would fail the activity immediately with explicit error reason
		failRequest := &types.RespondActivityTaskFailedRequest{
			TaskToken: token,
			Reason:    common.StringPtr(common.FailureReasonHeartbeatExceedsLimit),
			Details:   heartbeatRequest.Details[0:sizeLimitError],
			Identity:  heartbeatRequest.Identity,
		}
		err = wh.GetHistoryClient().RespondActivityTaskFailed(ctx, &types.HistoryRespondActivityTaskFailedRequest{
			DomainUUID:    common.StringPtr(taskToken.DomainID),
			FailedRequest: failRequest,
		})
		err = thrift.FromError(err)
		if err != nil {
			return nil, wh.error(err, scope)
		}
		resp = &gen.RecordActivityTaskHeartbeatResponse{CancelRequested: common.BoolPtr(true)}
	} else {
		req := &gen.RecordActivityTaskHeartbeatRequest{
			TaskToken: token,
			Details:   heartbeatRequest.Details,
			Identity:  heartbeatRequest.Identity,
		}

		clientResp, err := wh.GetHistoryClient().RecordActivityTaskHeartbeat(ctx, &types.HistoryRecordActivityTaskHeartbeatRequest{
			DomainUUID:       common.StringPtr(taskToken.DomainID),
			HeartbeatRequest: thrift.ToRecordActivityTaskHeartbeatRequest(req),
		})
		resp = thrift.FromRecordActivityTaskHeartbeatResponse(clientResp)
		err = thrift.FromError(err)
		if err != nil {
			return nil, wh.error(err, scope)
		}
	}

	return resp, nil
}

// RespondActivityTaskCompleted - response to an activity task
func (wh *WorkflowHandler) RespondActivityTaskCompleted(
	ctx context.Context,
	completeRequest *gen.RespondActivityTaskCompletedRequest,
) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope := wh.getDefaultScope(metrics.FrontendRespondActivityTaskCompletedScope)
	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope)
	}

	if completeRequest == nil {
		return wh.error(errRequestNotSet, scope)
	}

	// Count the request in the RPS, but we still accept it even if RPS is exceeded
	wh.allow(nil)

	if completeRequest.TaskToken == nil {
		return wh.error(errTaskTokenNotSet, scope)
	}
	taskToken, err := wh.tokenSerializer.Deserialize(completeRequest.TaskToken)
	if err != nil {
		return wh.error(err, scope)
	}
	if taskToken.DomainID == "" {
		return wh.error(errDomainNotSet, scope)
	}

	domainEntry, err := wh.GetDomainCache().GetDomainByID(taskToken.DomainID)
	if err != nil {
		return wh.error(err, scope)
	}
	if !wh.validIDLength(completeRequest.GetIdentity(), scope, domainEntry.GetInfo().Name) {
		return wh.error(errIdentityTooLong, scope)
	}

	scope, sw := wh.startRequestProfileWithDomain(
		metrics.FrontendRespondActivityTaskCompletedScope,
		domainWrapper{
			domain: domainEntry.GetInfo().Name,
		},
	)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return errShuttingDown
	}

	sizeLimitError := wh.config.BlobSizeLimitError(domainEntry.GetInfo().Name)
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(domainEntry.GetInfo().Name)

	if err := common.CheckEventBlobSizeLimit(
		len(completeRequest.Result),
		sizeLimitWarn,
		sizeLimitError,
		taskToken.DomainID,
		taskToken.WorkflowID,
		taskToken.RunID,
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("RespondActivityTaskCompleted"),
	); err != nil {
		// result exceeds blob size limit, we would record it as failure
		failRequest := &types.RespondActivityTaskFailedRequest{
			TaskToken: completeRequest.TaskToken,
			Reason:    common.StringPtr(common.FailureReasonCompleteResultExceedsLimit),
			Details:   completeRequest.Result[0:sizeLimitError],
			Identity:  completeRequest.Identity,
		}
		err = wh.GetHistoryClient().RespondActivityTaskFailed(ctx, &types.HistoryRespondActivityTaskFailedRequest{
			DomainUUID:    common.StringPtr(taskToken.DomainID),
			FailedRequest: failRequest,
		})
		err = thrift.FromError(err)
		if err != nil {
			return wh.error(err, scope)
		}
	} else {
		err = wh.GetHistoryClient().RespondActivityTaskCompleted(ctx, &types.HistoryRespondActivityTaskCompletedRequest{
			DomainUUID:      common.StringPtr(taskToken.DomainID),
			CompleteRequest: thrift.ToRespondActivityTaskCompletedRequest(completeRequest),
		})
		err = thrift.FromError(err)
		if err != nil {
			return wh.error(err, scope)
		}
	}

	return nil
}

// RespondActivityTaskCompletedByID - response to an activity task
func (wh *WorkflowHandler) RespondActivityTaskCompletedByID(
	ctx context.Context,
	completeRequest *gen.RespondActivityTaskCompletedByIDRequest,
) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendRespondActivityTaskCompletedByIDScope, completeRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope)
	}

	if completeRequest == nil {
		return wh.error(errRequestNotSet, scope)
	}

	// Count the request in the RPS, but we still accept it even if RPS is exceeded
	wh.allow(nil)

	domainID, err := wh.GetDomainCache().GetDomainID(completeRequest.GetDomain())
	if err != nil {
		return wh.error(err, scope)
	}
	workflowID := completeRequest.GetWorkflowID()
	runID := completeRequest.GetRunID() // runID is optional so can be empty
	activityID := completeRequest.GetActivityID()

	if domainID == "" {
		return wh.error(errDomainNotSet, scope)
	}
	if workflowID == "" {
		return wh.error(errWorkflowIDNotSet, scope)
	}
	if activityID == "" {
		return wh.error(errActivityIDNotSet, scope)
	}

	if !wh.validIDLength(completeRequest.GetIdentity(), scope, completeRequest.GetDomain()) {
		return wh.error(errIdentityTooLong, scope)
	}

	taskToken := &common.TaskToken{
		DomainID:   domainID,
		RunID:      runID,
		WorkflowID: workflowID,
		ScheduleID: common.EmptyEventID,
		ActivityID: activityID,
	}
	token, err := wh.tokenSerializer.Serialize(taskToken)
	if err != nil {
		return wh.error(err, scope)
	}

	domainEntry, err := wh.GetDomainCache().GetDomainByID(taskToken.DomainID)
	if err != nil {
		return wh.error(err, scope)
	}

	// add domain tag to scope, so further metrics will have the domain tag
	scope = scope.Tagged(metrics.DomainTag(domainEntry.GetInfo().Name))

	sizeLimitError := wh.config.BlobSizeLimitError(domainEntry.GetInfo().Name)
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(domainEntry.GetInfo().Name)

	if err := common.CheckEventBlobSizeLimit(
		len(completeRequest.Result),
		sizeLimitWarn,
		sizeLimitError,
		taskToken.DomainID,
		taskToken.WorkflowID,
		taskToken.RunID,
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("RespondActivityTaskCompletedByID"),
	); err != nil {
		// result exceeds blob size limit, we would record it as failure
		failRequest := &types.RespondActivityTaskFailedRequest{
			TaskToken: token,
			Reason:    common.StringPtr(common.FailureReasonCompleteResultExceedsLimit),
			Details:   completeRequest.Result[0:sizeLimitError],
			Identity:  completeRequest.Identity,
		}
		err = wh.GetHistoryClient().RespondActivityTaskFailed(ctx, &types.HistoryRespondActivityTaskFailedRequest{
			DomainUUID:    common.StringPtr(taskToken.DomainID),
			FailedRequest: failRequest,
		})
		err = thrift.FromError(err)
		if err != nil {
			return wh.error(err, scope)
		}
	} else {
		req := &types.RespondActivityTaskCompletedRequest{
			TaskToken: token,
			Result:    completeRequest.Result,
			Identity:  completeRequest.Identity,
		}

		err = wh.GetHistoryClient().RespondActivityTaskCompleted(ctx, &types.HistoryRespondActivityTaskCompletedRequest{
			DomainUUID:      common.StringPtr(taskToken.DomainID),
			CompleteRequest: req,
		})
		err = thrift.FromError(err)
		if err != nil {
			return wh.error(err, scope)
		}
	}

	return nil
}

// RespondActivityTaskFailed - response to an activity task failure
func (wh *WorkflowHandler) RespondActivityTaskFailed(
	ctx context.Context,
	failedRequest *gen.RespondActivityTaskFailedRequest,
) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope := wh.getDefaultScope(metrics.FrontendRespondActivityTaskFailedScope)
	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope)
	}

	if failedRequest == nil {
		return wh.error(errRequestNotSet, scope)
	}

	// Count the request in the RPS, but we still accept it even if RPS is exceeded
	wh.allow(nil)

	if failedRequest.TaskToken == nil {
		return wh.error(errTaskTokenNotSet, scope)
	}
	taskToken, err := wh.tokenSerializer.Deserialize(failedRequest.TaskToken)
	if err != nil {
		return wh.error(err, scope)
	}
	if taskToken.DomainID == "" {
		return wh.error(errDomainNotSet, scope)
	}

	domainEntry, err := wh.GetDomainCache().GetDomainByID(taskToken.DomainID)
	if err != nil {
		return wh.error(err, scope)
	}

	scope, sw := wh.startRequestProfileWithDomain(
		metrics.FrontendRespondActivityTaskFailedScope,
		domainWrapper{
			domain: domainEntry.GetInfo().Name,
		},
	)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return errShuttingDown
	}

	if !wh.validIDLength(failedRequest.GetIdentity(), scope, domainEntry.GetInfo().Name) {
		return wh.error(errIdentityTooLong, scope)
	}

	sizeLimitError := wh.config.BlobSizeLimitError(domainEntry.GetInfo().Name)
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(domainEntry.GetInfo().Name)

	if err := common.CheckEventBlobSizeLimit(
		len(failedRequest.Details),
		sizeLimitWarn,
		sizeLimitError,
		taskToken.DomainID,
		taskToken.WorkflowID,
		taskToken.RunID,
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("RespondActivityTaskFailed"),
	); err != nil {
		// details exceeds blob size limit, we would truncate the details and put a specific error reason
		failedRequest.Reason = common.StringPtr(common.FailureReasonFailureDetailsExceedsLimit)
		failedRequest.Details = failedRequest.Details[0:sizeLimitError]
	}

	err = wh.GetHistoryClient().RespondActivityTaskFailed(ctx, &types.HistoryRespondActivityTaskFailedRequest{
		DomainUUID:    common.StringPtr(taskToken.DomainID),
		FailedRequest: thrift.ToRespondActivityTaskFailedRequest(failedRequest),
	})
	err = thrift.FromError(err)
	if err != nil {
		return wh.error(err, scope)
	}
	return nil
}

// RespondActivityTaskFailedByID - response to an activity task failure
func (wh *WorkflowHandler) RespondActivityTaskFailedByID(
	ctx context.Context,
	failedRequest *gen.RespondActivityTaskFailedByIDRequest,
) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendRespondActivityTaskFailedByIDScope, failedRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope)
	}

	if failedRequest == nil {
		return wh.error(errRequestNotSet, scope)
	}

	// Count the request in the RPS, but we still accept it even if RPS is exceeded
	wh.allow(nil)

	domainID, err := wh.GetDomainCache().GetDomainID(failedRequest.GetDomain())
	if err != nil {
		return wh.error(err, scope)
	}
	workflowID := failedRequest.GetWorkflowID()
	runID := failedRequest.GetRunID() // runID is optional so can be empty
	activityID := failedRequest.GetActivityID()

	if domainID == "" {
		return wh.error(errDomainNotSet, scope)
	}
	if workflowID == "" {
		return wh.error(errWorkflowIDNotSet, scope)
	}
	if activityID == "" {
		return wh.error(errActivityIDNotSet, scope)
	}
	if !wh.validIDLength(failedRequest.GetIdentity(), scope, failedRequest.GetDomain()) {
		return wh.error(errIdentityTooLong, scope)
	}

	taskToken := &common.TaskToken{
		DomainID:   domainID,
		RunID:      runID,
		WorkflowID: workflowID,
		ScheduleID: common.EmptyEventID,
		ActivityID: activityID,
	}
	token, err := wh.tokenSerializer.Serialize(taskToken)
	if err != nil {
		return wh.error(err, scope)
	}

	domainEntry, err := wh.GetDomainCache().GetDomainByID(taskToken.DomainID)
	if err != nil {
		return wh.error(err, scope)
	}

	// add domain tag to scope, so further metrics will have the domain tag
	scope = scope.Tagged(metrics.DomainTag(domainEntry.GetInfo().Name))

	sizeLimitError := wh.config.BlobSizeLimitError(domainEntry.GetInfo().Name)
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(domainEntry.GetInfo().Name)

	if err := common.CheckEventBlobSizeLimit(
		len(failedRequest.Details),
		sizeLimitWarn,
		sizeLimitError,
		taskToken.DomainID,
		taskToken.WorkflowID,
		taskToken.RunID,
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("RespondActivityTaskFailedByID"),
	); err != nil {
		// details exceeds blob size limit, we would truncate the details and put a specific error reason
		failedRequest.Reason = common.StringPtr(common.FailureReasonFailureDetailsExceedsLimit)
		failedRequest.Details = failedRequest.Details[0:sizeLimitError]
	}

	req := &types.RespondActivityTaskFailedRequest{
		TaskToken: token,
		Reason:    failedRequest.Reason,
		Details:   failedRequest.Details,
		Identity:  failedRequest.Identity,
	}

	err = wh.GetHistoryClient().RespondActivityTaskFailed(ctx, &types.HistoryRespondActivityTaskFailedRequest{
		DomainUUID:    common.StringPtr(taskToken.DomainID),
		FailedRequest: req,
	})
	err = thrift.FromError(err)
	if err != nil {
		return wh.error(err, scope)
	}
	return nil
}

// RespondActivityTaskCanceled - called to cancel an activity task
func (wh *WorkflowHandler) RespondActivityTaskCanceled(
	ctx context.Context,
	cancelRequest *gen.RespondActivityTaskCanceledRequest,
) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope := wh.getDefaultScope(metrics.FrontendRespondActivityTaskCanceledScope)
	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope)
	}

	if cancelRequest == nil {
		return wh.error(errRequestNotSet, scope)
	}

	// Count the request in the RPS, but we still accept it even if RPS is exceeded
	wh.allow(nil)

	if cancelRequest.TaskToken == nil {
		return wh.error(errTaskTokenNotSet, scope)
	}
	taskToken, err := wh.tokenSerializer.Deserialize(cancelRequest.TaskToken)
	if err != nil {
		return wh.error(err, scope)
	}
	if taskToken.DomainID == "" {
		return wh.error(errDomainNotSet, scope)
	}

	domainEntry, err := wh.GetDomainCache().GetDomainByID(taskToken.DomainID)
	if err != nil {
		return wh.error(err, scope)
	}

	scope, sw := wh.startRequestProfileWithDomain(
		metrics.FrontendRespondActivityTaskCanceledScope,
		domainWrapper{
			domain: domainEntry.GetInfo().Name,
		},
	)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return errShuttingDown
	}

	if !wh.validIDLength(cancelRequest.GetIdentity(), scope, domainEntry.GetInfo().Name) {
		return wh.error(errIdentityTooLong, scope)
	}

	sizeLimitError := wh.config.BlobSizeLimitError(domainEntry.GetInfo().Name)
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(domainEntry.GetInfo().Name)

	if err := common.CheckEventBlobSizeLimit(
		len(cancelRequest.Details),
		sizeLimitWarn,
		sizeLimitError,
		taskToken.DomainID,
		taskToken.WorkflowID,
		taskToken.RunID,
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("RespondActivityTaskCanceled"),
	); err != nil {
		// details exceeds blob size limit, we would record it as failure
		failRequest := &types.RespondActivityTaskFailedRequest{
			TaskToken: cancelRequest.TaskToken,
			Reason:    common.StringPtr(common.FailureReasonCancelDetailsExceedsLimit),
			Details:   cancelRequest.Details[0:sizeLimitError],
			Identity:  cancelRequest.Identity,
		}
		err = wh.GetHistoryClient().RespondActivityTaskFailed(ctx, &types.HistoryRespondActivityTaskFailedRequest{
			DomainUUID:    common.StringPtr(taskToken.DomainID),
			FailedRequest: failRequest,
		})
		err = thrift.FromError(err)
		if err != nil {
			return wh.error(err, scope)
		}
	} else {
		err = wh.GetHistoryClient().RespondActivityTaskCanceled(ctx, &types.HistoryRespondActivityTaskCanceledRequest{
			DomainUUID:    common.StringPtr(taskToken.DomainID),
			CancelRequest: thrift.ToRespondActivityTaskCanceledRequest(cancelRequest),
		})
		err = thrift.FromError(err)
		if err != nil {
			return wh.error(err, scope)
		}
	}

	return nil
}

// RespondActivityTaskCanceledByID - called to cancel an activity task
func (wh *WorkflowHandler) RespondActivityTaskCanceledByID(
	ctx context.Context,
	cancelRequest *gen.RespondActivityTaskCanceledByIDRequest,
) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendRespondActivityTaskCanceledScope, cancelRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope)
	}

	if cancelRequest == nil {
		return wh.error(errRequestNotSet, scope)
	}

	// Count the request in the RPS, but we still accept it even if RPS is exceeded
	wh.allow(nil)

	domainID, err := wh.GetDomainCache().GetDomainID(cancelRequest.GetDomain())
	if err != nil {
		return wh.error(err, scope)
	}
	workflowID := cancelRequest.GetWorkflowID()
	runID := cancelRequest.GetRunID() // runID is optional so can be empty
	activityID := cancelRequest.GetActivityID()

	if domainID == "" {
		return wh.error(errDomainNotSet, scope)
	}
	if workflowID == "" {
		return wh.error(errWorkflowIDNotSet, scope)
	}
	if activityID == "" {
		return wh.error(errActivityIDNotSet, scope)
	}
	if !wh.validIDLength(cancelRequest.GetIdentity(), scope, cancelRequest.GetDomain()) {
		return wh.error(errIdentityTooLong, scope)
	}

	taskToken := &common.TaskToken{
		DomainID:   domainID,
		RunID:      runID,
		WorkflowID: workflowID,
		ScheduleID: common.EmptyEventID,
		ActivityID: activityID,
	}
	token, err := wh.tokenSerializer.Serialize(taskToken)
	if err != nil {
		return wh.error(err, scope)
	}

	domainEntry, err := wh.GetDomainCache().GetDomainByID(taskToken.DomainID)
	if err != nil {
		return wh.error(err, scope)
	}

	// add domain tag to scope, so further metrics will have the domain tag
	scope = scope.Tagged(metrics.DomainTag(domainEntry.GetInfo().Name))

	sizeLimitError := wh.config.BlobSizeLimitError(domainEntry.GetInfo().Name)
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(domainEntry.GetInfo().Name)

	if err := common.CheckEventBlobSizeLimit(
		len(cancelRequest.Details),
		sizeLimitWarn,
		sizeLimitError,
		taskToken.DomainID,
		taskToken.WorkflowID,
		taskToken.RunID,
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("RespondActivityTaskCanceledByID"),
	); err != nil {
		// details exceeds blob size limit, we would record it as failure
		failRequest := &types.RespondActivityTaskFailedRequest{
			TaskToken: token,
			Reason:    common.StringPtr(common.FailureReasonCancelDetailsExceedsLimit),
			Details:   cancelRequest.Details[0:sizeLimitError],
			Identity:  cancelRequest.Identity,
		}
		err = wh.GetHistoryClient().RespondActivityTaskFailed(ctx, &types.HistoryRespondActivityTaskFailedRequest{
			DomainUUID:    common.StringPtr(taskToken.DomainID),
			FailedRequest: failRequest,
		})
		err = thrift.FromError(err)
		if err != nil {
			return wh.error(err, scope)
		}
	} else {
		req := &gen.RespondActivityTaskCanceledRequest{
			TaskToken: token,
			Details:   cancelRequest.Details,
			Identity:  cancelRequest.Identity,
		}

		err = wh.GetHistoryClient().RespondActivityTaskCanceled(ctx, &types.HistoryRespondActivityTaskCanceledRequest{
			DomainUUID:    common.StringPtr(taskToken.DomainID),
			CancelRequest: thrift.ToRespondActivityTaskCanceledRequest(req),
		})
		err = thrift.FromError(err)
		if err != nil {
			return wh.error(err, scope)
		}
	}

	return nil
}

// RespondDecisionTaskCompleted - response to a decision task
func (wh *WorkflowHandler) RespondDecisionTaskCompleted(
	ctx context.Context,
	completeRequest *gen.RespondDecisionTaskCompletedRequest,
) (resp *gen.RespondDecisionTaskCompletedResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope := wh.getDefaultScope(metrics.FrontendRespondDecisionTaskCompletedScope)
	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if completeRequest == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	// Count the request in the RPS, but we still accept it even if RPS is exceeded
	wh.allow(nil)

	if completeRequest.TaskToken == nil {
		return nil, wh.error(errTaskTokenNotSet, scope)
	}
	taskToken, err := wh.tokenSerializer.Deserialize(completeRequest.TaskToken)
	if err != nil {
		return nil, wh.error(err, scope)
	}
	if taskToken.DomainID == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}

	domainEntry, err := wh.GetDomainCache().GetDomainByID(taskToken.DomainID)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	scope, sw := wh.startRequestProfileWithDomain(
		metrics.FrontendRespondDecisionTaskCompletedScope,
		domainWrapper{
			domain: domainEntry.GetInfo().Name,
		},
	)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	clientResp, err := wh.GetHistoryClient().RespondDecisionTaskCompleted(ctx, &types.HistoryRespondDecisionTaskCompletedRequest{
		DomainUUID:      common.StringPtr(taskToken.DomainID),
		CompleteRequest: thrift.ToRespondDecisionTaskCompletedRequest(completeRequest)},
	)
	histResp := thrift.FromHistoryRespondDecisionTaskCompletedResponse(clientResp)
	err = thrift.FromError(err)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	if !wh.validIDLength(completeRequest.GetIdentity(), scope, domainEntry.GetInfo().Name) {
		return nil, wh.error(errIdentityTooLong, scope)
	}

	completedResp := &gen.RespondDecisionTaskCompletedResponse{}
	completedResp.ActivitiesToDispatchLocally = histResp.ActivitiesToDispatchLocally
	if completeRequest.GetReturnNewDecisionTask() && histResp != nil && histResp.StartedResponse != nil {
		taskToken := &common.TaskToken{
			DomainID:        taskToken.DomainID,
			WorkflowID:      taskToken.WorkflowID,
			RunID:           taskToken.RunID,
			ScheduleID:      histResp.StartedResponse.GetScheduledEventId(),
			ScheduleAttempt: histResp.StartedResponse.GetAttempt(),
		}
		token, _ := wh.tokenSerializer.Serialize(taskToken)
		workflowExecution := &types.WorkflowExecution{
			WorkflowID: common.StringPtr(taskToken.WorkflowID),
			RunID:      common.StringPtr(taskToken.RunID),
		}
		matchingResp := common.CreateMatchingPollForDecisionTaskResponse(thrift.ToRecordDecisionTaskStartedResponse(histResp.StartedResponse), workflowExecution, token)

		newDecisionTask, err := wh.createPollForDecisionTaskResponse(ctx, scope, taskToken.DomainID, thrift.FromMatchingPollForDecisionTaskResponse(matchingResp), matchingResp.GetBranchToken())
		if err != nil {
			return nil, wh.error(err, scope)
		}
		completedResp.DecisionTask = newDecisionTask
	}

	return completedResp, nil
}

// RespondDecisionTaskFailed - failed response to a decision task
func (wh *WorkflowHandler) RespondDecisionTaskFailed(
	ctx context.Context,
	failedRequest *gen.RespondDecisionTaskFailedRequest,
) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope := wh.getDefaultScope(metrics.FrontendRespondDecisionTaskFailedScope)
	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope)
	}

	if failedRequest == nil {
		return wh.error(errRequestNotSet, scope)
	}

	// Count the request in the RPS, but we still accept it even if RPS is exceeded
	wh.allow(nil)

	if failedRequest.TaskToken == nil {
		return wh.error(errTaskTokenNotSet, scope)
	}
	taskToken, err := wh.tokenSerializer.Deserialize(failedRequest.TaskToken)
	if err != nil {
		return wh.error(err, scope)
	}
	if taskToken.DomainID == "" {
		return wh.error(errDomainNotSet, scope)
	}

	domainEntry, err := wh.GetDomainCache().GetDomainByID(taskToken.DomainID)
	if err != nil {
		return wh.error(err, scope)
	}

	scope, sw := wh.startRequestProfileWithDomain(
		metrics.FrontendRespondDecisionTaskFailedScope,
		domainWrapper{
			domain: domainEntry.GetInfo().Name,
		},
	)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return errShuttingDown
	}

	if !wh.validIDLength(failedRequest.GetIdentity(), scope, domainEntry.GetInfo().Name) {
		return wh.error(errIdentityTooLong, scope)
	}

	sizeLimitError := wh.config.BlobSizeLimitError(domainEntry.GetInfo().Name)
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(domainEntry.GetInfo().Name)

	if err := common.CheckEventBlobSizeLimit(
		len(failedRequest.Details),
		sizeLimitWarn,
		sizeLimitError,
		taskToken.DomainID,
		taskToken.WorkflowID,
		taskToken.RunID,
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("RespondDecisionTaskFailed"),
	); err != nil {
		// details exceed, we would just truncate the size for decision task failed as the details is not used anywhere by client code
		failedRequest.Details = failedRequest.Details[0:sizeLimitError]
	}

	err = wh.GetHistoryClient().RespondDecisionTaskFailed(ctx, &types.HistoryRespondDecisionTaskFailedRequest{
		DomainUUID:    common.StringPtr(taskToken.DomainID),
		FailedRequest: thrift.ToRespondDecisionTaskFailedRequest(failedRequest),
	})
	err = thrift.FromError(err)
	if err != nil {
		return wh.error(err, scope)
	}
	return nil
}

// RespondQueryTaskCompleted - response to a query task
func (wh *WorkflowHandler) RespondQueryTaskCompleted(
	ctx context.Context,
	completeRequest *gen.RespondQueryTaskCompletedRequest,
) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope := wh.getDefaultScope(metrics.FrontendRespondQueryTaskCompletedScope)
	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope)
	}

	if completeRequest == nil {
		return wh.error(errRequestNotSet, scope)
	}

	// Count the request in the RPS, but we still accept it even if RPS is exceeded
	wh.allow(nil)

	if completeRequest.TaskToken == nil {
		return wh.error(errTaskTokenNotSet, scope)
	}
	queryTaskToken, err := wh.tokenSerializer.DeserializeQueryTaskToken(completeRequest.TaskToken)
	if err != nil {
		return wh.error(err, scope)
	}
	if queryTaskToken.DomainID == "" || queryTaskToken.TaskList == "" || queryTaskToken.TaskID == "" {
		return wh.error(errInvalidTaskToken, scope)
	}

	domainEntry, err := wh.GetDomainCache().GetDomainByID(queryTaskToken.DomainID)
	if err != nil {
		return wh.error(err, scope)
	}

	scope, sw := wh.startRequestProfileWithDomain(
		metrics.FrontendRespondQueryTaskCompletedScope,
		domainWrapper{
			domain: domainEntry.GetInfo().Name,
		},
	)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return errShuttingDown
	}

	sizeLimitError := wh.config.BlobSizeLimitError(domainEntry.GetInfo().Name)
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(domainEntry.GetInfo().Name)

	if err := common.CheckEventBlobSizeLimit(
		len(completeRequest.GetQueryResult()),
		sizeLimitWarn,
		sizeLimitError,
		queryTaskToken.DomainID,
		"",
		"",
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("RespondQueryTaskCompleted"),
	); err != nil {
		completeRequest = &gen.RespondQueryTaskCompletedRequest{
			TaskToken:     completeRequest.TaskToken,
			CompletedType: common.QueryTaskCompletedTypePtr(gen.QueryTaskCompletedTypeFailed),
			QueryResult:   nil,
			ErrorMessage:  common.StringPtr(err.Error()),
		}
	}

	call := yarpc.CallFromContext(ctx)

	completeRequest.WorkerVersionInfo = &gen.WorkerVersionInfo{
		Impl:           common.StringPtr(call.Header(common.ClientImplHeaderName)),
		FeatureVersion: common.StringPtr(call.Header(common.FeatureVersionHeaderName)),
	}
	matchingRequest := &m.RespondQueryTaskCompletedRequest{
		DomainUUID:       common.StringPtr(queryTaskToken.DomainID),
		TaskList:         &gen.TaskList{Name: common.StringPtr(queryTaskToken.TaskList)},
		TaskID:           common.StringPtr(queryTaskToken.TaskID),
		CompletedRequest: completeRequest,
	}

	err = wh.GetMatchingClient().RespondQueryTaskCompleted(ctx, thrift.ToMatchingRespondQueryTaskCompletedRequest(matchingRequest))
	err = thrift.FromError(err)
	if err != nil {
		return wh.error(err, scope)
	}
	return nil
}

// StartWorkflowExecution - Creates a new workflow execution
func (wh *WorkflowHandler) StartWorkflowExecution(
	ctx context.Context,
	startRequest *gen.StartWorkflowExecutionRequest,
) (resp *gen.StartWorkflowExecutionResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendStartWorkflowExecutionScope, startRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if startRequest == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	if ok := wh.allow(startRequest); !ok {
		return nil, wh.error(createServiceBusyError(), scope)
	}

	domainName := startRequest.GetDomain()
	if domainName == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}

	if !wh.validIDLength(domainName, scope, domainName) {
		return nil, wh.error(errDomainTooLong, scope)
	}

	if startRequest.GetWorkflowId() == "" {
		return nil, wh.error(errWorkflowIDNotSet, scope)
	}

	if !wh.validIDLength(startRequest.GetWorkflowId(), scope, domainName) {
		return nil, wh.error(errWorkflowIDTooLong, scope)
	}

	if err := common.ValidateRetryPolicy(startRequest.RetryPolicy); err != nil {
		return nil, wh.error(err, scope)
	}

	if err := backoff.ValidateSchedule(startRequest.GetCronSchedule()); err != nil {
		return nil, wh.error(err, scope)
	}

	wh.GetLogger().Debug(
		"Received StartWorkflowExecution. WorkflowID",
		tag.WorkflowID(startRequest.GetWorkflowId()))

	if startRequest.WorkflowType == nil || startRequest.WorkflowType.GetName() == "" {
		return nil, wh.error(errWorkflowTypeNotSet, scope)
	}

	if !wh.validIDLength(startRequest.WorkflowType.GetName(), scope, domainName) {
		return nil, wh.error(errWorkflowTypeTooLong, scope)
	}

	if err := wh.validateTaskList(startRequest.TaskList, scope, domainName); err != nil {
		return nil, err
	}

	if startRequest.GetExecutionStartToCloseTimeoutSeconds() <= 0 {
		return nil, wh.error(errInvalidExecutionStartToCloseTimeoutSeconds, scope)
	}

	if startRequest.GetTaskStartToCloseTimeoutSeconds() <= 0 {
		return nil, wh.error(errInvalidTaskStartToCloseTimeoutSeconds, scope)
	}

	if startRequest.GetRequestId() == "" {
		return nil, wh.error(errRequestIDNotSet, scope)
	}

	if !wh.validIDLength(startRequest.GetRequestId(), scope, domainName) {
		return nil, wh.error(errRequestIDTooLong, scope)
	}

	if err := wh.searchAttributesValidator.ValidateSearchAttributes(thrift.ToSearchAttributes(startRequest.SearchAttributes), domainName); err != nil {
		err = thrift.FromError(err)
		return nil, wh.error(err, scope)
	}

	wh.GetLogger().Debug("Start workflow execution request domain", tag.WorkflowDomainName(domainName))
	domainID, err := wh.GetDomainCache().GetDomainID(domainName)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	// add domain tag to scope, so further metrics will have the domain tag
	scope = scope.Tagged(metrics.DomainTag(domainName))

	sizeLimitError := wh.config.BlobSizeLimitError(domainName)
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(domainName)
	actualSize := len(startRequest.Input)
	if startRequest.Memo != nil {
		actualSize += common.GetSizeOfMapStringToByteArray(startRequest.Memo.GetFields())
	}
	if err := common.CheckEventBlobSizeLimit(
		actualSize,
		sizeLimitWarn,
		sizeLimitError,
		domainID,
		startRequest.GetWorkflowId(),
		"",
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("StartWorkflowExecution"),
	); err != nil {
		return nil, wh.error(err, scope)
	}

	wh.GetLogger().Debug("Start workflow execution request domainID", tag.WorkflowDomainID(domainID))
	clientResp, err := wh.GetHistoryClient().StartWorkflowExecution(ctx, thrift.ToHistoryStartWorkflowExecutionRequest(common.CreateHistoryStartWorkflowRequest(domainID, startRequest)))
	resp = thrift.FromStartWorkflowExecutionResponse(clientResp)
	err = thrift.FromError(err)
	if err != nil {
		return nil, wh.error(err, scope)
	}
	return resp, nil
}

// GetWorkflowExecutionHistory - retrieves the history of workflow execution
func (wh *WorkflowHandler) GetWorkflowExecutionHistory(
	ctx context.Context,
	getRequest *gen.GetWorkflowExecutionHistoryRequest,
) (resp *gen.GetWorkflowExecutionHistoryResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendGetWorkflowExecutionHistoryScope, getRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}
	wfExecution := getRequest.GetExecution()

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if getRequest == nil {
		return nil, wh.error(errRequestNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if ok := wh.allow(getRequest); !ok {
		return nil, wh.error(createServiceBusyError(), scope, getWfIDRunIDTags(wfExecution)...)
	}

	domainName := getRequest.GetDomain()
	if domainName == "" {
		return nil, wh.error(errDomainNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}
	domainID, err := wh.GetDomainCache().GetDomainID(domainName)
	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if err := wh.validateExecutionAndEmitMetrics(getRequest.Execution, scope); err != nil {
		return nil, err
	}

	if getRequest.GetMaximumPageSize() <= 0 {
		getRequest.MaximumPageSize = common.Int32Ptr(int32(wh.config.HistoryMaxPageSize(getRequest.GetDomain())))
	}
	// force limit page size if exceed
	if getRequest.GetMaximumPageSize() > common.GetHistoryMaxPageSize {
		wh.GetThrottledLogger().Warn("GetHistory page size is larger than threshold",
			tag.WorkflowID(getRequest.Execution.GetWorkflowId()),
			tag.WorkflowRunID(getRequest.Execution.GetRunId()),
			tag.WorkflowDomainID(domainID),
			tag.WorkflowSize(int64(getRequest.GetMaximumPageSize())))
		getRequest.MaximumPageSize = common.Int32Ptr(common.GetHistoryMaxPageSize)
	}

	if !getRequest.GetSkipArchival() {
		enableArchivalRead := wh.GetArchivalMetadata().GetHistoryConfig().ReadEnabled()
		historyArchived := wh.historyArchived(ctx, getRequest, domainID)
		if enableArchivalRead && historyArchived {
			return wh.getArchivedHistory(ctx, getRequest, domainID, scope)
		}
	}

	// this function return the following 6 things,
	// 1. branch token
	// 2. the workflow run ID
	// 3. the last first event ID (the event ID of the last batch of events in the history)
	// 4. the next event ID
	// 5. whether the workflow is closed
	// 6. error if any
	queryHistory := func(
		domainUUID string,
		execution *gen.WorkflowExecution,
		expectedNextEventID int64,
		currentBranchToken []byte,
	) ([]byte, string, int64, int64, bool, error) {
		clientResp, err := wh.GetHistoryClient().PollMutableState(ctx, &types.PollMutableStateRequest{
			DomainUUID:          common.StringPtr(domainUUID),
			Execution:           thrift.ToWorkflowExecution(execution),
			ExpectedNextEventID: common.Int64Ptr(expectedNextEventID),
			CurrentBranchToken:  currentBranchToken,
		})
		response := thrift.FromPollMutableStateResponse(clientResp)
		err = thrift.FromError(err)

		if err != nil {
			return nil, "", 0, 0, false, err
		}
		isWorkflowRunning := response.GetWorkflowCloseState() == persistence.WorkflowCloseStatusNone

		return response.CurrentBranchToken,
			response.Execution.GetRunId(),
			response.GetLastFirstEventId(),
			response.GetNextEventId(),
			isWorkflowRunning,
			nil
	}

	isLongPoll := getRequest.GetWaitForNewEvent()
	isCloseEventOnly := getRequest.GetHistoryEventFilterType() == gen.HistoryEventFilterTypeCloseEvent
	execution := getRequest.Execution
	token := &getHistoryContinuationToken{}

	var runID string
	lastFirstEventID := common.FirstEventID
	var nextEventID int64
	var isWorkflowRunning bool

	// process the token for paging
	queryNextEventID := common.EndEventID
	if getRequest.NextPageToken != nil {
		token, err = deserializeHistoryToken(getRequest.NextPageToken)
		if err != nil {
			return nil, wh.error(errInvalidNextPageToken, scope, getWfIDRunIDTags(wfExecution)...)
		}
		if execution.RunId != nil && execution.GetRunId() != token.RunID {
			return nil, wh.error(errNextPageTokenRunIDMismatch, scope, getWfIDRunIDTags(wfExecution)...)
		}

		execution.RunId = common.StringPtr(token.RunID)

		// we need to update the current next event ID and whether workflow is running
		if len(token.PersistenceToken) == 0 && isLongPoll && token.IsWorkflowRunning {
			if !isCloseEventOnly {
				queryNextEventID = token.NextEventID
			}
			token.BranchToken, _, lastFirstEventID, nextEventID, isWorkflowRunning, err =
				queryHistory(domainID, execution, queryNextEventID, token.BranchToken)
			if err != nil {
				return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
			}
			token.FirstEventID = token.NextEventID
			token.NextEventID = nextEventID
			token.IsWorkflowRunning = isWorkflowRunning
		}
	} else {
		if !isCloseEventOnly {
			queryNextEventID = common.FirstEventID
		}
		token.BranchToken, runID, lastFirstEventID, nextEventID, isWorkflowRunning, err =
			queryHistory(domainID, execution, queryNextEventID, nil)
		if err != nil {
			return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
		}

		execution.RunId = &runID

		token.RunID = runID
		token.FirstEventID = common.FirstEventID
		token.NextEventID = nextEventID
		token.IsWorkflowRunning = isWorkflowRunning
		token.PersistenceToken = nil
	}

	call := yarpc.CallFromContext(ctx)
	clientFeatureVersion := call.Header(common.FeatureVersionHeaderName)
	clientImpl := call.Header(common.ClientImplHeaderName)
	supportsRawHistoryQuery := wh.versionChecker.SupportsRawHistoryQuery(clientImpl, clientFeatureVersion) == nil
	isRawHistoryEnabled := wh.config.SendRawWorkflowHistory(domainName) && supportsRawHistoryQuery

	history := &gen.History{}
	history.Events = []*gen.HistoryEvent{}
	var historyBlob []*gen.DataBlob

	// helper function to just getHistory
	getHistory := func(firstEventID, nextEventID int64, nextPageToken []byte) error {
		if isRawHistoryEnabled {
			historyBlob, token.PersistenceToken, err = wh.getRawHistory(
				ctx,
				scope,
				domainID,
				*execution,
				firstEventID,
				nextEventID,
				getRequest.GetMaximumPageSize(),
				nextPageToken,
				token.TransientDecision,
				token.BranchToken,
			)
		} else {
			history, token.PersistenceToken, err = wh.getHistory(
				ctx,
				scope,
				domainID,
				*execution,
				firstEventID,
				nextEventID,
				getRequest.GetMaximumPageSize(),
				nextPageToken,
				token.TransientDecision,
				token.BranchToken,
			)
		}
		if err != nil {
			return err
		}
		return nil
	}

	if isCloseEventOnly {
		if !isWorkflowRunning {
			if err := getHistory(lastFirstEventID, nextEventID, nil); err != nil {
				return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
			}
			if isRawHistoryEnabled {
				// since getHistory func will not return empty history, so the below is safe
				historyBlob = historyBlob[len(historyBlob)-1:]
			} else {
				// since getHistory func will not return empty history, so the below is safe
				history.Events = history.Events[len(history.Events)-1:]
			}
			token = nil
		} else if isLongPoll {
			// set the persistence token to be nil so next time we will query history for updates
			token.PersistenceToken = nil
		} else {
			token = nil
		}
	} else {
		// return all events
		if token.FirstEventID >= token.NextEventID {
			// currently there is no new event
			history.Events = []*gen.HistoryEvent{}
			if !isWorkflowRunning {
				token = nil
			}
		} else {
			if err := getHistory(token.FirstEventID, token.NextEventID, token.PersistenceToken); err != nil {
				return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
			}
			// here, for long pull on history events, we need to intercept the paging token from cassandra
			// and do something clever
			if len(token.PersistenceToken) == 0 && (!token.IsWorkflowRunning || !isLongPoll) {
				// meaning, there is no more history to be returned
				token = nil
			}
		}
	}

	nextToken, err := serializeHistoryToken(token)
	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}
	return &gen.GetWorkflowExecutionHistoryResponse{
		History:       history,
		RawHistory:    historyBlob,
		NextPageToken: nextToken,
		Archived:      common.BoolPtr(false),
	}, nil
}

// SignalWorkflowExecution is used to send a signal event to running workflow execution.  This results in
// WorkflowExecutionSignaled event recorded in the history and a decision task being created for the execution.
func (wh *WorkflowHandler) SignalWorkflowExecution(
	ctx context.Context,
	signalRequest *gen.SignalWorkflowExecutionRequest,
) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendSignalWorkflowExecutionScope, signalRequest)
	defer sw.Stop()

	wfExecution := signalRequest.GetWorkflowExecution()

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if signalRequest == nil {
		return wh.error(errRequestNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if ok := wh.allow(signalRequest); !ok {
		return wh.error(createServiceBusyError(), scope, getWfIDRunIDTags(wfExecution)...)
	}

	if signalRequest.GetDomain() == "" {
		return wh.error(errDomainNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if !wh.validIDLength(signalRequest.GetDomain(), scope, signalRequest.GetDomain()) {
		return wh.error(errDomainTooLong, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if err := wh.validateExecutionAndEmitMetrics(signalRequest.WorkflowExecution, scope); err != nil {
		return err
	}

	if signalRequest.GetSignalName() == "" {
		return wh.error(&gen.BadRequestError{Message: "SignalName is not set on request."},
			scope, getWfIDRunIDTags(wfExecution)...)
	}

	if !wh.validIDLength(signalRequest.GetSignalName(), scope, signalRequest.GetDomain()) {
		return wh.error(errSignalNameTooLong, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if !wh.validIDLength(signalRequest.GetRequestId(), scope, signalRequest.GetDomain()) {
		return wh.error(errRequestIDTooLong, scope, getWfIDRunIDTags(wfExecution)...)
	}

	domainID, err := wh.GetDomainCache().GetDomainID(signalRequest.GetDomain())
	if err != nil {
		return wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	sizeLimitError := wh.config.BlobSizeLimitError(signalRequest.GetDomain())
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(signalRequest.GetDomain())
	if err := common.CheckEventBlobSizeLimit(
		len(signalRequest.Input),
		sizeLimitWarn,
		sizeLimitError,
		domainID,
		signalRequest.GetWorkflowExecution().GetWorkflowId(),
		signalRequest.GetWorkflowExecution().GetRunId(),
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("SignalWorkflowExecution"),
	); err != nil {
		return wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	err = wh.GetHistoryClient().SignalWorkflowExecution(ctx, &types.HistorySignalWorkflowExecutionRequest{
		DomainUUID:    common.StringPtr(domainID),
		SignalRequest: thrift.ToSignalWorkflowExecutionRequest(signalRequest),
	})
	err = thrift.FromError(err)
	if err != nil {
		return wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	return nil
}

// SignalWithStartWorkflowExecution is used to ensure sending a signal event to a workflow execution.
// If workflow is running, this results in WorkflowExecutionSignaled event recorded in the history
// and a decision task being created for the execution.
// If workflow is not running or not found, this results in WorkflowExecutionStarted and WorkflowExecutionSignaled
// event recorded in history, and a decision task being created for the execution
func (wh *WorkflowHandler) SignalWithStartWorkflowExecution(
	ctx context.Context,
	signalWithStartRequest *gen.SignalWithStartWorkflowExecutionRequest,
) (resp *gen.StartWorkflowExecutionResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendSignalWithStartWorkflowExecutionScope, signalWithStartRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	wfExecution := &gen.WorkflowExecution{
		WorkflowId: signalWithStartRequest.WorkflowId,
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if signalWithStartRequest == nil {
		return nil, wh.error(errRequestNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if ok := wh.allow(signalWithStartRequest); !ok {
		return nil, wh.error(createServiceBusyError(), scope, getWfIDRunIDTags(wfExecution)...)
	}

	domainName := signalWithStartRequest.GetDomain()
	if domainName == "" {
		return nil, wh.error(errDomainNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if !wh.validIDLength(domainName, scope, domainName) {
		return nil, wh.error(errDomainTooLong, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if signalWithStartRequest.GetWorkflowId() == "" {
		return nil, wh.error(&gen.BadRequestError{Message: "WorkflowId is not set on request."},
			scope, getWfIDRunIDTags(wfExecution)...)
	}

	if !wh.validIDLength(signalWithStartRequest.GetWorkflowId(), scope, domainName) {
		return nil, wh.error(errWorkflowIDTooLong, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if signalWithStartRequest.GetSignalName() == "" {
		return nil, wh.error(&gen.BadRequestError{Message: "SignalName is not set on request."},
			scope, getWfIDRunIDTags(wfExecution)...)
	}

	if !wh.validIDLength(signalWithStartRequest.GetSignalName(), scope, domainName) {
		return nil, wh.error(errSignalNameTooLong, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if signalWithStartRequest.WorkflowType == nil || signalWithStartRequest.WorkflowType.GetName() == "" {
		return nil, wh.error(&gen.BadRequestError{Message: "WorkflowType is not set on request."},
			scope, getWfIDRunIDTags(wfExecution)...)
	}

	if !wh.validIDLength(signalWithStartRequest.WorkflowType.GetName(), scope, domainName) {
		return nil, wh.error(errWorkflowTypeTooLong, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if err := wh.validateTaskList(signalWithStartRequest.TaskList, scope, domainName); err != nil {
		return nil, err
	}

	if !wh.validIDLength(signalWithStartRequest.GetRequestId(), scope, domainName) {
		return nil, wh.error(errRequestIDTooLong, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if signalWithStartRequest.GetExecutionStartToCloseTimeoutSeconds() <= 0 {
		return nil, wh.error(&gen.BadRequestError{
			Message: "A valid ExecutionStartToCloseTimeoutSeconds is not set on request."},
			scope, getWfIDRunIDTags(wfExecution)...)
	}

	if signalWithStartRequest.GetTaskStartToCloseTimeoutSeconds() <= 0 {
		return nil, wh.error(&gen.BadRequestError{
			Message: "A valid TaskStartToCloseTimeoutSeconds is not set on request."},
			scope, getWfIDRunIDTags(wfExecution)...)
	}

	if err := common.ValidateRetryPolicy(signalWithStartRequest.RetryPolicy); err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if err := backoff.ValidateSchedule(signalWithStartRequest.GetCronSchedule()); err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if err := wh.searchAttributesValidator.ValidateSearchAttributes(thrift.ToSearchAttributes(signalWithStartRequest.SearchAttributes), domainName); err != nil {
		err = thrift.FromError(err)
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	domainID, err := wh.GetDomainCache().GetDomainID(domainName)
	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	sizeLimitError := wh.config.BlobSizeLimitError(domainName)
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(domainName)
	if err := common.CheckEventBlobSizeLimit(
		len(signalWithStartRequest.SignalInput),
		sizeLimitWarn,
		sizeLimitError,
		domainID,
		signalWithStartRequest.GetWorkflowId(),
		"",
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("SignalWithStartWorkflowExecution"),
	); err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}
	actualSize := len(signalWithStartRequest.Input) + common.GetSizeOfMapStringToByteArray(signalWithStartRequest.Memo.GetFields())
	if err := common.CheckEventBlobSizeLimit(
		actualSize,
		sizeLimitWarn,
		sizeLimitError,
		domainID,
		signalWithStartRequest.GetWorkflowId(),
		"",
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("SignalWithStartWorkflowExecution"),
	); err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	clientResp, err := wh.GetHistoryClient().SignalWithStartWorkflowExecution(ctx, &types.HistorySignalWithStartWorkflowExecutionRequest{
		DomainUUID:             common.StringPtr(domainID),
		SignalWithStartRequest: thrift.ToSignalWithStartWorkflowExecutionRequest(signalWithStartRequest),
	})
	resp = thrift.FromStartWorkflowExecutionResponse(clientResp)
	err = thrift.FromError(err)
	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	return resp, nil
}

// TerminateWorkflowExecution terminates an existing workflow execution by recording WorkflowExecutionTerminated event
// in the history and immediately terminating the execution instance.
func (wh *WorkflowHandler) TerminateWorkflowExecution(
	ctx context.Context,
	terminateRequest *gen.TerminateWorkflowExecutionRequest,
) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendTerminateWorkflowExecutionScope, terminateRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return errShuttingDown
	}

	wfExecution := terminateRequest.GetWorkflowExecution()

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if terminateRequest == nil {
		return wh.error(errRequestNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if ok := wh.allow(terminateRequest); !ok {
		return wh.error(createServiceBusyError(), scope, getWfIDRunIDTags(wfExecution)...)
	}

	if terminateRequest.GetDomain() == "" {
		return wh.error(errDomainNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if err := wh.validateExecutionAndEmitMetrics(terminateRequest.WorkflowExecution, scope); err != nil {
		return err
	}

	domainID, err := wh.GetDomainCache().GetDomainID(terminateRequest.GetDomain())
	if err != nil {
		return wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	err = wh.GetHistoryClient().TerminateWorkflowExecution(ctx, &types.HistoryTerminateWorkflowExecutionRequest{
		DomainUUID:       common.StringPtr(domainID),
		TerminateRequest: thrift.ToTerminateWorkflowExecutionRequest(terminateRequest),
	})
	err = thrift.FromError(err)
	if err != nil {
		return wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	return nil
}

// ResetWorkflowExecution reset an existing workflow execution to the nextFirstEventID
// in the history and immediately terminating the current execution instance.
func (wh *WorkflowHandler) ResetWorkflowExecution(
	ctx context.Context,
	resetRequest *gen.ResetWorkflowExecutionRequest,
) (resp *gen.ResetWorkflowExecutionResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendResetWorkflowExecutionScope, resetRequest)
	defer sw.Stop()

	wfExecution := resetRequest.GetWorkflowExecution()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if resetRequest == nil {
		return nil, wh.error(errRequestNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if ok := wh.allow(resetRequest); !ok {
		return nil, wh.error(createServiceBusyError(), scope, getWfIDRunIDTags(wfExecution)...)
	}

	if resetRequest.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if err := wh.validateExecutionAndEmitMetrics(resetRequest.WorkflowExecution, scope); err != nil {
		return nil, err
	}

	domainID, err := wh.GetDomainCache().GetDomainID(resetRequest.GetDomain())
	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	clientResp, err := wh.GetHistoryClient().ResetWorkflowExecution(ctx, &types.HistoryResetWorkflowExecutionRequest{
		DomainUUID:   common.StringPtr(domainID),
		ResetRequest: thrift.ToResetWorkflowExecutionRequest(resetRequest),
	})
	resp = thrift.FromResetWorkflowExecutionResponse(clientResp)
	err = thrift.FromError(err)
	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	return resp, nil
}

// RequestCancelWorkflowExecution - requests to cancel a workflow execution
func (wh *WorkflowHandler) RequestCancelWorkflowExecution(
	ctx context.Context,
	cancelRequest *gen.RequestCancelWorkflowExecutionRequest,
) (retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendRequestCancelWorkflowExecutionScope, cancelRequest)
	defer sw.Stop()

	wfExecution := cancelRequest.GetWorkflowExecution()

	if wh.isShuttingDown() {
		return errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if cancelRequest == nil {
		return wh.error(errRequestNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if ok := wh.allow(cancelRequest); !ok {
		return wh.error(createServiceBusyError(), scope, getWfIDRunIDTags(wfExecution)...)
	}

	if cancelRequest.GetDomain() == "" {
		return wh.error(errDomainNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if err := wh.validateExecutionAndEmitMetrics(cancelRequest.WorkflowExecution, scope); err != nil {
		return err
	}

	domainID, err := wh.GetDomainCache().GetDomainID(cancelRequest.GetDomain())
	if err != nil {
		return wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	err = wh.GetHistoryClient().RequestCancelWorkflowExecution(ctx, &types.HistoryRequestCancelWorkflowExecutionRequest{
		DomainUUID:    common.StringPtr(domainID),
		CancelRequest: thrift.ToRequestCancelWorkflowExecutionRequest(cancelRequest),
	})
	err = thrift.FromError(err)
	if err != nil {
		return wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	return nil
}

// ListOpenWorkflowExecutions - retrieves info for open workflow executions in a domain
func (wh *WorkflowHandler) ListOpenWorkflowExecutions(
	ctx context.Context,
	listRequest *gen.ListOpenWorkflowExecutionsRequest,
) (resp *gen.ListOpenWorkflowExecutionsResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendListOpenWorkflowExecutionsScope, listRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if listRequest == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	if ok := wh.allow(listRequest); !ok {
		return nil, wh.error(createServiceBusyError(), scope)
	}

	if listRequest.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}

	if listRequest.StartTimeFilter == nil {
		return nil, wh.error(&gen.BadRequestError{Message: "StartTimeFilter is required"}, scope)
	}

	if listRequest.StartTimeFilter.EarliestTime == nil {
		return nil, wh.error(&gen.BadRequestError{Message: "EarliestTime in StartTimeFilter is required"}, scope)
	}

	if listRequest.StartTimeFilter.LatestTime == nil {
		return nil, wh.error(&gen.BadRequestError{Message: "LatestTime in StartTimeFilter is required"}, scope)
	}

	if listRequest.StartTimeFilter.GetEarliestTime() > listRequest.StartTimeFilter.GetLatestTime() {
		return nil, wh.error(&gen.BadRequestError{Message: "EarliestTime in StartTimeFilter should not be larger than LatestTime"}, scope)
	}

	if listRequest.ExecutionFilter != nil && listRequest.TypeFilter != nil {
		return nil, wh.error(&gen.BadRequestError{
			Message: "Only one of ExecutionFilter or TypeFilter is allowed"}, scope)
	}

	if listRequest.GetMaximumPageSize() <= 0 {
		listRequest.MaximumPageSize = common.Int32Ptr(int32(wh.config.VisibilityMaxPageSize(listRequest.GetDomain())))
	}

	if wh.isListRequestPageSizeTooLarge(listRequest.GetMaximumPageSize(), listRequest.GetDomain()) {
		return nil, wh.error(&gen.BadRequestError{
			Message: fmt.Sprintf("Pagesize is larger than allow %d", wh.config.ESIndexMaxResultWindow())}, scope)
	}

	domain := listRequest.GetDomain()
	domainID, err := wh.GetDomainCache().GetDomainID(domain)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	baseReq := persistence.ListWorkflowExecutionsRequest{
		DomainUUID:    domainID,
		Domain:        domain,
		PageSize:      int(listRequest.GetMaximumPageSize()),
		NextPageToken: listRequest.NextPageToken,
		EarliestTime:  listRequest.StartTimeFilter.GetEarliestTime(),
		LatestTime:    listRequest.StartTimeFilter.GetLatestTime(),
	}

	var persistenceResp *persistence.ListWorkflowExecutionsResponse
	if listRequest.ExecutionFilter != nil {
		if wh.config.DisableListVisibilityByFilter(domain) {
			err = errNoPermission
		} else {
			persistenceResp, err = wh.GetVisibilityManager().ListOpenWorkflowExecutionsByWorkflowID(
				ctx,
				&persistence.ListWorkflowExecutionsByWorkflowIDRequest{
					ListWorkflowExecutionsRequest: baseReq,
					WorkflowID:                    listRequest.ExecutionFilter.GetWorkflowId(),
				})
		}
		wh.GetLogger().Debug("List open workflow with filter",
			tag.WorkflowDomainName(listRequest.GetDomain()), tag.WorkflowListWorkflowFilterByID)
	} else if listRequest.TypeFilter != nil {
		if wh.config.DisableListVisibilityByFilter(domain) {
			err = errNoPermission
		} else {
			persistenceResp, err = wh.GetVisibilityManager().ListOpenWorkflowExecutionsByType(
				ctx,
				&persistence.ListWorkflowExecutionsByTypeRequest{
					ListWorkflowExecutionsRequest: baseReq,
					WorkflowTypeName:              listRequest.TypeFilter.GetName(),
				},
			)
		}
		wh.GetLogger().Debug("List open workflow with filter",
			tag.WorkflowDomainName(listRequest.GetDomain()), tag.WorkflowListWorkflowFilterByType)
	} else {
		persistenceResp, err = wh.GetVisibilityManager().ListOpenWorkflowExecutions(ctx, &baseReq)
	}

	if err != nil {
		return nil, wh.error(err, scope)
	}

	resp = &gen.ListOpenWorkflowExecutionsResponse{}
	resp.Executions = persistenceResp.Executions
	resp.NextPageToken = persistenceResp.NextPageToken
	return resp, nil
}

// ListArchivedWorkflowExecutions - retrieves archived info for closed workflow executions in a domain
func (wh *WorkflowHandler) ListArchivedWorkflowExecutions(
	ctx context.Context,
	listRequest *gen.ListArchivedWorkflowExecutionsRequest,
) (resp *gen.ListArchivedWorkflowExecutionsResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendListArchivedWorkflowExecutionsScope, listRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if listRequest == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	if ok := wh.allow(listRequest); !ok {
		return nil, wh.error(createServiceBusyError(), scope)
	}

	if listRequest.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}

	if listRequest.GetPageSize() <= 0 {
		listRequest.PageSize = common.Int32Ptr(int32(wh.config.VisibilityMaxPageSize(listRequest.GetDomain())))
	}

	maxPageSize := wh.config.VisibilityArchivalQueryMaxPageSize()
	if int(listRequest.GetPageSize()) > maxPageSize {
		return nil, wh.error(&gen.BadRequestError{
			Message: fmt.Sprintf("Pagesize is larger than allowed %d", maxPageSize)}, scope)
	}

	if !wh.GetArchivalMetadata().GetVisibilityConfig().ClusterConfiguredForArchival() {
		return nil, wh.error(&gen.BadRequestError{Message: "Cluster is not configured for visibility archival"}, scope)
	}

	if !wh.GetArchivalMetadata().GetVisibilityConfig().ReadEnabled() {
		return nil, wh.error(&gen.BadRequestError{Message: "Cluster is not configured for reading archived visibility records"}, scope)
	}

	entry, err := wh.GetDomainCache().GetDomain(listRequest.GetDomain())
	if err != nil {
		return nil, wh.error(err, scope)
	}

	if entry.GetConfig().VisibilityArchivalStatus != gen.ArchivalStatusEnabled {
		return nil, wh.error(&gen.BadRequestError{Message: "Domain is not configured for visibility archival"}, scope)
	}

	URI, err := archiver.NewURI(entry.GetConfig().VisibilityArchivalURI)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	visibilityArchiver, err := wh.GetArchiverProvider().GetVisibilityArchiver(URI.Scheme(), common.FrontendServiceName)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	archiverRequest := &archiver.QueryVisibilityRequest{
		DomainID:      entry.GetInfo().ID,
		PageSize:      int(listRequest.GetPageSize()),
		NextPageToken: listRequest.NextPageToken,
		Query:         listRequest.GetQuery(),
	}

	archiverResponse, err := visibilityArchiver.Query(ctx, URI, archiverRequest)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	// special handling of ExecutionTime for cron or retry
	for _, execution := range archiverResponse.Executions {
		if execution.GetExecutionTime() == 0 {
			execution.ExecutionTime = common.Int64Ptr(execution.GetStartTime())
		}
	}

	return &gen.ListArchivedWorkflowExecutionsResponse{
		Executions:    archiverResponse.Executions,
		NextPageToken: archiverResponse.NextPageToken,
	}, nil
}

// ListClosedWorkflowExecutions - retrieves info for closed workflow executions in a domain
func (wh *WorkflowHandler) ListClosedWorkflowExecutions(
	ctx context.Context,
	listRequest *gen.ListClosedWorkflowExecutionsRequest,
) (resp *gen.ListClosedWorkflowExecutionsResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendListClosedWorkflowExecutionsScope, listRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if listRequest == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	if ok := wh.allow(listRequest); !ok {
		return nil, wh.error(createServiceBusyError(), scope)
	}

	if listRequest.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}

	if listRequest.StartTimeFilter == nil {
		return nil, wh.error(&gen.BadRequestError{Message: "StartTimeFilter is required"}, scope)
	}

	if listRequest.StartTimeFilter.EarliestTime == nil {
		return nil, wh.error(&gen.BadRequestError{Message: "EarliestTime in StartTimeFilter is required"}, scope)
	}

	if listRequest.StartTimeFilter.LatestTime == nil {
		return nil, wh.error(&gen.BadRequestError{Message: "LatestTime in StartTimeFilter is required"}, scope)
	}

	if listRequest.StartTimeFilter.GetEarliestTime() > listRequest.StartTimeFilter.GetLatestTime() {
		return nil, wh.error(&gen.BadRequestError{Message: "EarliestTime in StartTimeFilter should not be larger than LatestTime"}, scope)
	}

	filterCount := 0
	if listRequest.TypeFilter != nil {
		filterCount++
	}
	if listRequest.StatusFilter != nil {
		filterCount++
	}

	if filterCount > 1 {
		return nil, wh.error(&gen.BadRequestError{
			Message: "Only one of ExecutionFilter, TypeFilter or StatusFilter is allowed"}, scope)
	} // If ExecutionFilter is provided with one of TypeFilter or StatusFilter, use ExecutionFilter and ignore other filter

	if listRequest.GetMaximumPageSize() <= 0 {
		listRequest.MaximumPageSize = common.Int32Ptr(int32(wh.config.VisibilityMaxPageSize(listRequest.GetDomain())))
	}

	if wh.isListRequestPageSizeTooLarge(listRequest.GetMaximumPageSize(), listRequest.GetDomain()) {
		return nil, wh.error(&gen.BadRequestError{
			Message: fmt.Sprintf("Pagesize is larger than allow %d", wh.config.ESIndexMaxResultWindow())}, scope)
	}

	domain := listRequest.GetDomain()
	domainID, err := wh.GetDomainCache().GetDomainID(domain)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	baseReq := persistence.ListWorkflowExecutionsRequest{
		DomainUUID:    domainID,
		Domain:        domain,
		PageSize:      int(listRequest.GetMaximumPageSize()),
		NextPageToken: listRequest.NextPageToken,
		EarliestTime:  listRequest.StartTimeFilter.GetEarliestTime(),
		LatestTime:    listRequest.StartTimeFilter.GetLatestTime(),
	}

	var persistenceResp *persistence.ListWorkflowExecutionsResponse
	if listRequest.ExecutionFilter != nil {
		if wh.config.DisableListVisibilityByFilter(domain) {
			err = errNoPermission
		} else {
			persistenceResp, err = wh.GetVisibilityManager().ListClosedWorkflowExecutionsByWorkflowID(
				ctx,
				&persistence.ListWorkflowExecutionsByWorkflowIDRequest{
					ListWorkflowExecutionsRequest: baseReq,
					WorkflowID:                    listRequest.ExecutionFilter.GetWorkflowId(),
				},
			)
		}
		wh.GetLogger().Debug("List closed workflow with filter",
			tag.WorkflowDomainName(listRequest.GetDomain()), tag.WorkflowListWorkflowFilterByID)
	} else if listRequest.TypeFilter != nil {
		if wh.config.DisableListVisibilityByFilter(domain) {
			err = errNoPermission
		} else {
			persistenceResp, err = wh.GetVisibilityManager().ListClosedWorkflowExecutionsByType(
				ctx,
				&persistence.ListWorkflowExecutionsByTypeRequest{
					ListWorkflowExecutionsRequest: baseReq,
					WorkflowTypeName:              listRequest.TypeFilter.GetName(),
				},
			)
		}
		wh.GetLogger().Debug("List closed workflow with filter",
			tag.WorkflowDomainName(listRequest.GetDomain()), tag.WorkflowListWorkflowFilterByType)
	} else if listRequest.StatusFilter != nil {
		if wh.config.DisableListVisibilityByFilter(domain) {
			err = errNoPermission
		} else {
			persistenceResp, err = wh.GetVisibilityManager().ListClosedWorkflowExecutionsByStatus(
				ctx,
				&persistence.ListClosedWorkflowExecutionsByStatusRequest{
					ListWorkflowExecutionsRequest: baseReq,
					Status:                        listRequest.GetStatusFilter(),
				},
			)
		}
		wh.GetLogger().Debug("List closed workflow with filter",
			tag.WorkflowDomainName(listRequest.GetDomain()), tag.WorkflowListWorkflowFilterByStatus)
	} else {
		persistenceResp, err = wh.GetVisibilityManager().ListClosedWorkflowExecutions(ctx, &baseReq)
	}

	if err != nil {
		return nil, wh.error(err, scope)
	}

	resp = &gen.ListClosedWorkflowExecutionsResponse{}
	resp.Executions = persistenceResp.Executions
	resp.NextPageToken = persistenceResp.NextPageToken
	return resp, nil
}

// ListWorkflowExecutions - retrieves info for workflow executions in a domain
func (wh *WorkflowHandler) ListWorkflowExecutions(
	ctx context.Context,
	listRequest *gen.ListWorkflowExecutionsRequest,
) (resp *gen.ListWorkflowExecutionsResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendListWorkflowExecutionsScope, listRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if listRequest == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	if ok := wh.allow(listRequest); !ok {
		return nil, wh.error(createServiceBusyError(), scope)
	}

	if listRequest.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}

	if listRequest.GetPageSize() <= 0 {
		listRequest.PageSize = common.Int32Ptr(int32(wh.config.VisibilityMaxPageSize(listRequest.GetDomain())))
	}

	if wh.isListRequestPageSizeTooLarge(listRequest.GetPageSize(), listRequest.GetDomain()) {
		return nil, wh.error(&gen.BadRequestError{
			Message: fmt.Sprintf("Pagesize is larger than allow %d", wh.config.ESIndexMaxResultWindow())}, scope)
	}

	validatedQuery, err := wh.visibilityQueryValidator.ValidateQuery(listRequest.GetQuery())
	if err != nil {
		err = thrift.FromError(err)
		return nil, wh.error(err, scope)
	}

	domain := listRequest.GetDomain()
	domainID, err := wh.GetDomainCache().GetDomainID(domain)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	req := &persistence.ListWorkflowExecutionsByQueryRequest{
		DomainUUID:    domainID,
		Domain:        domain,
		PageSize:      int(listRequest.GetPageSize()),
		NextPageToken: listRequest.NextPageToken,
		Query:         validatedQuery,
	}
	persistenceResp, err := wh.GetVisibilityManager().ListWorkflowExecutions(ctx, req)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	resp = &gen.ListWorkflowExecutionsResponse{}
	resp.Executions = persistenceResp.Executions
	resp.NextPageToken = persistenceResp.NextPageToken
	return resp, nil
}

// ScanWorkflowExecutions - retrieves info for large amount of workflow executions in a domain without order
func (wh *WorkflowHandler) ScanWorkflowExecutions(
	ctx context.Context,
	listRequest *gen.ListWorkflowExecutionsRequest,
) (resp *gen.ListWorkflowExecutionsResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendScanWorkflowExecutionsScope, listRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if listRequest == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	if ok := wh.allow(listRequest); !ok {
		return nil, wh.error(createServiceBusyError(), scope)
	}

	if listRequest.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}

	if listRequest.GetPageSize() <= 0 {
		listRequest.PageSize = common.Int32Ptr(int32(wh.config.VisibilityMaxPageSize(listRequest.GetDomain())))
	}

	if wh.isListRequestPageSizeTooLarge(listRequest.GetPageSize(), listRequest.GetDomain()) {
		return nil, wh.error(&gen.BadRequestError{
			Message: fmt.Sprintf("Pagesize is larger than allow %d", wh.config.ESIndexMaxResultWindow())}, scope)
	}

	validatedQuery, err := wh.visibilityQueryValidator.ValidateQuery(listRequest.GetQuery())
	if err != nil {
		err = thrift.FromError(err)
		return nil, wh.error(err, scope)
	}

	domain := listRequest.GetDomain()
	domainID, err := wh.GetDomainCache().GetDomainID(domain)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	req := &persistence.ListWorkflowExecutionsByQueryRequest{
		DomainUUID:    domainID,
		Domain:        domain,
		PageSize:      int(listRequest.GetPageSize()),
		NextPageToken: listRequest.NextPageToken,
		Query:         validatedQuery,
	}
	persistenceResp, err := wh.GetVisibilityManager().ScanWorkflowExecutions(ctx, req)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	resp = &gen.ListWorkflowExecutionsResponse{}
	resp.Executions = persistenceResp.Executions
	resp.NextPageToken = persistenceResp.NextPageToken
	return resp, nil
}

// CountWorkflowExecutions - count number of workflow executions in a domain
func (wh *WorkflowHandler) CountWorkflowExecutions(
	ctx context.Context,
	countRequest *gen.CountWorkflowExecutionsRequest,
) (resp *gen.CountWorkflowExecutionsResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendCountWorkflowExecutionsScope, countRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if countRequest == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	if ok := wh.allow(countRequest); !ok {
		return nil, wh.error(createServiceBusyError(), scope)
	}

	if countRequest.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}

	validatedQuery, err := wh.visibilityQueryValidator.ValidateQuery(countRequest.GetQuery())
	if err != nil {
		err = thrift.FromError(err)
		return nil, wh.error(err, scope)
	}

	domain := countRequest.GetDomain()
	domainID, err := wh.GetDomainCache().GetDomainID(domain)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	req := &persistence.CountWorkflowExecutionsRequest{
		DomainUUID: domainID,
		Domain:     domain,
		Query:      validatedQuery,
	}
	persistenceResp, err := wh.GetVisibilityManager().CountWorkflowExecutions(ctx, req)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	resp = &gen.CountWorkflowExecutionsResponse{
		Count: common.Int64Ptr(persistenceResp.Count),
	}
	return resp, nil
}

// GetSearchAttributes return valid indexed keys
func (wh *WorkflowHandler) GetSearchAttributes(ctx context.Context) (resp *gen.GetSearchAttributesResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfile(metrics.FrontendGetSearchAttributesScope)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	keys := wh.config.ValidSearchAttributes()
	resp = &gen.GetSearchAttributesResponse{
		Keys: wh.convertIndexedKeyToThrift(keys),
	}
	return resp, nil
}

// ResetStickyTaskList reset the volatile information in mutable state of a given workflow.
func (wh *WorkflowHandler) ResetStickyTaskList(
	ctx context.Context,
	resetRequest *gen.ResetStickyTaskListRequest,
) (resp *gen.ResetStickyTaskListResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendResetStickyTaskListScope, resetRequest)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if resetRequest == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	if resetRequest.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}

	if err := wh.validateExecutionAndEmitMetrics(resetRequest.Execution, scope); err != nil {
		return nil, err
	}

	domainID, err := wh.GetDomainCache().GetDomainID(resetRequest.GetDomain())
	if err != nil {
		return nil, wh.error(err, scope)
	}

	_, err = wh.GetHistoryClient().ResetStickyTaskList(ctx, &types.HistoryResetStickyTaskListRequest{
		DomainUUID: common.StringPtr(domainID),
		Execution:  thrift.ToWorkflowExecution(resetRequest.Execution),
	})
	err = thrift.FromError(err)
	if err != nil {
		return nil, wh.error(err, scope)
	}
	return &gen.ResetStickyTaskListResponse{}, nil
}

// QueryWorkflow returns query result for a specified workflow execution
func (wh *WorkflowHandler) QueryWorkflow(
	ctx context.Context,
	queryRequest *gen.QueryWorkflowRequest,
) (resp *gen.QueryWorkflowResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendQueryWorkflowScope, queryRequest)
	defer sw.Stop()

	wfExecution := queryRequest.GetExecution()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if wh.config.DisallowQuery(queryRequest.GetDomain()) {
		return nil, wh.error(errQueryDisallowedForDomain, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if ok := wh.allow(queryRequest); !ok {
		return nil, wh.error(createServiceBusyError(), scope)
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if queryRequest == nil {
		return nil, wh.error(errRequestNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if queryRequest.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}
	if err := wh.validateExecutionAndEmitMetrics(queryRequest.Execution, scope); err != nil {
		return nil, err
	}

	if queryRequest.Query == nil {
		return nil, wh.error(errQueryNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if queryRequest.Query.GetQueryType() == "" {
		return nil, wh.error(errQueryTypeNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	domainID, err := wh.GetDomainCache().GetDomainID(queryRequest.GetDomain())
	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	sizeLimitError := wh.config.BlobSizeLimitError(queryRequest.GetDomain())
	sizeLimitWarn := wh.config.BlobSizeLimitWarn(queryRequest.GetDomain())

	if err := common.CheckEventBlobSizeLimit(
		len(queryRequest.GetQuery().GetQueryArgs()),
		sizeLimitWarn,
		sizeLimitError,
		domainID,
		queryRequest.GetExecution().GetWorkflowId(),
		queryRequest.GetExecution().GetRunId(),
		scope,
		wh.GetThrottledLogger(),
		tag.BlobSizeViolationOperation("QueryWorkflow")); err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	req := &types.HistoryQueryWorkflowRequest{
		DomainUUID: common.StringPtr(domainID),
		Request:    thrift.ToQueryWorkflowRequest(queryRequest),
	}
	clientResp, err := wh.GetHistoryClient().QueryWorkflow(ctx, req)
	hResponse := thrift.FromHistoryQueryWorkflowResponse(clientResp)
	err = thrift.FromError(err)
	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}
	return hResponse.GetResponse(), nil
}

// DescribeWorkflowExecution returns information about the specified workflow execution.
func (wh *WorkflowHandler) DescribeWorkflowExecution(
	ctx context.Context,
	request *gen.DescribeWorkflowExecutionRequest,
) (resp *gen.DescribeWorkflowExecutionResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendDescribeWorkflowExecutionScope, request)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	wfExecution := request.GetExecution()

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if request == nil {
		return nil, wh.error(errRequestNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if ok := wh.allow(request); !ok {
		return nil, wh.error(createServiceBusyError(), scope, getWfIDRunIDTags(wfExecution)...)
	}

	if request.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope, getWfIDRunIDTags(wfExecution)...)
	}
	domainID, err := wh.GetDomainCache().GetDomainID(request.GetDomain())
	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	if err := wh.validateExecutionAndEmitMetrics(request.Execution, scope); err != nil {
		return nil, err
	}

	clientResp, err := wh.GetHistoryClient().DescribeWorkflowExecution(ctx, &types.HistoryDescribeWorkflowExecutionRequest{
		DomainUUID: common.StringPtr(domainID),
		Request:    thrift.ToDescribeWorkflowExecutionRequest(request),
	})
	response := thrift.FromDescribeWorkflowExecutionResponse(clientResp)
	err = thrift.FromError(err)

	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	return response, nil
}

// DescribeTaskList returns information about the target tasklist, right now this API returns the
// pollers which polled this tasklist in last few minutes. If includeTaskListStatus field is true,
// it will also return status of tasklist's ackManager (readLevel, ackLevel, backlogCountHint and taskIDBlock).
func (wh *WorkflowHandler) DescribeTaskList(
	ctx context.Context,
	request *gen.DescribeTaskListRequest,
) (resp *gen.DescribeTaskListResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendDescribeTaskListScope, request)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if err := wh.versionChecker.ClientSupported(ctx, wh.config.EnableClientVersionCheck()); err != nil {
		return nil, wh.error(err, scope)
	}

	if request == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	if ok := wh.allow(request); !ok {
		return nil, wh.error(createServiceBusyError(), scope)
	}

	if request.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}
	domainID, err := wh.GetDomainCache().GetDomainID(request.GetDomain())
	if err != nil {
		return nil, wh.error(err, scope)
	}

	if err := wh.validateTaskList(request.TaskList, scope, request.GetDomain()); err != nil {
		return nil, err
	}

	if err := wh.validateTaskListType(request.TaskListType, scope); err != nil {
		return nil, err
	}

	clientResp, err := wh.GetMatchingClient().DescribeTaskList(ctx, &types.MatchingDescribeTaskListRequest{
		DomainUUID:  common.StringPtr(domainID),
		DescRequest: thrift.ToDescribeTaskListRequest(request),
	})
	response := thrift.FromDescribeTaskListResponse(clientResp)
	err = thrift.FromError(err)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	return response, nil
}

// ListTaskListPartitions returns all the partition and host for a taskList
func (wh *WorkflowHandler) ListTaskListPartitions(
	ctx context.Context,
	request *gen.ListTaskListPartitionsRequest,
) (resp *gen.ListTaskListPartitionsResponse, retError error) {
	defer log.CapturePanic(wh.GetLogger(), &retError)

	scope, sw := wh.startRequestProfileWithDomain(metrics.FrontendListTaskListPartitionsScope, request)
	defer sw.Stop()

	if wh.isShuttingDown() {
		return nil, errShuttingDown
	}

	if request == nil {
		return nil, wh.error(errRequestNotSet, scope)
	}

	if ok := wh.allow(request); !ok {
		return nil, wh.error(createServiceBusyError(), scope)
	}

	if request.GetDomain() == "" {
		return nil, wh.error(errDomainNotSet, scope)
	}

	if err := wh.validateTaskList(request.TaskList, scope, request.GetDomain()); err != nil {
		return nil, err
	}

	clientResp, err := wh.GetMatchingClient().ListTaskListPartitions(ctx, &types.MatchingListTaskListPartitionsRequest{
		Domain:   request.Domain,
		TaskList: thrift.ToTaskList(request.TaskList),
	})
	resp = thrift.FromListTaskListPartitionsResponse(clientResp)
	err = thrift.FromError(err)
	return resp, err
}

func (wh *WorkflowHandler) getRawHistory(
	ctx context.Context,
	scope metrics.Scope,
	domainID string,
	execution gen.WorkflowExecution,
	firstEventID int64,
	nextEventID int64,
	pageSize int32,
	nextPageToken []byte,
	transientDecision *gen.TransientDecisionInfo,
	branchToken []byte,
) ([]*gen.DataBlob, []byte, error) {
	rawHistory := []*gen.DataBlob{}
	shardID := common.WorkflowIDToHistoryShard(*execution.WorkflowId, wh.config.NumHistoryShards)

	resp, err := wh.GetHistoryManager().ReadRawHistoryBranch(ctx, &persistence.ReadHistoryBranchRequest{
		BranchToken:   branchToken,
		MinEventID:    firstEventID,
		MaxEventID:    nextEventID,
		PageSize:      int(pageSize),
		NextPageToken: nextPageToken,
		ShardID:       common.IntPtr(shardID),
	})
	if err != nil {
		return nil, nil, err
	}

	var encoding *gen.EncodingType
	for _, data := range resp.HistoryEventBlobs {
		switch data.Encoding {
		case common.EncodingTypeJSON:
			encoding = gen.EncodingTypeJSON.Ptr()
		case common.EncodingTypeThriftRW:
			encoding = gen.EncodingTypeThriftRW.Ptr()
		default:
			panic(fmt.Sprintf("Invalid encoding type for raw history, encoding type: %s", data.Encoding))
		}
		rawHistory = append(rawHistory, &gen.DataBlob{
			EncodingType: encoding,
			Data:         data.Data,
		})
	}

	if len(nextPageToken) == 0 && transientDecision != nil {
		if err := wh.validateTransientDecisionEvents(nextEventID, transientDecision); err != nil {
			scope.IncCounter(metrics.CadenceErrIncompleteHistoryCounter)
			wh.GetLogger().Error("getHistory error",
				tag.WorkflowDomainID(domainID),
				tag.WorkflowID(execution.GetWorkflowId()),
				tag.WorkflowRunID(execution.GetRunId()),
				tag.Error(err))
		}
		blob, err := wh.GetPayloadSerializer().SerializeBatchEvents(
			[]*gen.HistoryEvent{transientDecision.ScheduledEvent, transientDecision.StartedEvent}, common.EncodingTypeThriftRW)
		if err != nil {
			return nil, nil, err
		}
		rawHistory = append(rawHistory, &gen.DataBlob{
			EncodingType: gen.EncodingTypeThriftRW.Ptr(),
			Data:         blob.Data,
		})
	}

	return rawHistory, resp.NextPageToken, nil
}

func (wh *WorkflowHandler) getHistory(
	ctx context.Context,
	scope metrics.Scope,
	domainID string,
	execution gen.WorkflowExecution,
	firstEventID, nextEventID int64,
	pageSize int32,
	nextPageToken []byte,
	transientDecision *gen.TransientDecisionInfo,
	branchToken []byte,
) (*gen.History, []byte, error) {

	var historyEvents []*gen.HistoryEvent
	var size int

	isFirstPage := len(nextPageToken) == 0
	shardID := common.WorkflowIDToHistoryShard(*execution.WorkflowId, wh.config.NumHistoryShards)
	var err error
	historyEvents, size, nextPageToken, err = persistence.ReadFullPageV2Events(ctx, wh.GetHistoryManager(), &persistence.ReadHistoryBranchRequest{
		BranchToken:   branchToken,
		MinEventID:    firstEventID,
		MaxEventID:    nextEventID,
		PageSize:      int(pageSize),
		NextPageToken: nextPageToken,
		ShardID:       common.IntPtr(shardID),
	})
	if err != nil {
		return nil, nil, err
	}

	scope.RecordTimer(metrics.HistorySize, time.Duration(size))

	isLastPage := len(nextPageToken) == 0
	if err := verifyHistoryIsComplete(
		historyEvents,
		firstEventID,
		nextEventID-1,
		isFirstPage,
		isLastPage,
		int(pageSize)); err != nil {
		scope.IncCounter(metrics.CadenceErrIncompleteHistoryCounter)
		wh.GetLogger().Error("getHistory: incomplete history",
			tag.WorkflowDomainID(domainID),
			tag.WorkflowID(execution.GetWorkflowId()),
			tag.WorkflowRunID(execution.GetRunId()),
			tag.Error(err))
		return nil, nil, err
	}

	if len(nextPageToken) == 0 && transientDecision != nil {
		if err := wh.validateTransientDecisionEvents(nextEventID, transientDecision); err != nil {
			scope.IncCounter(metrics.CadenceErrIncompleteHistoryCounter)
			wh.GetLogger().Error("getHistory error",
				tag.WorkflowDomainID(domainID),
				tag.WorkflowID(execution.GetWorkflowId()),
				tag.WorkflowRunID(execution.GetRunId()),
				tag.Error(err))
		}
		// Append the transient decision events once we are done enumerating everything from the events table
		historyEvents = append(historyEvents, transientDecision.ScheduledEvent, transientDecision.StartedEvent)
	}

	executionHistory := &gen.History{}
	executionHistory.Events = historyEvents
	return executionHistory, nextPageToken, nil
}

func (wh *WorkflowHandler) validateTransientDecisionEvents(
	expectedNextEventID int64,
	decision *gen.TransientDecisionInfo,
) error {

	if decision.ScheduledEvent.GetEventId() == expectedNextEventID &&
		decision.StartedEvent.GetEventId() == expectedNextEventID+1 {
		return nil
	}

	return fmt.Errorf(
		"invalid transient decision: "+
			"expectedScheduledEventID=%v expectedStartedEventID=%v but have scheduledEventID=%v startedEventID=%v",
		expectedNextEventID,
		expectedNextEventID+1,
		decision.ScheduledEvent.GetEventId(),
		decision.StartedEvent.GetEventId())
}

// startRequestProfile initiates recording of request metrics
func (wh *WorkflowHandler) startRequestProfile(scope int) (metrics.Scope, metrics.Stopwatch) {
	metricsScope := wh.GetMetricsClient().Scope(scope).Tagged(metrics.DomainUnknownTag())
	// timer should be emitted with the all tag
	sw := metricsScope.StartTimer(metrics.CadenceLatency)
	metricsScope.IncCounter(metrics.CadenceRequests)
	return metricsScope, sw
}

// startRequestProfileWithDomain initiates recording of request metrics and returns a domain tagged scope
func (wh *WorkflowHandler) startRequestProfileWithDomain(scope int, d domainGetter) (metrics.Scope, metrics.Stopwatch) {
	metricsScope := getMetricsScopeWithDomain(scope, d, wh.GetMetricsClient())
	sw := metricsScope.StartTimer(metrics.CadenceLatency)
	metricsScope.IncCounter(metrics.CadenceRequests)
	return metricsScope, sw
}

// getDefaultScope returns a default scope to use for request metrics
func (wh *WorkflowHandler) getDefaultScope(scope int) metrics.Scope {
	return wh.GetMetricsClient().Scope(scope).Tagged(metrics.DomainUnknownTag())
}

func frontendInternalServiceError(fmtStr string, args ...interface{}) error {
	// NOTE: For internal error, we can't return thrift error from cadence-frontend.
	// Because in uber internal metrics, thrift errors are counted as user errors.
	return fmt.Errorf(fmtStr, args...)
}

func (wh *WorkflowHandler) error(err error, scope metrics.Scope, tagsForErrorLog ...tag.Tag) error {
	switch err := err.(type) {
	case *gen.InternalServiceError:
		wh.GetLogger().WithTags(tagsForErrorLog...).Error("Internal service error", tag.Error(err))
		scope.IncCounter(metrics.CadenceFailures)
		return frontendInternalServiceError("cadence internal error, msg: %v", err.Message)
	case *gen.BadRequestError:
		scope.IncCounter(metrics.CadenceErrBadRequestCounter)
		return err
	case *gen.DomainNotActiveError:
		scope.IncCounter(metrics.CadenceErrBadRequestCounter)
		return err
	case *gen.ServiceBusyError:
		scope.IncCounter(metrics.CadenceErrServiceBusyCounter)
		return err
	case *gen.EntityNotExistsError:
		scope.IncCounter(metrics.CadenceErrEntityNotExistsCounter)
		return err
	case *gen.WorkflowExecutionAlreadyStartedError:
		scope.IncCounter(metrics.CadenceErrExecutionAlreadyStartedCounter)
		return err
	case *gen.DomainAlreadyExistsError:
		scope.IncCounter(metrics.CadenceErrDomainAlreadyExistsCounter)
		return err
	case *gen.CancellationAlreadyRequestedError:
		scope.IncCounter(metrics.CadenceErrCancellationAlreadyRequestedCounter)
		return err
	case *gen.QueryFailedError:
		scope.IncCounter(metrics.CadenceErrQueryFailedCounter)
		return err
	case *gen.LimitExceededError:
		scope.IncCounter(metrics.CadenceErrLimitExceededCounter)
		return err
	case *gen.ClientVersionNotSupportedError:
		scope.IncCounter(metrics.CadenceErrClientVersionNotSupportedCounter)
		return err
	case *yarpcerrors.Status:
		if err.Code() == yarpcerrors.CodeDeadlineExceeded {
			scope.IncCounter(metrics.CadenceErrContextTimeoutCounter)
			return err
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		scope.IncCounter(metrics.CadenceErrContextTimeoutCounter)
		return err
	}
	wh.GetLogger().WithTags(tagsForErrorLog...).Error("Uncategorized error",
		tag.Error(err))
	scope.IncCounter(metrics.CadenceFailures)
	return frontendInternalServiceError("cadence internal uncategorized error, msg: %v", err.Error())
}

func (wh *WorkflowHandler) validateTaskListType(t *gen.TaskListType, scope metrics.Scope) error {
	if t == nil {
		return wh.error(errTaskListTypeNotSet, scope)
	}
	return nil
}

func (wh *WorkflowHandler) validateTaskList(t *gen.TaskList, scope metrics.Scope, domain string) error {
	if t == nil || t.Name == nil || t.GetName() == "" {
		return wh.error(errTaskListNotSet, scope)
	}
	if !wh.validIDLength(t.GetName(), scope, domain) {
		return wh.error(errTaskListTooLong, scope)
	}
	if wh.isRawListList(t) && len(t.GetName()) > wh.config.MaxRawTaskListNameLimit(domain) {
		return wh.error(errRawTaskListTooLong, scope)
	}
	return nil
}

func (wh *WorkflowHandler) isRawListList(t *gen.TaskList) bool {
	return t.GetKind() != gen.TaskListKindSticky && !strings.HasPrefix(t.GetName(), common.ReservedTaskListPrefix)
}

func (wh *WorkflowHandler) validateExecutionAndEmitMetrics(w *gen.WorkflowExecution, scope metrics.Scope) error {
	err := validateExecution(w)
	if err != nil {
		return wh.error(err, scope)
	}
	return nil
}

func validateExecution(w *gen.WorkflowExecution) error {
	if w == nil {
		return errExecutionNotSet
	}
	if w.WorkflowId == nil || w.GetWorkflowId() == "" {
		return errWorkflowIDNotSet
	}
	if w.GetRunId() != "" && uuid.Parse(w.GetRunId()) == nil {
		return errInvalidRunID
	}
	return nil
}

func (wh *WorkflowHandler) createPollForDecisionTaskResponse(
	ctx context.Context,
	scope metrics.Scope,
	domainID string,
	matchingResp *m.PollForDecisionTaskResponse,
	branchToken []byte,
) (*gen.PollForDecisionTaskResponse, error) {

	if matchingResp.WorkflowExecution == nil {
		// this will happen if there is no decision task to be send to worker / caller
		return &gen.PollForDecisionTaskResponse{}, nil
	}

	var history *gen.History
	var continuation []byte
	var err error

	if matchingResp.GetStickyExecutionEnabled() && matchingResp.Query != nil {
		// meaning sticky query, we should not return any events to worker
		// since query task only check the current status
		history = &gen.History{
			Events: []*gen.HistoryEvent{},
		}
	} else {
		// here we have 3 cases:
		// 1. sticky && non query task
		// 2. non sticky &&  non query task
		// 3. non sticky && query task
		// for 1, partial history have to be send back
		// for 2 and 3, full history have to be send back

		var persistenceToken []byte

		firstEventID := common.FirstEventID
		nextEventID := matchingResp.GetNextEventId()
		if matchingResp.GetStickyExecutionEnabled() {
			firstEventID = matchingResp.GetPreviousStartedEventId() + 1
		}
		domain, dErr := wh.GetDomainCache().GetDomainByID(domainID)
		if dErr != nil {
			return nil, dErr
		}
		scope = scope.Tagged(metrics.DomainTag(domain.GetInfo().Name))
		history, persistenceToken, err = wh.getHistory(
			ctx,
			scope,
			domainID,
			*matchingResp.WorkflowExecution,
			firstEventID,
			nextEventID,
			int32(wh.config.HistoryMaxPageSize(domain.GetInfo().Name)),
			nil,
			matchingResp.DecisionInfo,
			branchToken,
		)
		if err != nil {
			return nil, err
		}

		if len(persistenceToken) != 0 {
			continuation, err = serializeHistoryToken(&getHistoryContinuationToken{
				RunID:             matchingResp.WorkflowExecution.GetRunId(),
				FirstEventID:      firstEventID,
				NextEventID:       nextEventID,
				PersistenceToken:  persistenceToken,
				TransientDecision: matchingResp.DecisionInfo,
				BranchToken:       branchToken,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	resp := &gen.PollForDecisionTaskResponse{
		TaskToken:                 matchingResp.TaskToken,
		WorkflowExecution:         matchingResp.WorkflowExecution,
		WorkflowType:              matchingResp.WorkflowType,
		PreviousStartedEventId:    matchingResp.PreviousStartedEventId,
		StartedEventId:            matchingResp.StartedEventId,
		Query:                     matchingResp.Query,
		BacklogCountHint:          matchingResp.BacklogCountHint,
		Attempt:                   matchingResp.Attempt,
		History:                   history,
		NextPageToken:             continuation,
		WorkflowExecutionTaskList: matchingResp.WorkflowExecutionTaskList,
		ScheduledTimestamp:        matchingResp.ScheduledTimestamp,
		StartedTimestamp:          matchingResp.StartedTimestamp,
		Queries:                   matchingResp.Queries,
	}

	return resp, nil
}

func verifyHistoryIsComplete(
	events []*gen.HistoryEvent,
	expectedFirstEventID int64,
	expectedLastEventID int64,
	isFirstPage bool,
	isLastPage bool,
	pageSize int,
) error {

	nEvents := len(events)
	if nEvents == 0 {
		if isLastPage {
			// we seem to be returning a non-nil pageToken on the lastPage which
			// in turn cases the client to call getHistory again - only to find
			// there are no more events to consume - bail out if this is the case here
			return nil
		}
		return fmt.Errorf("invalid history: contains zero events")
	}

	firstEventID := events[0].GetEventId()
	lastEventID := events[nEvents-1].GetEventId()

	if !isFirstPage { // atleast one page of history has been read previously
		if firstEventID <= expectedFirstEventID {
			// not first page and no events have been read in the previous pages - not possible
			return &gen.InternalServiceError{
				Message: fmt.Sprintf(
					"invalid history: expected first eventID to be > %v but got %v", expectedFirstEventID, firstEventID),
			}
		}
		expectedFirstEventID = firstEventID
	}

	if !isLastPage {
		// estimate lastEventID based on pageSize. This is a lower bound
		// since the persistence layer counts "batch of events" as a single page
		expectedLastEventID = expectedFirstEventID + int64(pageSize) - 1
	}

	nExpectedEvents := expectedLastEventID - expectedFirstEventID + 1

	if firstEventID == expectedFirstEventID &&
		((isLastPage && lastEventID == expectedLastEventID && int64(nEvents) == nExpectedEvents) ||
			(!isLastPage && lastEventID >= expectedLastEventID && int64(nEvents) >= nExpectedEvents)) {
		return nil
	}

	return &gen.InternalServiceError{
		Message: fmt.Sprintf(
			"incomplete history: "+
				"expected events [%v-%v] but got events [%v-%v] of length %v:"+
				"isFirstPage=%v,isLastPage=%v,pageSize=%v",
			expectedFirstEventID,
			expectedLastEventID,
			firstEventID,
			lastEventID,
			nEvents,
			isFirstPage,
			isLastPage,
			pageSize),
	}
}

func deserializeHistoryToken(bytes []byte) (*getHistoryContinuationToken, error) {
	token := &getHistoryContinuationToken{}
	err := json.Unmarshal(bytes, token)
	return token, err
}

func serializeHistoryToken(token *getHistoryContinuationToken) ([]byte, error) {
	if token == nil {
		return nil, nil
	}

	bytes, err := json.Marshal(token)
	return bytes, err
}

func createServiceBusyError() *gen.ServiceBusyError {
	err := &gen.ServiceBusyError{}
	err.Message = "Too many outstanding requests to the cadence service"
	return err
}

func isFailoverRequest(updateRequest *gen.UpdateDomainRequest) bool {
	return updateRequest.ReplicationConfiguration != nil && updateRequest.ReplicationConfiguration.ActiveClusterName != nil
}

func isGraceFailoverRequest(updateRequest *gen.UpdateDomainRequest) bool {
	return updateRequest.IsSetFailoverTimeoutInSeconds()
}

func (wh *WorkflowHandler) checkOngoingFailover(
	ctx context.Context,
	domainName *string,
) error {

	clusterMetadata := wh.GetClusterMetadata()
	respChan := make(chan *types.DescribeDomainResponse, len(clusterMetadata.GetAllClusterInfo()))
	wg := &sync.WaitGroup{}

	describeDomain := func(
		ctx context.Context,
		client frontend.Client,
		domainName *string,
	) {
		defer wg.Done()
		resp, _ := client.DescribeDomain(
			ctx,
			&types.DescribeDomainRequest{
				Name: domainName,
			},
		)
		respChan <- resp
	}

	for clusterName, cluster := range clusterMetadata.GetAllClusterInfo() {
		if !cluster.Enabled {
			continue
		}
		frontendClient := wh.GetRemoteFrontendClient(clusterName)
		wg.Add(1)
		go describeDomain(
			ctx,
			frontendClient,
			domainName,
		)
	}
	wg.Wait()
	close(respChan)

	var failoverVersion *int64
	for resp := range respChan {
		if resp == nil {
			return &gen.InternalServiceError{
				Message: "Failed to verify failover version from all clusters",
			}
		}
		if failoverVersion == nil {
			failoverVersion = resp.FailoverVersion
		}
		if *failoverVersion != resp.GetFailoverVersion() {
			return &gen.BadRequestError{
				Message: "Concurrent failover is not allow.",
			}
		}
	}
	return nil
}

func (wh *WorkflowHandler) historyArchived(ctx context.Context, request *gen.GetWorkflowExecutionHistoryRequest, domainID string) bool {
	if request.GetExecution() == nil || request.GetExecution().GetRunId() == "" {
		return false
	}
	getMutableStateRequest := &h.GetMutableStateRequest{
		DomainUUID: common.StringPtr(domainID),
		Execution:  request.Execution,
	}
	_, err := wh.GetHistoryClient().GetMutableState(ctx, thrift.ToGetMutableStateRequest(getMutableStateRequest))
	err = thrift.FromError(err)
	if err == nil {
		return false
	}
	switch err.(type) {
	case *gen.EntityNotExistsError:
		// the only case in which history is assumed to be archived is if getting mutable state returns entity not found error
		return true
	}
	return false
}

func (wh *WorkflowHandler) getArchivedHistory(
	ctx context.Context,
	request *gen.GetWorkflowExecutionHistoryRequest,
	domainID string,
	scope metrics.Scope,
) (*gen.GetWorkflowExecutionHistoryResponse, error) {
	wfExecution := request.GetExecution()
	entry, err := wh.GetDomainCache().GetDomainByID(domainID)
	if err != nil {
		return nil, wh.error(err, scope)
	}

	URIString := entry.GetConfig().HistoryArchivalURI
	if URIString == "" {
		// if URI is empty, it means the domain has never enabled for archival.
		// the error is not "workflow has passed retention period", because
		// we have no way to tell if the requested workflow exists or not.
		return nil, wh.error(errHistoryNotFound, scope, getWfIDRunIDTags(wfExecution)...)
	}

	URI, err := archiver.NewURI(URIString)
	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	historyArchiver, err := wh.GetArchiverProvider().GetHistoryArchiver(URI.Scheme(), common.FrontendServiceName)
	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	resp, err := historyArchiver.Get(ctx, URI, &archiver.GetHistoryRequest{
		DomainID:      domainID,
		WorkflowID:    request.GetExecution().GetWorkflowId(),
		RunID:         request.GetExecution().GetRunId(),
		NextPageToken: request.GetNextPageToken(),
		PageSize:      int(request.GetMaximumPageSize()),
	})
	if err != nil {
		return nil, wh.error(err, scope, getWfIDRunIDTags(wfExecution)...)
	}

	history := &gen.History{}
	for _, batch := range resp.HistoryBatches {
		history.Events = append(history.Events, batch.Events...)
	}
	return &gen.GetWorkflowExecutionHistoryResponse{
		History:       history,
		NextPageToken: resp.NextPageToken,
		Archived:      common.BoolPtr(true),
	}, nil
}

func (wh *WorkflowHandler) validIDLength(id string, scope metrics.Scope, domain string) bool {
	valid := len(id) <= wh.config.MaxIDLengthLimit(domain)
	if len(id) > wh.config.MaxIDLengthWarnLimit() {
		scope.IncCounter(metrics.CadenceErrIDLengthExceededWarnLimit)
	}
	return valid
}

func (wh *WorkflowHandler) convertIndexedKeyToThrift(keys map[string]interface{}) map[string]gen.IndexedValueType {
	converted := make(map[string]gen.IndexedValueType)
	for k, v := range keys {
		converted[k] = common.ConvertIndexedValueTypeToThriftType(v, wh.GetLogger())
	}
	return converted
}

func (wh *WorkflowHandler) isListRequestPageSizeTooLarge(pageSize int32, domain string) bool {
	return wh.config.EnableReadVisibilityFromES(domain) &&
		pageSize > int32(wh.config.ESIndexMaxResultWindow())
}

func (wh *WorkflowHandler) allow(d domainGetter) bool {
	domain := ""
	if d != nil {
		domain = d.GetDomain()
	}
	return wh.rateLimiter.Allow(quotas.Info{Domain: domain})
}

// GetClusterInfo return information about cadence deployment
func (wh *WorkflowHandler) GetClusterInfo(
	ctx context.Context,
) (resp *gen.ClusterInfo, err error) {
	defer log.CapturePanic(wh.GetLogger(), &err)

	scope := wh.getDefaultScope(metrics.FrontendClientGetClusterInfoScope)
	if ok := wh.allow(nil); !ok {
		return nil, wh.error(createServiceBusyError(), scope)
	}

	return &gen.ClusterInfo{
		SupportedClientVersions: &gen.SupportedClientVersions{
			GoSdk:   common.StringPtr(client.SupportedGoSDKVersion),
			JavaSdk: common.StringPtr(client.SupportedJavaSDKVersion),
		},
	}, nil
}

func checkPermission(
	config *Config,
	securityToken *string,
) error {
	if config.EnableAdminProtection() {
		if securityToken == nil {
			return errNoPermission
		}
		requiredToken := config.AdminOperationToken()
		if *securityToken != requiredToken {
			return errNoPermission
		}
	}
	return nil
}

type domainWrapper struct {
	domain string
}

func (d domainWrapper) GetDomain() string {
	return d.domain
}

func (hs HealthStatus) String() string {
	switch hs {
	case HealthStatusOK:
		return "OK"
	case HealthStatusWarmingUp:
		return "WarmingUp"
	case HealthStatusShuttingDown:
		return "ShuttingDown"
	default:
		return "unknown"
	}
}

func getWfIDRunIDTags(wf *gen.WorkflowExecution) []tag.Tag {
	if wf == nil {
		return nil
	}
	return []tag.Tag{
		tag.WorkflowID(wf.GetWorkflowId()),
		tag.WorkflowRunID(wf.GetRunId()),
	}
}
