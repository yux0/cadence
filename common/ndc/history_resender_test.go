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

package ndc

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/uber/cadence/.gen/go/history"
	"github.com/uber/cadence/.gen/go/history/historyservicetest"
	"github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/client/admin"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/cluster"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/loggerimpl"
	"github.com/uber/cadence/common/mocks"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/reconciliation/entity"
	"github.com/uber/cadence/common/reconciliation/invariant"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/common/types/mapper/thrift"
)

type (
	historyResenderSuite struct {
		suite.Suite
		*require.Assertions

		controller        *gomock.Controller
		mockDomainCache   *cache.MockDomainCache
		mockAdminClient   *admin.MockClient
		mockHistoryClient *historyservicetest.MockClient

		domainID   string
		domainName string

		mockClusterMetadata *mocks.ClusterMetadata

		serializer persistence.PayloadSerializer
		logger     log.Logger

		rereplicator *HistoryResenderImpl
	}
)

func TestHistoryResenderSuite(t *testing.T) {
	s := new(historyResenderSuite)
	suite.Run(t, s)
}

func (s *historyResenderSuite) SetupSuite() {
}

func (s *historyResenderSuite) TearDownSuite() {

}

func (s *historyResenderSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	s.controller = gomock.NewController(s.T())
	s.mockAdminClient = admin.NewMockClient(s.controller)
	s.mockHistoryClient = historyservicetest.NewMockClient(s.controller)
	s.mockDomainCache = cache.NewMockDomainCache(s.controller)

	s.logger = loggerimpl.NewDevelopmentForTest(s.Suite)
	s.mockClusterMetadata = &mocks.ClusterMetadata{}
	s.mockClusterMetadata.On("IsGlobalDomainEnabled").Return(true)

	s.domainID = uuid.New()
	s.domainName = "some random domain name"
	domainEntry := cache.NewGlobalDomainCacheEntryForTest(
		&persistence.DomainInfo{ID: s.domainID, Name: s.domainName},
		&persistence.DomainConfig{Retention: 1},
		&persistence.DomainReplicationConfig{
			ActiveClusterName: cluster.TestCurrentClusterName,
			Clusters: []*persistence.ClusterReplicationConfig{
				{ClusterName: cluster.TestCurrentClusterName},
				{ClusterName: cluster.TestAlternativeClusterName},
			},
		},
		1234,
		nil,
	)
	s.mockDomainCache.EXPECT().GetDomainByID(s.domainID).Return(domainEntry, nil).AnyTimes()
	s.mockDomainCache.EXPECT().GetDomain(s.domainName).Return(domainEntry, nil).AnyTimes()
	s.serializer = persistence.NewPayloadSerializer()

	s.rereplicator = NewHistoryResender(
		s.mockDomainCache,
		s.mockAdminClient,
		func(ctx context.Context, request *history.ReplicateEventsV2Request) error {
			return s.mockHistoryClient.ReplicateEventsV2(ctx, request)
		},
		persistence.NewPayloadSerializer(),
		nil,
		nil,
		s.logger,
	)
}

func (s *historyResenderSuite) TearDownTest() {
	s.controller.Finish()
}

func (s *historyResenderSuite) TestSendSingleWorkflowHistory() {
	workflowID := "some random workflow ID"
	runID := uuid.New()
	startEventID := int64(123)
	startEventVersion := int64(100)
	token := []byte{1}
	pageSize := defaultPageSize
	eventBatch := []*shared.HistoryEvent{
		{
			EventId:   common.Int64Ptr(2),
			Version:   common.Int64Ptr(123),
			Timestamp: common.Int64Ptr(time.Now().UnixNano()),
			EventType: shared.EventTypeDecisionTaskScheduled.Ptr(),
		},
		{
			EventId:   common.Int64Ptr(3),
			Version:   common.Int64Ptr(123),
			Timestamp: common.Int64Ptr(time.Now().UnixNano()),
			EventType: shared.EventTypeDecisionTaskStarted.Ptr(),
		},
	}
	blob := s.serializeEvents(eventBatch)
	versionHistoryItems := []*shared.VersionHistoryItem{
		{
			EventID: common.Int64Ptr(1),
			Version: common.Int64Ptr(1),
		},
	}

	s.mockAdminClient.EXPECT().GetWorkflowExecutionRawHistoryV2(
		gomock.Any(),
		&types.GetWorkflowExecutionRawHistoryV2Request{
			Domain: common.StringPtr(s.domainName),
			Execution: &types.WorkflowExecution{
				WorkflowID: common.StringPtr(workflowID),
				RunID:      common.StringPtr(runID),
			},
			StartEventID:      common.Int64Ptr(startEventID),
			StartEventVersion: common.Int64Ptr(startEventVersion),
			MaximumPageSize:   common.Int32Ptr(pageSize),
			NextPageToken:     nil,
		}).Return(&types.GetWorkflowExecutionRawHistoryV2Response{
		HistoryBatches: []*types.DataBlob{thrift.ToDataBlob(blob)},
		NextPageToken:  token,
		VersionHistory: &types.VersionHistory{
			Items: thrift.ToVersionHistoryItemArray(versionHistoryItems),
		},
	}, nil).Times(1)

	s.mockAdminClient.EXPECT().GetWorkflowExecutionRawHistoryV2(
		gomock.Any(),
		&types.GetWorkflowExecutionRawHistoryV2Request{
			Domain: common.StringPtr(s.domainName),
			Execution: &types.WorkflowExecution{
				WorkflowID: common.StringPtr(workflowID),
				RunID:      common.StringPtr(runID),
			},
			StartEventID:      common.Int64Ptr(startEventID),
			StartEventVersion: common.Int64Ptr(startEventVersion),
			MaximumPageSize:   common.Int32Ptr(pageSize),
			NextPageToken:     token,
		}).Return(&types.GetWorkflowExecutionRawHistoryV2Response{
		HistoryBatches: []*types.DataBlob{thrift.ToDataBlob(blob)},
		NextPageToken:  nil,
		VersionHistory: &types.VersionHistory{
			Items: thrift.ToVersionHistoryItemArray(versionHistoryItems),
		},
	}, nil).Times(1)

	s.mockHistoryClient.EXPECT().ReplicateEventsV2(
		gomock.Any(),
		&history.ReplicateEventsV2Request{
			DomainUUID: common.StringPtr(s.domainID),
			WorkflowExecution: &shared.WorkflowExecution{
				WorkflowId: common.StringPtr(workflowID),
				RunId:      common.StringPtr(runID),
			},
			VersionHistoryItems: versionHistoryItems,
			Events:              blob,
		}).Return(nil).Times(2)

	err := s.rereplicator.SendSingleWorkflowHistory(
		s.domainID,
		workflowID,
		runID,
		common.Int64Ptr(startEventID),
		common.Int64Ptr(startEventVersion),
		nil,
		nil,
	)

	s.Nil(err)
}

func (s *historyResenderSuite) TestCreateReplicateRawEventsRequest() {
	workflowID := "some random workflow ID"
	runID := uuid.New()
	blob := &shared.DataBlob{
		EncodingType: shared.EncodingTypeThriftRW.Ptr(),
		Data:         []byte("some random history blob"),
	}
	versionHistoryItems := []*shared.VersionHistoryItem{
		{
			EventID: common.Int64Ptr(1),
			Version: common.Int64Ptr(1),
		},
	}

	s.Equal(&history.ReplicateEventsV2Request{
		DomainUUID: common.StringPtr(s.domainID),
		WorkflowExecution: &shared.WorkflowExecution{
			WorkflowId: common.StringPtr(workflowID),
			RunId:      common.StringPtr(runID),
		},
		VersionHistoryItems: versionHistoryItems,
		Events:              blob,
	}, s.rereplicator.createReplicationRawRequest(
		s.domainID,
		workflowID,
		runID,
		blob,
		versionHistoryItems))
}

func (s *historyResenderSuite) TestSendReplicationRawRequest() {
	workflowID := "some random workflow ID"
	runID := uuid.New()
	item := &shared.VersionHistoryItem{
		EventID: common.Int64Ptr(1),
		Version: common.Int64Ptr(1),
	}
	request := &history.ReplicateEventsV2Request{
		DomainUUID: common.StringPtr(s.domainID),
		WorkflowExecution: &shared.WorkflowExecution{
			WorkflowId: common.StringPtr(workflowID),
			RunId:      common.StringPtr(runID),
		},
		Events: &shared.DataBlob{
			EncodingType: shared.EncodingTypeThriftRW.Ptr(),
			Data:         []byte("some random history blob"),
		},
		VersionHistoryItems: []*shared.VersionHistoryItem{item},
	}

	s.mockHistoryClient.EXPECT().ReplicateEventsV2(gomock.Any(), request).Return(nil).Times(1)
	err := s.rereplicator.sendReplicationRawRequest(context.Background(), request)
	s.Nil(err)
}

func (s *historyResenderSuite) TestSendReplicationRawRequest_Err() {
	workflowID := "some random workflow ID"
	runID := uuid.New()
	item := &shared.VersionHistoryItem{
		EventID: common.Int64Ptr(1),
		Version: common.Int64Ptr(1),
	}
	request := &history.ReplicateEventsV2Request{
		DomainUUID: common.StringPtr(s.domainID),
		WorkflowExecution: &shared.WorkflowExecution{
			WorkflowId: common.StringPtr(workflowID),
			RunId:      common.StringPtr(runID),
		},
		Events: &shared.DataBlob{
			EncodingType: shared.EncodingTypeThriftRW.Ptr(),
			Data:         []byte("some random history blob"),
		},
		VersionHistoryItems: []*shared.VersionHistoryItem{item},
	}
	retryErr := &shared.RetryTaskV2Error{
		DomainId:   common.StringPtr(s.domainID),
		WorkflowId: common.StringPtr(workflowID),
		RunId:      common.StringPtr(runID),
	}

	s.mockHistoryClient.EXPECT().ReplicateEventsV2(gomock.Any(), request).Return(retryErr).Times(1)
	err := s.rereplicator.sendReplicationRawRequest(context.Background(), request)
	s.Equal(retryErr, err)
}

func (s *historyResenderSuite) TestGetHistory() {
	workflowID := "some random workflow ID"
	runID := uuid.New()
	startEventID := int64(123)
	endEventID := int64(345)
	version := int64(20)
	nextTokenIn := []byte("some random next token in")
	nextTokenOut := []byte("some random next token out")
	pageSize := int32(59)
	blob := []byte("some random events blob")
	encodingTypeThriftRW := types.EncodingTypeThriftRW

	response := &types.GetWorkflowExecutionRawHistoryV2Response{
		HistoryBatches: []*types.DataBlob{&types.DataBlob{
			EncodingType: &encodingTypeThriftRW,
			Data:         blob,
		}},
		NextPageToken: nextTokenOut,
	}
	s.mockAdminClient.EXPECT().GetWorkflowExecutionRawHistoryV2(gomock.Any(), &types.GetWorkflowExecutionRawHistoryV2Request{
		Domain: common.StringPtr(s.domainName),
		Execution: &types.WorkflowExecution{
			WorkflowID: common.StringPtr(workflowID),
			RunID:      common.StringPtr(runID),
		},
		StartEventID:      common.Int64Ptr(startEventID),
		StartEventVersion: common.Int64Ptr(version),
		EndEventID:        common.Int64Ptr(endEventID),
		EndEventVersion:   common.Int64Ptr(version),
		MaximumPageSize:   common.Int32Ptr(pageSize),
		NextPageToken:     nextTokenIn,
	}).Return(response, nil).Times(1)

	out, err := s.rereplicator.getHistory(
		context.Background(),
		s.domainID,
		workflowID,
		runID,
		&startEventID,
		&version,
		&endEventID,
		&version,
		nextTokenIn,
		pageSize)
	s.Nil(err)
	s.Equal(response, thrift.ToGetWorkflowExecutionRawHistoryV2Response(out))
}

func (s *historyResenderSuite) TestCurrentExecutionCheck() {
	domainID := uuid.New()
	workflowID1 := uuid.New()
	workflowID2 := uuid.New()
	runID := uuid.New()
	invariantMock := invariant.NewMockInvariant(s.controller)
	s.rereplicator = NewHistoryResender(
		s.mockDomainCache,
		s.mockAdminClient,
		func(ctx context.Context, request *history.ReplicateEventsV2Request) error {
			return s.mockHistoryClient.ReplicateEventsV2(ctx, request)
		},
		persistence.NewPayloadSerializer(),
		nil,
		invariantMock,
		s.logger,
	)
	execution1 := &entity.CurrentExecution{
		Execution: entity.Execution{
			DomainID:   domainID,
			WorkflowID: workflowID1,
			State:      persistence.WorkflowStateRunning,
		},
	}
	execution2 := &entity.CurrentExecution{
		Execution: entity.Execution{
			DomainID:   domainID,
			WorkflowID: workflowID2,
			State:      persistence.WorkflowStateRunning,
		},
	}
	invariantMock.EXPECT().Check(gomock.Any(), execution1).Return(invariant.CheckResult{
		CheckResultType: invariant.CheckResultTypeCorrupted,
	}).Times(1)
	invariantMock.EXPECT().Check(gomock.Any(), execution2).Return(invariant.CheckResult{
		CheckResultType: invariant.CheckResultTypeHealthy,
	}).Times(1)
	invariantMock.EXPECT().Fix(gomock.Any(), gomock.Any()).Return(invariant.FixResult{}).Times(1)

	skipTask := s.rereplicator.fixCurrentExecution(context.Background(), domainID, workflowID1, runID)
	s.False(skipTask)
	skipTask = s.rereplicator.fixCurrentExecution(context.Background(), domainID, workflowID2, runID)
	s.True(skipTask)
}

func (s *historyResenderSuite) serializeEvents(events []*shared.HistoryEvent) *shared.DataBlob {
	blob, err := s.serializer.SerializeBatchEvents(events, common.EncodingTypeThriftRW)
	s.Nil(err)
	return &shared.DataBlob{
		EncodingType: shared.EncodingTypeThriftRW.Ptr(),
		Data:         blob.Data,
	}
}
