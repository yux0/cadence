// Copyright (c) 2019 Uber Technologies, Inc.
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

	health "github.com/uber/cadence/.gen/go/health"
	"github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common/authorization"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/resource"
	"github.com/uber/cadence/common/types"
)

var errUnauthorized = &types.BadRequestError{Message: "Request unauthorized."}

// AccessControlledWorkflowHandler frontend handler wrapper for authentication and authorization
type AccessControlledWorkflowHandler struct {
	resource.Resource

	frontendHandler Handler
	authorizer      authorization.Authorizer
}

var _ Handler = (*AccessControlledWorkflowHandler)(nil)

// NewAccessControlledHandlerImpl creates frontend handler with authentication support
func NewAccessControlledHandlerImpl(wfHandler Handler, resource resource.Resource, authorizer authorization.Authorizer) *AccessControlledWorkflowHandler {
	if authorizer == nil {
		authorizer = authorization.NewNopAuthorizer()
	}

	return &AccessControlledWorkflowHandler{
		Resource:        resource,
		frontendHandler: wfHandler,
		authorizer:      authorizer,
	}
}

// Health callback for for health check
func (a *AccessControlledWorkflowHandler) Health(ctx context.Context) (*health.HealthStatus, error) {
	return a.frontendHandler.Health(ctx)
}

// CountWorkflowExecutions API call
func (a *AccessControlledWorkflowHandler) CountWorkflowExecutions(
	ctx context.Context,
	request *types.CountWorkflowExecutionsRequest,
) (*types.CountWorkflowExecutionsResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendCountWorkflowExecutionsScope, request)

	attr := &authorization.Attributes{
		APIName:    "CountWorkflowExecutions",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.CountWorkflowExecutions(ctx, request)
}

// DeprecateDomain API call
func (a *AccessControlledWorkflowHandler) DeprecateDomain(
	ctx context.Context,
	request *types.DeprecateDomainRequest,
) error {

	scope := a.getMetricsScopeWithDomainName(metrics.FrontendDeprecateDomainScope, request.GetName())

	attr := &authorization.Attributes{
		APIName:    "DeprecateDomain",
		DomainName: request.GetName(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return err
	}
	if !isAuthorized {
		return errUnauthorized
	}

	return a.frontendHandler.DeprecateDomain(ctx, request)
}

// DescribeDomain API call
func (a *AccessControlledWorkflowHandler) DescribeDomain(
	ctx context.Context,
	request *types.DescribeDomainRequest,
) (*types.DescribeDomainResponse, error) {

	scope := a.getMetricsScopeWithDomainName(metrics.FrontendDescribeDomainScope, request.GetName())

	attr := &authorization.Attributes{
		APIName:    "DescribeDomain",
		DomainName: request.GetName(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.DescribeDomain(ctx, request)
}

// DescribeTaskList API call
func (a *AccessControlledWorkflowHandler) DescribeTaskList(
	ctx context.Context,
	request *shared.DescribeTaskListRequest,
) (*shared.DescribeTaskListResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendDescribeTaskListScope, request)

	attr := &authorization.Attributes{
		APIName:    "DescribeTaskList",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.DescribeTaskList(ctx, request)
}

// DescribeWorkflowExecution API call
func (a *AccessControlledWorkflowHandler) DescribeWorkflowExecution(
	ctx context.Context,
	request *shared.DescribeWorkflowExecutionRequest,
) (*shared.DescribeWorkflowExecutionResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendDescribeWorkflowExecutionScope, request)

	attr := &authorization.Attributes{
		APIName:    "DescribeWorkflowExecution",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.DescribeWorkflowExecution(ctx, request)
}

// GetSearchAttributes API call
func (a *AccessControlledWorkflowHandler) GetSearchAttributes(
	ctx context.Context,
) (*shared.GetSearchAttributesResponse, error) {
	return a.frontendHandler.GetSearchAttributes(ctx)
}

// GetWorkflowExecutionHistory API call
func (a *AccessControlledWorkflowHandler) GetWorkflowExecutionHistory(
	ctx context.Context,
	request *shared.GetWorkflowExecutionHistoryRequest,
) (*shared.GetWorkflowExecutionHistoryResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendGetWorkflowExecutionHistoryScope, request)

	attr := &authorization.Attributes{
		APIName:    "GetWorkflowExecutionHistory",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.GetWorkflowExecutionHistory(ctx, request)
}

// ListArchivedWorkflowExecutions API call
func (a *AccessControlledWorkflowHandler) ListArchivedWorkflowExecutions(
	ctx context.Context,
	request *shared.ListArchivedWorkflowExecutionsRequest,
) (*shared.ListArchivedWorkflowExecutionsResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendListArchivedWorkflowExecutionsScope, request)

	attr := &authorization.Attributes{
		APIName:    "ListArchivedWorkflowExecutions",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.ListArchivedWorkflowExecutions(ctx, request)
}

// ListClosedWorkflowExecutions API call
func (a *AccessControlledWorkflowHandler) ListClosedWorkflowExecutions(
	ctx context.Context,
	request *types.ListClosedWorkflowExecutionsRequest,
) (*types.ListClosedWorkflowExecutionsResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendListClosedWorkflowExecutionsScope, request)

	attr := &authorization.Attributes{
		APIName:    "ListClosedWorkflowExecutions",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.ListClosedWorkflowExecutions(ctx, request)
}

// ListDomains API call
func (a *AccessControlledWorkflowHandler) ListDomains(
	ctx context.Context,
	request *types.ListDomainsRequest,
) (*types.ListDomainsResponse, error) {

	scope := a.GetMetricsClient().Scope(metrics.FrontendListDomainsScope)

	attr := &authorization.Attributes{
		APIName: "ListDomains",
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.ListDomains(ctx, request)
}

// ListOpenWorkflowExecutions API call
func (a *AccessControlledWorkflowHandler) ListOpenWorkflowExecutions(
	ctx context.Context,
	request *types.ListOpenWorkflowExecutionsRequest,
) (*types.ListOpenWorkflowExecutionsResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendListOpenWorkflowExecutionsScope, request)

	attr := &authorization.Attributes{
		APIName:    "ListOpenWorkflowExecutions",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.ListOpenWorkflowExecutions(ctx, request)
}

// ListWorkflowExecutions API call
func (a *AccessControlledWorkflowHandler) ListWorkflowExecutions(
	ctx context.Context,
	request *types.ListWorkflowExecutionsRequest,
) (*types.ListWorkflowExecutionsResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendListWorkflowExecutionsScope, request)

	attr := &authorization.Attributes{
		APIName:    "ListWorkflowExecutions",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.ListWorkflowExecutions(ctx, request)
}

// PollForActivityTask API call
func (a *AccessControlledWorkflowHandler) PollForActivityTask(
	ctx context.Context,
	request *shared.PollForActivityTaskRequest,
) (*shared.PollForActivityTaskResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendPollForActivityTaskScope, request)

	attr := &authorization.Attributes{
		APIName:    "PollForActivityTask",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.PollForActivityTask(ctx, request)
}

// PollForDecisionTask API call
func (a *AccessControlledWorkflowHandler) PollForDecisionTask(
	ctx context.Context,
	request *shared.PollForDecisionTaskRequest,
) (*shared.PollForDecisionTaskResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendPollForDecisionTaskScope, request)

	attr := &authorization.Attributes{
		APIName:    "PollForDecisionTask",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.PollForDecisionTask(ctx, request)
}

// QueryWorkflow API call
func (a *AccessControlledWorkflowHandler) QueryWorkflow(
	ctx context.Context,
	request *shared.QueryWorkflowRequest,
) (*shared.QueryWorkflowResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendQueryWorkflowScope, request)

	attr := &authorization.Attributes{
		APIName:    "QueryWorkflow",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.QueryWorkflow(ctx, request)
}

// GetClusterInfo API call
func (a *AccessControlledWorkflowHandler) GetClusterInfo(
	ctx context.Context,
) (*shared.ClusterInfo, error) {
	return a.frontendHandler.GetClusterInfo(ctx)
}

// RecordActivityTaskHeartbeat API call
func (a *AccessControlledWorkflowHandler) RecordActivityTaskHeartbeat(
	ctx context.Context,
	request *shared.RecordActivityTaskHeartbeatRequest,
) (*shared.RecordActivityTaskHeartbeatResponse, error) {
	// TODO(vancexu): add auth check for service API
	return a.frontendHandler.RecordActivityTaskHeartbeat(ctx, request)
}

// RecordActivityTaskHeartbeatByID API call
func (a *AccessControlledWorkflowHandler) RecordActivityTaskHeartbeatByID(
	ctx context.Context,
	request *shared.RecordActivityTaskHeartbeatByIDRequest,
) (*shared.RecordActivityTaskHeartbeatResponse, error) {
	return a.frontendHandler.RecordActivityTaskHeartbeatByID(ctx, request)
}

// RegisterDomain API call
func (a *AccessControlledWorkflowHandler) RegisterDomain(
	ctx context.Context,
	request *types.RegisterDomainRequest,
) error {

	scope := a.getMetricsScopeWithDomainName(metrics.FrontendRegisterDomainScope, request.GetName())

	attr := &authorization.Attributes{
		APIName:    "RegisterDomain",
		DomainName: request.GetName(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return err
	}
	if !isAuthorized {
		return errUnauthorized
	}

	return a.frontendHandler.RegisterDomain(ctx, request)
}

// RequestCancelWorkflowExecution API call
func (a *AccessControlledWorkflowHandler) RequestCancelWorkflowExecution(
	ctx context.Context,
	request *shared.RequestCancelWorkflowExecutionRequest,
) error {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendRequestCancelWorkflowExecutionScope, request)

	attr := &authorization.Attributes{
		APIName:    "RequestCancelWorkflowExecution",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return err
	}
	if !isAuthorized {
		return errUnauthorized
	}

	return a.frontendHandler.RequestCancelWorkflowExecution(ctx, request)
}

// ResetStickyTaskList API call
func (a *AccessControlledWorkflowHandler) ResetStickyTaskList(
	ctx context.Context,
	request *shared.ResetStickyTaskListRequest,
) (*shared.ResetStickyTaskListResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendResetStickyTaskListScope, request)

	attr := &authorization.Attributes{
		APIName:    "ResetStickyTaskList",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.ResetStickyTaskList(ctx, request)
}

// ResetWorkflowExecution API call
func (a *AccessControlledWorkflowHandler) ResetWorkflowExecution(
	ctx context.Context,
	request *shared.ResetWorkflowExecutionRequest,
) (*shared.ResetWorkflowExecutionResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendResetWorkflowExecutionScope, request)

	attr := &authorization.Attributes{
		APIName:    "ResetWorkflowExecution",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.ResetWorkflowExecution(ctx, request)
}

// RespondActivityTaskCanceled API call
func (a *AccessControlledWorkflowHandler) RespondActivityTaskCanceled(
	ctx context.Context,
	request *shared.RespondActivityTaskCanceledRequest,
) error {
	return a.frontendHandler.RespondActivityTaskCanceled(ctx, request)
}

// RespondActivityTaskCanceledByID API call
func (a *AccessControlledWorkflowHandler) RespondActivityTaskCanceledByID(
	ctx context.Context,
	request *shared.RespondActivityTaskCanceledByIDRequest,
) error {
	return a.frontendHandler.RespondActivityTaskCanceledByID(ctx, request)
}

// RespondActivityTaskCompleted API call
func (a *AccessControlledWorkflowHandler) RespondActivityTaskCompleted(
	ctx context.Context,
	request *shared.RespondActivityTaskCompletedRequest,
) error {
	return a.frontendHandler.RespondActivityTaskCompleted(ctx, request)
}

// RespondActivityTaskCompletedByID API call
func (a *AccessControlledWorkflowHandler) RespondActivityTaskCompletedByID(
	ctx context.Context,
	request *shared.RespondActivityTaskCompletedByIDRequest,
) error {
	return a.frontendHandler.RespondActivityTaskCompletedByID(ctx, request)
}

// RespondActivityTaskFailed API call
func (a *AccessControlledWorkflowHandler) RespondActivityTaskFailed(
	ctx context.Context,
	request *shared.RespondActivityTaskFailedRequest,
) error {
	return a.frontendHandler.RespondActivityTaskFailed(ctx, request)
}

// RespondActivityTaskFailedByID API call
func (a *AccessControlledWorkflowHandler) RespondActivityTaskFailedByID(
	ctx context.Context,
	request *shared.RespondActivityTaskFailedByIDRequest,
) error {
	return a.frontendHandler.RespondActivityTaskFailedByID(ctx, request)
}

// RespondDecisionTaskCompleted API call
func (a *AccessControlledWorkflowHandler) RespondDecisionTaskCompleted(
	ctx context.Context,
	request *shared.RespondDecisionTaskCompletedRequest,
) (*shared.RespondDecisionTaskCompletedResponse, error) {
	return a.frontendHandler.RespondDecisionTaskCompleted(ctx, request)
}

// RespondDecisionTaskFailed API call
func (a *AccessControlledWorkflowHandler) RespondDecisionTaskFailed(
	ctx context.Context,
	request *shared.RespondDecisionTaskFailedRequest,
) error {
	return a.frontendHandler.RespondDecisionTaskFailed(ctx, request)
}

// RespondQueryTaskCompleted API call
func (a *AccessControlledWorkflowHandler) RespondQueryTaskCompleted(
	ctx context.Context,
	request *shared.RespondQueryTaskCompletedRequest,
) error {
	return a.frontendHandler.RespondQueryTaskCompleted(ctx, request)
}

// ScanWorkflowExecutions API call
func (a *AccessControlledWorkflowHandler) ScanWorkflowExecutions(
	ctx context.Context,
	request *types.ListWorkflowExecutionsRequest,
) (*types.ListWorkflowExecutionsResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendScanWorkflowExecutionsScope, request)

	attr := &authorization.Attributes{
		APIName:    "ScanWorkflowExecutions",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.ScanWorkflowExecutions(ctx, request)
}

// SignalWithStartWorkflowExecution API call
func (a *AccessControlledWorkflowHandler) SignalWithStartWorkflowExecution(
	ctx context.Context,
	request *shared.SignalWithStartWorkflowExecutionRequest,
) (*shared.StartWorkflowExecutionResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendSignalWithStartWorkflowExecutionScope, request)

	attr := &authorization.Attributes{
		APIName:    "SignalWithStartWorkflowExecution",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.SignalWithStartWorkflowExecution(ctx, request)
}

// SignalWorkflowExecution API call
func (a *AccessControlledWorkflowHandler) SignalWorkflowExecution(
	ctx context.Context,
	request *shared.SignalWorkflowExecutionRequest,
) error {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendSignalWorkflowExecutionScope, request)

	attr := &authorization.Attributes{
		APIName:    "SignalWorkflowExecution",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return err
	}
	if !isAuthorized {
		return errUnauthorized
	}

	return a.frontendHandler.SignalWorkflowExecution(ctx, request)
}

// StartWorkflowExecution API call
func (a *AccessControlledWorkflowHandler) StartWorkflowExecution(
	ctx context.Context,
	request *shared.StartWorkflowExecutionRequest,
) (*shared.StartWorkflowExecutionResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendStartWorkflowExecutionScope, request)

	attr := &authorization.Attributes{
		APIName:    "StartWorkflowExecution",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.StartWorkflowExecution(ctx, request)
}

// TerminateWorkflowExecution API call
func (a *AccessControlledWorkflowHandler) TerminateWorkflowExecution(
	ctx context.Context,
	request *shared.TerminateWorkflowExecutionRequest,
) error {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendTerminateWorkflowExecutionScope, request)

	attr := &authorization.Attributes{
		APIName:    "TerminateWorkflowExecution",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return err
	}
	if !isAuthorized {
		return errUnauthorized
	}

	return a.frontendHandler.TerminateWorkflowExecution(ctx, request)
}

// ListTaskListPartitions API call
func (a *AccessControlledWorkflowHandler) ListTaskListPartitions(
	ctx context.Context,
	request *shared.ListTaskListPartitionsRequest,
) (*shared.ListTaskListPartitionsResponse, error) {

	scope := a.getMetricsScopeWithDomain(metrics.FrontendListTaskListPartitionsScope, request)

	attr := &authorization.Attributes{
		APIName:    "ListTaskListPartitions",
		DomainName: request.GetDomain(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.ListTaskListPartitions(ctx, request)
}

// UpdateDomain API call
func (a *AccessControlledWorkflowHandler) UpdateDomain(
	ctx context.Context,
	request *types.UpdateDomainRequest,
) (*types.UpdateDomainResponse, error) {

	scope := a.getMetricsScopeWithDomainName(metrics.FrontendUpdateDomainScope, request.GetName())

	attr := &authorization.Attributes{
		APIName:    "UpdateDomain",
		DomainName: request.GetName(),
	}
	isAuthorized, err := a.isAuthorized(ctx, attr, scope)
	if err != nil {
		return nil, err
	}
	if !isAuthorized {
		return nil, errUnauthorized
	}

	return a.frontendHandler.UpdateDomain(ctx, request)
}

func (a *AccessControlledWorkflowHandler) isAuthorized(
	ctx context.Context,
	attr *authorization.Attributes,
	scope metrics.Scope,
) (bool, error) {
	sw := scope.StartTimer(metrics.CadenceAuthorizationLatency)
	defer sw.Stop()

	result, err := a.authorizer.Authorize(ctx, attr)
	if err != nil {
		scope.IncCounter(metrics.CadenceErrAuthorizeFailedCounter)
		return false, err
	}
	isAuth := result.Decision == authorization.DecisionAllow
	if !isAuth {
		scope.IncCounter(metrics.CadenceErrUnauthorizedCounter)
	}
	return isAuth, nil
}

// getMetricsScopeWithDomain return metrics scope with domain tag
func (a *AccessControlledWorkflowHandler) getMetricsScopeWithDomain(
	scope int,
	d domainGetter,
) metrics.Scope {
	return getMetricsScopeWithDomain(scope, d, a.GetMetricsClient())
}

func getMetricsScopeWithDomain(
	scope int,
	d domainGetter,
	metricsClient metrics.Client,
) metrics.Scope {
	var metricsScope metrics.Scope
	if d != nil {
		metricsScope = metricsClient.Scope(scope).Tagged(metrics.DomainTag(d.GetDomain()))
	} else {
		metricsScope = metricsClient.Scope(scope).Tagged(metrics.DomainUnknownTag())
	}
	return metricsScope
}

// getMetricsScopeWithDomainName is for XXXDomain APIs, whose request is not domainGetter
func (a *AccessControlledWorkflowHandler) getMetricsScopeWithDomainName(
	scope int,
	domainName string,
) metrics.Scope {
	return a.GetMetricsClient().Scope(scope).Tagged(metrics.DomainTag(domainName))
}
