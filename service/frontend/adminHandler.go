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

package frontend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/pborman/uuid"

	"github.com/uber/cadence/.gen/go/admin"
	"github.com/uber/cadence/.gen/go/admin/adminserviceserver"
	h "github.com/uber/cadence/.gen/go/history"
	"github.com/uber/cadence/.gen/go/replicator"
	gen "github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/backoff"
	"github.com/uber/cadence/common/client"
	"github.com/uber/cadence/common/definition"
	"github.com/uber/cadence/common/domain"
	"github.com/uber/cadence/common/elasticsearch"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/ndc"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/resource"
	"github.com/uber/cadence/common/service"
	"github.com/uber/cadence/common/service/dynamicconfig"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/common/types/mapper/thrift"
)

var _ adminserviceserver.Interface = (*adminHandlerImpl)(nil)

const (
	endMessageID int64 = 1<<63 - 1
)

var (
	errMaxMessageIDNotSet = &gen.BadRequestError{Message: "Max messageID is not set."}
)

//go:generate mockgen -copyright_file=../../LICENSE -package $GOPACKAGE -source $GOFILE -destination handler_mock.go -package frontend github.com/uber/cadence/service/frontend AdminHandler

type (
	// AdminHandler interface for admin service
	AdminHandler interface {
		AddSearchAttribute(context.Context, *admin.AddSearchAttributeRequest) error
		CloseShard(context.Context, *gen.CloseShardRequest) error
		DescribeCluster(context.Context) (*admin.DescribeClusterResponse, error)
		DescribeHistoryHost(context.Context, *gen.DescribeHistoryHostRequest) (*gen.DescribeHistoryHostResponse, error)
		DescribeQueue(context.Context, *gen.DescribeQueueRequest) (*gen.DescribeQueueResponse, error)
		DescribeWorkflowExecution(context.Context, *admin.DescribeWorkflowExecutionRequest) (*admin.DescribeWorkflowExecutionResponse, error)
		GetDLQReplicationMessages(context.Context, *replicator.GetDLQReplicationMessagesRequest) (*replicator.GetDLQReplicationMessagesResponse, error)
		GetDomainReplicationMessages(context.Context, *replicator.GetDomainReplicationMessagesRequest) (*replicator.GetDomainReplicationMessagesResponse, error)
		GetReplicationMessages(context.Context, *replicator.GetReplicationMessagesRequest) (*replicator.GetReplicationMessagesResponse, error)
		GetWorkflowExecutionRawHistoryV2(context.Context, *admin.GetWorkflowExecutionRawHistoryV2Request) (*admin.GetWorkflowExecutionRawHistoryV2Response, error)
		MergeDLQMessages(context.Context, *replicator.MergeDLQMessagesRequest) (*replicator.MergeDLQMessagesResponse, error)
		PurgeDLQMessages(context.Context, *replicator.PurgeDLQMessagesRequest) error
		ReadDLQMessages(context.Context, *replicator.ReadDLQMessagesRequest) (*replicator.ReadDLQMessagesResponse, error)
		ReapplyEvents(context.Context, *gen.ReapplyEventsRequest) error
		RefreshWorkflowTasks(context.Context, *gen.RefreshWorkflowTasksRequest) error
		RemoveTask(context.Context, *gen.RemoveTaskRequest) error
		ResendReplicationTasks(context.Context, *admin.ResendReplicationTasksRequest) error
		ResetQueue(context.Context, *gen.ResetQueueRequest) error
	}

	// adminHandlerImpl is an implementation for admin service independent of wire protocol
	adminHandlerImpl struct {
		resource.Resource

		numberOfHistoryShards int
		params                *service.BootstrapParams
		config                *Config
		domainDLQHandler      domain.DLQMessageHandler
		domainFailoverWatcher domain.FailoverWatcher
		eventSerializder      persistence.PayloadSerializer
		esClient              elasticsearch.GenericClient
	}

	getWorkflowRawHistoryV2Token struct {
		DomainName        string
		WorkflowID        string
		RunID             string
		StartEventID      int64
		StartEventVersion int64
		EndEventID        int64
		EndEventVersion   int64
		PersistenceToken  []byte
		VersionHistories  *gen.VersionHistories
	}
)

var (
	adminServiceRetryPolicy = common.CreateAdminServiceRetryPolicy()
	resendStartEventID      = common.Int64Ptr(0)
)

// NewAdminHandler creates a thrift handler for the cadence admin service
func NewAdminHandler(
	resource resource.Resource,
	params *service.BootstrapParams,
	config *Config,
) *adminHandlerImpl {

	domainReplicationTaskExecutor := domain.NewReplicationTaskExecutor(
		resource.GetMetadataManager(),
		resource.GetTimeSource(),
		resource.GetLogger(),
	)
	return &adminHandlerImpl{
		Resource:              resource,
		numberOfHistoryShards: params.PersistenceConfig.NumHistoryShards,
		params:                params,
		config:                config,
		domainDLQHandler: domain.NewDLQMessageHandler(
			domainReplicationTaskExecutor,
			resource.GetDomainReplicationQueue(),
			resource.GetLogger(),
		),
		domainFailoverWatcher: domain.NewFailoverWatcher(
			resource.GetDomainCache(),
			resource.GetMetadataManager(),
			resource.GetTimeSource(),
			config.DomainFailoverRefreshInterval,
			config.DomainFailoverRefreshTimerJitterCoefficient,
			resource.GetMetricsClient(),
			resource.GetLogger(),
		),
		eventSerializder: persistence.NewPayloadSerializer(),
		esClient:         params.ESClient,
	}
}

// Start starts the handler
func (adh *adminHandlerImpl) Start() {

	if adh.config.EnableGracefulFailover() {
		adh.domainFailoverWatcher.Start()
	}
}

// Stop stops the handler
func (adh *adminHandlerImpl) Stop() {
	adh.domainFailoverWatcher.Stop()
}

// AddSearchAttribute add search attribute to whitelist
func (adh *adminHandlerImpl) AddSearchAttribute(
	ctx context.Context,
	request *admin.AddSearchAttributeRequest,
) (retError error) {

	defer log.CapturePanic(adh.GetLogger(), &retError)
	scope, sw := adh.startRequestProfile(metrics.AdminAddSearchAttributeScope)
	defer sw.Stop()

	// validate request
	if request == nil {
		return adh.error(errRequestNotSet, scope)
	}
	if err := checkPermission(adh.config, request.SecurityToken); err != nil {
		return adh.error(errNoPermission, scope)
	}
	if len(request.GetSearchAttribute()) == 0 {
		return adh.error(&gen.BadRequestError{Message: "SearchAttributes are not provided"}, scope)
	}
	if err := adh.validateConfigForAdvanceVisibility(); err != nil {
		return adh.error(&gen.BadRequestError{Message: fmt.Sprintf("AdvancedVisibilityStore is not configured for this Cadence Cluster")}, scope)
	}

	searchAttr := request.GetSearchAttribute()
	currentValidAttr, _ := adh.params.DynamicConfig.GetMapValue(
		dynamicconfig.ValidSearchAttributes, nil, definition.GetDefaultIndexedKeys())
	for k, v := range searchAttr {
		if definition.IsSystemIndexedKey(k) {
			return adh.error(&gen.BadRequestError{Message: fmt.Sprintf("Key [%s] is reserved by system", k)}, scope)
		}
		if _, exist := currentValidAttr[k]; exist {
			return adh.error(&gen.BadRequestError{Message: fmt.Sprintf("Key [%s] is already whitelist", k)}, scope)
		}

		currentValidAttr[k] = int(v)
	}

	// update dynamic config
	err := adh.params.DynamicConfig.UpdateValue(dynamicconfig.ValidSearchAttributes, currentValidAttr)
	if err != nil {
		return adh.error(&gen.InternalServiceError{Message: fmt.Sprintf("Failed to update dynamic config, err: %v", err)}, scope)
	}

	// update elasticsearch mapping, new added field will not be able to remove or update
	index := adh.params.ESConfig.GetVisibilityIndex()
	for k, v := range searchAttr {
		valueType := convertIndexedValueTypeToESDataType(v)
		if len(valueType) == 0 {
			return adh.error(&gen.BadRequestError{Message: fmt.Sprintf("Unknown value type, %v", v)}, scope)
		}
		err := adh.params.ESClient.PutMapping(ctx, index, definition.Attr, k, valueType)
		if adh.esClient.IsNotFoundError(err) {
			err = adh.params.ESClient.CreateIndex(ctx, index)
			if err != nil {
				return adh.error(&gen.InternalServiceError{Message: fmt.Sprintf("Failed to create ES index, err: %v", err)}, scope)
			}
			err = adh.params.ESClient.PutMapping(ctx, index, definition.Attr, k, valueType)
		}
		if err != nil {
			return adh.error(&gen.InternalServiceError{Message: fmt.Sprintf("Failed to update ES mapping, err: %v", err)}, scope)
		}
	}

	return nil
}

// DescribeWorkflowExecution returns information about the specified workflow execution.
func (adh *adminHandlerImpl) DescribeWorkflowExecution(
	ctx context.Context,
	request *admin.DescribeWorkflowExecutionRequest,
) (resp *admin.DescribeWorkflowExecutionResponse, retError error) {

	defer log.CapturePanic(adh.GetLogger(), &retError)
	scope, sw := adh.startRequestProfile(metrics.AdminDescribeWorkflowExecutionScope)
	defer sw.Stop()

	if request == nil {
		return nil, adh.error(errRequestNotSet, scope)
	}

	if err := validateExecution(request.Execution); err != nil {
		return nil, adh.error(err, scope)
	}

	shardID := common.WorkflowIDToHistoryShard(*request.Execution.WorkflowId, adh.numberOfHistoryShards)
	shardIDstr := string(shardID)
	shardIDForOutput := strconv.Itoa(shardID)

	historyHost, err := adh.GetMembershipMonitor().Lookup(common.HistoryServiceName, shardIDstr)
	if err != nil {
		return nil, adh.error(err, scope)
	}

	domainID, err := adh.GetDomainCache().GetDomainID(request.GetDomain())

	historyAddr := historyHost.GetAddress()
	clientResp, err := adh.GetHistoryClient().DescribeMutableState(ctx, &types.DescribeMutableStateRequest{
		DomainUUID: &domainID,
		Execution:  thrift.ToWorkflowExecution(request.Execution),
	})
	resp2 := thrift.FromDescribeMutableStateResponse(clientResp)
	err = thrift.FromError(err)
	if err != nil {
		return &admin.DescribeWorkflowExecutionResponse{}, err
	}
	return &admin.DescribeWorkflowExecutionResponse{
		ShardId:                common.StringPtr(shardIDForOutput),
		HistoryAddr:            common.StringPtr(historyAddr),
		MutableStateInDatabase: resp2.MutableStateInDatabase,
		MutableStateInCache:    resp2.MutableStateInCache,
	}, err
}

// RemoveTask returns information about the internal states of a history host
func (adh *adminHandlerImpl) RemoveTask(
	ctx context.Context,
	request *gen.RemoveTaskRequest,
) (retError error) {

	defer log.CapturePanic(adh.GetLogger(), &retError)
	scope, sw := adh.startRequestProfile(metrics.AdminRemoveTaskScope)
	defer sw.Stop()

	if request == nil || request.ShardID == nil || request.Type == nil || request.TaskID == nil {
		return adh.error(errRequestNotSet, scope)
	}
	err := adh.GetHistoryClient().RemoveTask(ctx, thrift.ToRemoveTaskRequest(request))
	err = thrift.FromError(err)
	return err
}

// CloseShard returns information about the internal states of a history host
func (adh *adminHandlerImpl) CloseShard(
	ctx context.Context,
	request *gen.CloseShardRequest,
) (retError error) {

	defer log.CapturePanic(adh.GetLogger(), &retError)
	scope, sw := adh.startRequestProfile(metrics.AdminCloseShardScope)
	defer sw.Stop()

	if request == nil || request.ShardID == nil {
		return adh.error(errRequestNotSet, scope)
	}
	err := adh.GetHistoryClient().CloseShard(ctx, thrift.ToCloseShardRequest(request))
	err = thrift.FromError(err)
	return err
}

// ResetQueue resets processing queue states
func (adh *adminHandlerImpl) ResetQueue(
	ctx context.Context,
	request *gen.ResetQueueRequest,
) (retError error) {

	defer log.CapturePanic(adh.GetLogger(), &retError)
	scope, sw := adh.startRequestProfile(metrics.AdminResetQueueScope)
	defer sw.Stop()

	if request == nil || request.ShardID == nil || request.ClusterName == nil || request.Type == nil {
		return adh.error(errRequestNotSet, scope)
	}
	if request.GetClusterName() == "" {
		return adh.error(errClusterNameNotSet, scope)
	}

	err := adh.GetHistoryClient().ResetQueue(ctx, thrift.ToResetQueueRequest(request))
	err = thrift.FromError(err)
	return err
}

// DescribeQueue describes processing queue states
func (adh *adminHandlerImpl) DescribeQueue(
	ctx context.Context,
	request *gen.DescribeQueueRequest,
) (resp *gen.DescribeQueueResponse, retError error) {

	defer log.CapturePanic(adh.GetLogger(), &retError)
	scope, sw := adh.startRequestProfile(metrics.AdminDescribeQueueScope)
	defer sw.Stop()

	if request == nil || request.ShardID == nil || request.ClusterName == nil || request.Type == nil {
		return nil, adh.error(errRequestNotSet, scope)
	}
	if request.GetClusterName() == "" {
		return nil, adh.error(errClusterNameNotSet, scope)
	}

	clientResp, err := adh.GetHistoryClient().DescribeQueue(ctx, thrift.ToDescribeQueueRequest(request))
	return thrift.FromDescribeQueueResponse(clientResp), thrift.FromError(err)
}

// DescribeHistoryHost returns information about the internal states of a history host
func (adh *adminHandlerImpl) DescribeHistoryHost(
	ctx context.Context,
	request *gen.DescribeHistoryHostRequest,
) (resp *gen.DescribeHistoryHostResponse, retError error) {

	defer log.CapturePanic(adh.GetLogger(), &retError)
	scope, sw := adh.startRequestProfile(metrics.AdminDescribeHistoryHostScope)
	defer sw.Stop()

	if request == nil || (request.ShardIdForHost == nil && request.ExecutionForHost == nil && request.HostAddress == nil) {
		return nil, adh.error(errRequestNotSet, scope)
	}

	if request.ExecutionForHost != nil {
		if err := validateExecution(request.ExecutionForHost); err != nil {
			return nil, adh.error(err, scope)
		}
	}

	clientResp, err := adh.GetHistoryClient().DescribeHistoryHost(ctx, thrift.ToDescribeHistoryHostRequest(request))
	return thrift.FromDescribeHistoryHostResponse(clientResp), thrift.FromError(err)
}

// GetWorkflowExecutionRawHistoryV2 - retrieves the history of workflow execution
func (adh *adminHandlerImpl) GetWorkflowExecutionRawHistoryV2(
	ctx context.Context,
	request *admin.GetWorkflowExecutionRawHistoryV2Request,
) (resp *admin.GetWorkflowExecutionRawHistoryV2Response, retError error) {

	defer log.CapturePanic(adh.GetLogger(), &retError)
	scope, sw := adh.startRequestProfile(metrics.AdminGetWorkflowExecutionRawHistoryV2Scope)
	defer sw.Stop()

	if err := adh.validateGetWorkflowExecutionRawHistoryV2Request(
		request,
	); err != nil {
		return nil, adh.error(err, scope)
	}
	domainID, err := adh.GetDomainCache().GetDomainID(request.GetDomain())
	if err != nil {
		return nil, adh.error(err, scope)
	}
	scope = scope.Tagged(metrics.DomainTag(request.GetDomain()))

	execution := request.Execution
	var pageToken *getWorkflowRawHistoryV2Token
	var targetVersionHistory *persistence.VersionHistory
	if request.NextPageToken == nil {
		clientResp, err := adh.GetHistoryClient().GetMutableState(ctx, &types.GetMutableStateRequest{
			DomainUUID: common.StringPtr(domainID),
			Execution:  thrift.ToWorkflowExecution(execution),
		})
		response := thrift.FromGetMutableStateResponse(clientResp)
		err = thrift.FromError(err)
		if err != nil {
			return nil, adh.error(err, scope)
		}

		versionHistories := persistence.NewVersionHistoriesFromThrift(
			response.GetVersionHistories(),
		)
		targetVersionHistory, err = adh.setRequestDefaultValueAndGetTargetVersionHistory(
			request,
			versionHistories,
		)
		if err != nil {
			return nil, adh.error(err, scope)
		}

		pageToken = adh.generatePaginationToken(request, versionHistories)
	} else {
		pageToken, err = deserializeRawHistoryToken(request.NextPageToken)
		if err != nil {
			return nil, adh.error(err, scope)
		}
		versionHistories := pageToken.VersionHistories
		if versionHistories == nil {
			return nil, adh.error(&gen.BadRequestError{Message: "Invalid version histories."}, scope)
		}
		targetVersionHistory, err = adh.setRequestDefaultValueAndGetTargetVersionHistory(
			request,
			persistence.NewVersionHistoriesFromThrift(versionHistories),
		)
		if err != nil {
			return nil, adh.error(err, scope)
		}
	}

	if err := adh.validatePaginationToken(
		request,
		pageToken,
	); err != nil {
		return nil, adh.error(err, scope)
	}

	if pageToken.StartEventID+1 == pageToken.EndEventID {
		// API is exclusive-exclusive. Return empty response here.
		return &admin.GetWorkflowExecutionRawHistoryV2Response{
			HistoryBatches: []*gen.DataBlob{},
			NextPageToken:  nil, // no further pagination
			VersionHistory: targetVersionHistory.ToThrift(),
		}, nil
	}
	pageSize := int(request.GetMaximumPageSize())
	shardID := common.WorkflowIDToHistoryShard(
		execution.GetWorkflowId(),
		adh.numberOfHistoryShards,
	)
	rawHistoryResponse, err := adh.GetHistoryManager().ReadRawHistoryBranch(ctx, &persistence.ReadHistoryBranchRequest{
		BranchToken: targetVersionHistory.GetBranchToken(),
		// GetWorkflowExecutionRawHistoryV2 is exclusive exclusive.
		// ReadRawHistoryBranch is inclusive exclusive.
		MinEventID:    pageToken.StartEventID + 1,
		MaxEventID:    pageToken.EndEventID,
		PageSize:      pageSize,
		NextPageToken: pageToken.PersistenceToken,
		ShardID:       common.IntPtr(shardID),
	})
	if err != nil {
		if _, ok := err.(*gen.EntityNotExistsError); ok {
			// when no events can be returned from DB, DB layer will return
			// EntityNotExistsError, this API shall return empty response
			return &admin.GetWorkflowExecutionRawHistoryV2Response{
				HistoryBatches: []*gen.DataBlob{},
				NextPageToken:  nil, // no further pagination
				VersionHistory: targetVersionHistory.ToThrift(),
			}, nil
		}
		return nil, err
	}

	pageToken.PersistenceToken = rawHistoryResponse.NextPageToken
	size := rawHistoryResponse.Size
	// N.B. - Dual emit is required here so that we can see aggregate timer stats across all
	// domains along with the individual domains stats
	adh.GetMetricsClient().RecordTimer(metrics.AdminGetWorkflowExecutionRawHistoryScope, metrics.HistorySize, time.Duration(size))
	scope.RecordTimer(metrics.HistorySize, time.Duration(size))

	rawBlobs := rawHistoryResponse.HistoryEventBlobs
	blobs := []*gen.DataBlob{}
	for _, blob := range rawBlobs {
		blobs = append(blobs, blob.ToThrift())
	}

	result := &admin.GetWorkflowExecutionRawHistoryV2Response{
		HistoryBatches: blobs,
		VersionHistory: targetVersionHistory.ToThrift(),
	}
	if len(pageToken.PersistenceToken) == 0 {
		result.NextPageToken = nil
	} else {
		result.NextPageToken, err = serializeRawHistoryToken(pageToken)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// DescribeCluster return information about cadence deployment
func (adh *adminHandlerImpl) DescribeCluster(
	ctx context.Context,
) (resp *admin.DescribeClusterResponse, retError error) {

	defer log.CapturePanic(adh.GetLogger(), &retError)
	scope, sw := adh.startRequestProfile(metrics.AdminGetWorkflowExecutionRawHistoryV2Scope)
	defer sw.Stop()

	membershipInfo := &admin.MembershipInfo{}
	if monitor := adh.GetMembershipMonitor(); monitor != nil {
		currentHost, err := monitor.WhoAmI()
		if err != nil {
			return nil, adh.error(err, scope)
		}

		membershipInfo.CurrentHost = &admin.HostInfo{
			Identity: common.StringPtr(currentHost.Identity()),
		}

		members, err := monitor.GetReachableMembers()
		if err != nil {
			return nil, adh.error(err, scope)
		}

		membershipInfo.ReachableMembers = members

		var rings []*admin.RingInfo
		for _, role := range []string{common.FrontendServiceName, common.HistoryServiceName, common.MatchingServiceName, common.WorkerServiceName} {
			resolver, err := monitor.GetResolver(role)
			if err != nil {
				return nil, adh.error(err, scope)
			}

			var servers []*admin.HostInfo
			for _, server := range resolver.Members() {
				servers = append(servers, &admin.HostInfo{
					Identity: common.StringPtr(server.Identity()),
				})
			}

			rings = append(rings, &admin.RingInfo{
				Role:        common.StringPtr(role),
				MemberCount: common.Int32Ptr(int32(resolver.MemberCount())),
				Members:     servers,
			})
		}
		membershipInfo.Rings = rings
	}

	return &admin.DescribeClusterResponse{
		SupportedClientVersions: &gen.SupportedClientVersions{
			GoSdk:   common.StringPtr(client.SupportedGoSDKVersion),
			JavaSdk: common.StringPtr(client.SupportedJavaSDKVersion),
		},
		MembershipInfo: membershipInfo,
	}, nil
}

// GetReplicationMessages returns new replication tasks since the read level provided in the token.
func (adh *adminHandlerImpl) GetReplicationMessages(
	ctx context.Context,
	request *replicator.GetReplicationMessagesRequest,
) (resp *replicator.GetReplicationMessagesResponse, err error) {

	defer log.CapturePanic(adh.GetLogger(), &err)
	scope, sw := adh.startRequestProfile(metrics.AdminGetReplicationMessagesScope)
	defer sw.Stop()

	if request == nil {
		return nil, adh.error(errRequestNotSet, scope)
	}
	if !request.IsSetClusterName() {
		return nil, adh.error(errClusterNameNotSet, scope)
	}

	clientResp, err := adh.GetHistoryRawClient().GetReplicationMessages(ctx, thrift.ToGetReplicationMessagesRequest(request))
	resp = thrift.FromGetReplicationMessagesResponse(clientResp)
	err = thrift.FromError(err)
	if err != nil {
		return nil, adh.error(err, scope)
	}
	return resp, nil
}

// GetDomainReplicationMessages returns new domain replication tasks since last retrieved task ID.
func (adh *adminHandlerImpl) GetDomainReplicationMessages(
	ctx context.Context,
	request *replicator.GetDomainReplicationMessagesRequest,
) (resp *replicator.GetDomainReplicationMessagesResponse, err error) {

	defer log.CapturePanic(adh.GetLogger(), &err)
	scope, sw := adh.startRequestProfile(metrics.AdminGetDomainReplicationMessagesScope)
	defer sw.Stop()

	if request == nil {
		return nil, adh.error(errRequestNotSet, scope)
	}

	if adh.GetDomainReplicationQueue() == nil {
		return nil, adh.error(errors.New("domain replication queue not enabled for cluster"), scope)
	}

	lastMessageID := defaultLastMessageID
	if request.IsSetLastRetrievedMessageId() {
		lastMessageID = request.GetLastRetrievedMessageId()
	}

	if lastMessageID == defaultLastMessageID {
		clusterAckLevels, err := adh.GetDomainReplicationQueue().GetAckLevels(ctx)
		if err == nil {
			if ackLevel, ok := clusterAckLevels[request.GetClusterName()]; ok {
				lastMessageID = ackLevel
			}
		}
	}

	replicationTasks, lastMessageID, err := adh.GetDomainReplicationQueue().GetReplicationMessages(
		ctx,
		lastMessageID,
		getDomainReplicationMessageBatchSize,
	)
	if err != nil {
		return nil, adh.error(err, scope)
	}

	lastProcessedMessageID := defaultLastMessageID
	if request.IsSetLastProcessedMessageId() {
		lastProcessedMessageID = request.GetLastProcessedMessageId()
	}

	if lastProcessedMessageID != defaultLastMessageID {
		err := adh.GetDomainReplicationQueue().UpdateAckLevel(ctx, lastProcessedMessageID, request.GetClusterName())
		if err != nil {
			adh.GetLogger().Warn("Failed to update domain replication queue ack level.",
				tag.TaskID(int64(lastProcessedMessageID)),
				tag.ClusterName(request.GetClusterName()))
		}
	}

	return &replicator.GetDomainReplicationMessagesResponse{
		Messages: &replicator.ReplicationMessages{
			ReplicationTasks:       replicationTasks,
			LastRetrievedMessageId: common.Int64Ptr(int64(lastMessageID)),
		},
	}, nil
}

// GetDLQReplicationMessages returns new replication tasks based on the dlq info.
func (adh *adminHandlerImpl) GetDLQReplicationMessages(
	ctx context.Context,
	request *replicator.GetDLQReplicationMessagesRequest,
) (resp *replicator.GetDLQReplicationMessagesResponse, err error) {

	defer log.CapturePanic(adh.GetLogger(), &err)
	scope, sw := adh.startRequestProfile(metrics.AdminGetDLQReplicationMessagesScope)
	defer sw.Stop()

	if request == nil {
		return nil, adh.error(errRequestNotSet, scope)
	}
	if len(request.GetTaskInfos()) == 0 {
		return nil, adh.error(errEmptyReplicationInfo, scope)
	}

	clientResp, err := adh.GetHistoryClient().GetDLQReplicationMessages(ctx, thrift.ToGetDLQReplicationMessagesRequest(request))
	resp = thrift.FromGetDLQReplicationMessagesResponse(clientResp)
	err = thrift.FromError(err)
	if err != nil {
		return nil, adh.error(err, scope)
	}
	return resp, nil
}

// ReapplyEvents applies stale events to the current workflow and the current run
func (adh *adminHandlerImpl) ReapplyEvents(
	ctx context.Context,
	request *gen.ReapplyEventsRequest,
) (err error) {

	defer log.CapturePanic(adh.GetLogger(), &err)
	scope, sw := adh.startRequestProfile(metrics.AdminReapplyEventsScope)
	defer sw.Stop()

	if request == nil {
		return adh.error(errRequestNotSet, scope)
	}
	if request.DomainName == nil || request.GetDomainName() == "" {
		return adh.error(errDomainNotSet, scope)
	}
	if request.WorkflowExecution == nil {
		return adh.error(errExecutionNotSet, scope)
	}
	if request.GetWorkflowExecution().GetWorkflowId() == "" {
		return adh.error(errWorkflowIDNotSet, scope)
	}
	if request.GetEvents() == nil {
		return adh.error(errWorkflowIDNotSet, scope)
	}
	domainEntry, err := adh.GetDomainCache().GetDomain(request.GetDomainName())
	if err != nil {
		return adh.error(err, scope)
	}

	err = adh.GetHistoryClient().ReapplyEvents(ctx, &types.HistoryReapplyEventsRequest{
		DomainUUID: common.StringPtr(domainEntry.GetInfo().ID),
		Request:    thrift.ToReapplyEventsRequest(request),
	})
	err = thrift.FromError(err)
	if err != nil {
		return adh.error(err, scope)
	}
	return nil
}

// ReadDLQMessages reads messages from DLQ
func (adh *adminHandlerImpl) ReadDLQMessages(
	ctx context.Context,
	request *replicator.ReadDLQMessagesRequest,
) (resp *replicator.ReadDLQMessagesResponse, err error) {

	defer log.CapturePanic(adh.GetLogger(), &err)
	scope, sw := adh.startRequestProfile(metrics.AdminReadDLQMessagesScope)
	defer sw.Stop()

	if request == nil {
		return nil, adh.error(errRequestNotSet, scope)
	}

	if !request.IsSetType() {
		return nil, adh.error(errEmptyQueueType, scope)
	}

	if request.GetMaximumPageSize() <= 0 {
		request.MaximumPageSize = common.Int32Ptr(common.ReadDLQMessagesPageSize)
	}

	if !request.IsSetInclusiveEndMessageID() {
		request.InclusiveEndMessageID = common.Int64Ptr(common.EndMessageID)
	}

	var tasks []*replicator.ReplicationTask
	var token []byte
	var op func() error
	switch request.GetType() {
	case replicator.DLQTypeReplication:
		clientResp, err := adh.GetHistoryClient().ReadDLQMessages(ctx, thrift.ToReadDLQMessagesRequest(request))
		return thrift.FromReadDLQMessagesResponse(clientResp), thrift.FromError(err)
	case replicator.DLQTypeDomain:
		op = func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				var err error
				tasks, token, err = adh.domainDLQHandler.Read(
					ctx,
					request.GetInclusiveEndMessageID(),
					int(request.GetMaximumPageSize()),
					request.GetNextPageToken())
				return err
			}
		}
	default:
		return nil, &gen.BadRequestError{Message: "The DLQ type is not supported."}
	}
	err = backoff.Retry(op, adminServiceRetryPolicy, common.IsServiceTransientError)
	if err != nil {
		return nil, adh.error(err, scope)
	}

	return &replicator.ReadDLQMessagesResponse{
		ReplicationTasks: tasks,
		NextPageToken:    token,
	}, nil
}

// PurgeDLQMessages purge messages from DLQ
func (adh *adminHandlerImpl) PurgeDLQMessages(
	ctx context.Context,
	request *replicator.PurgeDLQMessagesRequest,
) (err error) {

	defer log.CapturePanic(adh.GetLogger(), &err)
	scope, sw := adh.startRequestProfile(metrics.AdminPurgeDLQMessagesScope)
	defer sw.Stop()

	if request == nil {
		return adh.error(errRequestNotSet, scope)
	}

	if !request.IsSetType() {
		return adh.error(errEmptyQueueType, scope)
	}

	if !request.IsSetInclusiveEndMessageID() {
		request.InclusiveEndMessageID = common.Int64Ptr(endMessageID)
	}

	var op func() error
	switch request.GetType() {
	case replicator.DLQTypeReplication:
		err = adh.GetHistoryClient().PurgeDLQMessages(ctx, thrift.ToPurgeDLQMessagesRequest(request))
		return thrift.FromError(err)
	case replicator.DLQTypeDomain:
		op = func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return adh.domainDLQHandler.Purge(
					ctx,
					request.GetInclusiveEndMessageID(),
				)
			}
		}
	default:
		return &gen.BadRequestError{Message: "The DLQ type is not supported."}
	}
	err = backoff.Retry(op, adminServiceRetryPolicy, common.IsServiceTransientError)
	if err != nil {
		return adh.error(err, scope)
	}

	return nil
}

// MergeDLQMessages merges DLQ messages
func (adh *adminHandlerImpl) MergeDLQMessages(
	ctx context.Context,
	request *replicator.MergeDLQMessagesRequest,
) (resp *replicator.MergeDLQMessagesResponse, err error) {

	defer log.CapturePanic(adh.GetLogger(), &err)
	scope, sw := adh.startRequestProfile(metrics.AdminMergeDLQMessagesScope)
	defer sw.Stop()

	if request == nil {
		return nil, adh.error(errRequestNotSet, scope)
	}

	if !request.IsSetType() {
		return nil, adh.error(errEmptyQueueType, scope)
	}

	if !request.IsSetInclusiveEndMessageID() {
		request.InclusiveEndMessageID = common.Int64Ptr(endMessageID)
	}

	var token []byte
	var op func() error
	switch request.GetType() {
	case replicator.DLQTypeReplication:
		clientResp, err := adh.GetHistoryClient().MergeDLQMessages(ctx, thrift.ToMergeDLQMessagesRequest(request))
		return thrift.FromMergeDLQMessagesResponse(clientResp), thrift.FromError(err)
	case replicator.DLQTypeDomain:

		op = func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				var err error
				token, err = adh.domainDLQHandler.Merge(
					ctx,
					request.GetInclusiveEndMessageID(),
					int(request.GetMaximumPageSize()),
					request.GetNextPageToken(),
				)
				return err
			}
		}
	default:
		return nil, &gen.BadRequestError{Message: "The DLQ type is not supported."}
	}
	err = backoff.Retry(op, adminServiceRetryPolicy, common.IsServiceTransientError)
	if err != nil {
		return nil, adh.error(err, scope)
	}

	return &replicator.MergeDLQMessagesResponse{
		NextPageToken: token,
	}, nil
}

// RefreshWorkflowTasks re-generates the workflow tasks
func (adh *adminHandlerImpl) RefreshWorkflowTasks(
	ctx context.Context,
	request *gen.RefreshWorkflowTasksRequest,
) (err error) {
	defer log.CapturePanic(adh.GetLogger(), &err)
	scope, sw := adh.startRequestProfile(metrics.AdminRefreshWorkflowTasksScope)
	defer sw.Stop()

	if request == nil {
		return adh.error(errRequestNotSet, scope)
	}
	if err := validateExecution(request.Execution); err != nil {
		return adh.error(err, scope)
	}
	domainEntry, err := adh.GetDomainCache().GetDomain(request.GetDomain())
	if err != nil {
		return adh.error(err, scope)
	}

	err = adh.GetHistoryClient().RefreshWorkflowTasks(ctx, &types.HistoryRefreshWorkflowTasksRequest{
		DomainUIID: common.StringPtr(domainEntry.GetInfo().ID),
		Request:    thrift.ToRefreshWorkflowTasksRequest(request),
	})
	err = thrift.FromError(err)
	if err != nil {
		return adh.error(err, scope)
	}
	return nil
}

// ResendReplicationTasks requests replication task from remote cluster
func (adh *adminHandlerImpl) ResendReplicationTasks(
	ctx context.Context,
	request *admin.ResendReplicationTasksRequest,
) (err error) {
	defer log.CapturePanic(adh.GetLogger(), &err)
	scope, sw := adh.startRequestProfile(metrics.AdminResendReplicationTasksScope)
	defer sw.Stop()

	if request == nil {
		return adh.error(errRequestNotSet, scope)
	}
	resender := ndc.NewHistoryResender(
		adh.GetDomainCache(),
		adh.GetRemoteAdminClient(request.GetRemoteCluster()),
		func(ctx context.Context, request *h.ReplicateEventsV2Request) error {
			err := adh.GetHistoryClient().ReplicateEventsV2(ctx, thrift.ToReplicateEventsV2Request(request))
			return thrift.FromError(err)
		},
		adh.eventSerializder,
		nil,
		nil,
		adh.GetLogger(),
	)
	return resender.SendSingleWorkflowHistory(
		request.GetDomainID(),
		request.GetWorkflowID(),
		request.GetRunID(),
		resendStartEventID,
		request.StartVersion,
		nil,
		nil,
	)
}

func (adh *adminHandlerImpl) validateGetWorkflowExecutionRawHistoryV2Request(
	request *admin.GetWorkflowExecutionRawHistoryV2Request,
) error {

	execution := request.Execution
	if len(execution.GetWorkflowId()) == 0 {
		return &gen.BadRequestError{Message: "Invalid WorkflowID."}
	}
	// TODO currently, this API is only going to be used by re-send history events
	// to remote cluster if kafka is lossy again, in the future, this API can be used
	// by CLI and client, then empty runID (meaning the current workflow) should be allowed
	if len(execution.GetRunId()) == 0 || uuid.Parse(execution.GetRunId()) == nil {
		return &gen.BadRequestError{Message: "Invalid RunID."}
	}

	pageSize := int(request.GetMaximumPageSize())
	if pageSize <= 0 {
		return &gen.BadRequestError{Message: "Invalid PageSize."}
	}

	if request.StartEventId == nil &&
		request.StartEventVersion == nil &&
		request.EndEventId == nil &&
		request.EndEventVersion == nil {
		return &gen.BadRequestError{Message: "Invalid event query range."}
	}

	if (request.StartEventId != nil && request.StartEventVersion == nil) ||
		(request.StartEventId == nil && request.StartEventVersion != nil) {
		return &gen.BadRequestError{Message: "Invalid start event id and start event version combination."}
	}

	if (request.EndEventId != nil && request.EndEventVersion == nil) ||
		(request.EndEventId == nil && request.EndEventVersion != nil) {
		return &gen.BadRequestError{Message: "Invalid end event id and end event version combination."}
	}
	return nil
}

func (adh *adminHandlerImpl) validateConfigForAdvanceVisibility() error {
	if adh.params.ESConfig == nil || adh.params.ESClient == nil {
		return errors.New("ES related config not found")
	}
	return nil
}

func (adh *adminHandlerImpl) setRequestDefaultValueAndGetTargetVersionHistory(
	request *admin.GetWorkflowExecutionRawHistoryV2Request,
	versionHistories *persistence.VersionHistories,
) (*persistence.VersionHistory, error) {

	targetBranch, err := versionHistories.GetCurrentVersionHistory()
	if err != nil {
		return nil, err
	}
	firstItem, err := targetBranch.GetFirstItem()
	if err != nil {
		return nil, err
	}
	lastItem, err := targetBranch.GetLastItem()
	if err != nil {
		return nil, err
	}

	if request.StartEventId == nil || request.StartEventVersion == nil {
		// If start event is not set, get the events from the first event
		// As the API is exclusive-exclusive, use first event id - 1 here
		request.StartEventId = common.Int64Ptr(common.FirstEventID - 1)
		request.StartEventVersion = common.Int64Ptr(firstItem.GetVersion())
	}
	if request.EndEventId == nil || request.EndEventVersion == nil {
		// If end event is not set, get the events until the end event
		// As the API is exclusive-exclusive, use end event id + 1 here
		request.EndEventId = common.Int64Ptr(lastItem.GetEventID() + 1)
		request.EndEventVersion = common.Int64Ptr(lastItem.GetVersion())
	}

	if request.GetStartEventId() < 0 {
		return nil, &gen.BadRequestError{Message: "Invalid FirstEventID && NextEventID combination."}
	}

	// get branch based on the end event if end event is defined in the request
	if request.GetEndEventId() == lastItem.GetEventID()+1 &&
		request.GetEndEventVersion() == lastItem.GetVersion() {
		// this is a special case, target branch remains the same
	} else {
		endItem := persistence.NewVersionHistoryItem(request.GetEndEventId(), request.GetEndEventVersion())
		idx, err := versionHistories.FindFirstVersionHistoryIndexByItem(endItem)
		if err != nil {
			return nil, err
		}

		targetBranch, err = versionHistories.GetVersionHistory(idx)
		if err != nil {
			return nil, err
		}
	}

	startItem := persistence.NewVersionHistoryItem(request.GetStartEventId(), request.GetStartEventVersion())
	// If the request start event is defined. The start event may be on a different branch as current branch.
	// We need to find the LCA of the start event and the current branch.
	if request.GetStartEventId() == common.FirstEventID-1 &&
		request.GetStartEventVersion() == firstItem.GetVersion() {
		// this is a special case, start event is on the same branch as target branch
	} else {
		if !targetBranch.ContainsItem(startItem) {
			idx, err := versionHistories.FindFirstVersionHistoryIndexByItem(startItem)
			if err != nil {
				return nil, err
			}
			startBranch, err := versionHistories.GetVersionHistory(idx)
			if err != nil {
				return nil, err
			}
			startItem, err = targetBranch.FindLCAItem(startBranch)
			if err != nil {
				return nil, err
			}
			request.StartEventId = common.Int64Ptr(startItem.GetEventID())
			request.StartEventVersion = common.Int64Ptr(startItem.GetVersion())
		}
	}

	return targetBranch, nil
}

func (adh *adminHandlerImpl) generatePaginationToken(
	request *admin.GetWorkflowExecutionRawHistoryV2Request,
	versionHistories *persistence.VersionHistories,
) *getWorkflowRawHistoryV2Token {

	execution := request.Execution
	return &getWorkflowRawHistoryV2Token{
		DomainName:        request.GetDomain(),
		WorkflowID:        execution.GetWorkflowId(),
		RunID:             execution.GetRunId(),
		StartEventID:      request.GetStartEventId(),
		StartEventVersion: request.GetStartEventVersion(),
		EndEventID:        request.GetEndEventId(),
		EndEventVersion:   request.GetEndEventVersion(),
		VersionHistories:  versionHistories.ToThrift(),
		PersistenceToken:  nil, // this is the initialized value
	}
}

func (adh *adminHandlerImpl) validatePaginationToken(
	request *admin.GetWorkflowExecutionRawHistoryV2Request,
	token *getWorkflowRawHistoryV2Token,
) error {

	execution := request.Execution
	if request.GetDomain() != token.DomainName ||
		execution.GetWorkflowId() != token.WorkflowID ||
		execution.GetRunId() != token.RunID ||
		request.GetStartEventId() != token.StartEventID ||
		request.GetStartEventVersion() != token.StartEventVersion ||
		request.GetEndEventId() != token.EndEventID ||
		request.GetEndEventVersion() != token.EndEventVersion {
		return &gen.BadRequestError{Message: "Invalid pagination token."}
	}
	return nil
}

// startRequestProfile initiates recording of request metrics
func (adh *adminHandlerImpl) startRequestProfile(scope int) (metrics.Scope, metrics.Stopwatch) {
	metricsScope := adh.GetMetricsClient().Scope(scope)
	sw := metricsScope.StartTimer(metrics.CadenceLatency)
	metricsScope.IncCounter(metrics.CadenceRequests)
	return metricsScope, sw
}

func (adh *adminHandlerImpl) error(err error, scope metrics.Scope) error {
	switch err.(type) {
	case *gen.InternalServiceError:
		adh.GetLogger().Error("Internal service error", tag.Error(err))
		scope.IncCounter(metrics.CadenceFailures)
		return err
	case *gen.BadRequestError:
		scope.IncCounter(metrics.CadenceErrBadRequestCounter)
		return err
	case *gen.ServiceBusyError:
		scope.IncCounter(metrics.CadenceErrServiceBusyCounter)
		return err
	case *gen.EntityNotExistsError:
		return err
	default:
		adh.GetLogger().Error("Uncategorized error", tag.Error(err))
		scope.IncCounter(metrics.CadenceFailures)
		return &gen.InternalServiceError{Message: err.Error()}
	}
}

func convertIndexedValueTypeToESDataType(valueType gen.IndexedValueType) string {
	switch valueType {
	case gen.IndexedValueTypeString:
		return "text"
	case gen.IndexedValueTypeKeyword:
		return "keyword"
	case gen.IndexedValueTypeInt:
		return "long"
	case gen.IndexedValueTypeDouble:
		return "double"
	case gen.IndexedValueTypeBool:
		return "boolean"
	case gen.IndexedValueTypeDatetime:
		return "date"
	default:
		return ""
	}
}

func serializeRawHistoryToken(token *getWorkflowRawHistoryV2Token) ([]byte, error) {
	if token == nil {
		return nil, nil
	}

	bytes, err := json.Marshal(token)
	return bytes, err
}

func deserializeRawHistoryToken(bytes []byte) (*getWorkflowRawHistoryV2Token, error) {
	token := &getWorkflowRawHistoryV2Token{}
	err := json.Unmarshal(bytes, token)
	return token, err
}
