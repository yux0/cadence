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
	ctx "context"

	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/cluster"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/mocks"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/service/history/config"
	"github.com/uber/cadence/service/history/constants"
	"github.com/uber/cadence/service/history/execution"
	"github.com/uber/cadence/service/history/reset"
	"github.com/uber/cadence/service/history/shard"
)

type (
	transactionManagerSuite struct {
		suite.Suite
		*require.Assertions

		controller           *gomock.Controller
		mockShard            *shard.TestContext
		mockCreateManager    *MocktransactionManagerForNewWorkflow
		mockUpdateManager    *MocktransactionManagerForExistingWorkflow
		mockEventsReapplier  *MockEventsReapplier
		mockWorkflowResetter *reset.MockWorkflowResetter
		mockClusterMetadata  *cluster.MockMetadata

		mockExecutionManager *mocks.ExecutionManager

		logger      log.Logger
		domainEntry *cache.DomainCacheEntry

		transactionManager *transactionManagerImpl
	}
)

func TestTransactionManagerSuite(t *testing.T) {
	s := new(transactionManagerSuite)
	suite.Run(t, s)
}

func (s *transactionManagerSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	s.controller = gomock.NewController(s.T())
	s.mockCreateManager = NewMocktransactionManagerForNewWorkflow(s.controller)
	s.mockUpdateManager = NewMocktransactionManagerForExistingWorkflow(s.controller)
	s.mockEventsReapplier = NewMockEventsReapplier(s.controller)
	s.mockWorkflowResetter = reset.NewMockWorkflowResetter(s.controller)

	s.mockShard = shard.NewTestContext(
		s.controller,
		&persistence.ShardInfo{
			ShardID:          10,
			RangeID:          1,
			TransferAckLevel: 0,
		},
		config.NewForTest(),
	)

	s.mockClusterMetadata = s.mockShard.Resource.ClusterMetadata
	s.mockExecutionManager = s.mockShard.Resource.ExecutionMgr

	s.logger = s.mockShard.GetLogger()
	s.domainEntry = constants.TestGlobalDomainEntry

	s.transactionManager = newTransactionManager(s.mockShard, execution.NewCache(s.mockShard), s.mockEventsReapplier, s.logger)
	s.transactionManager.createManager = s.mockCreateManager
	s.transactionManager.updateManager = s.mockUpdateManager
	s.transactionManager.workflowResetter = s.mockWorkflowResetter
}

func (s *transactionManagerSuite) TearDownTest() {
	s.controller.Finish()
	s.mockShard.Finish(s.T())
}

func (s *transactionManagerSuite) TestCreateWorkflow() {
	ctx := ctx.Background()
	now := time.Now()
	targetWorkflow := execution.NewMockWorkflow(s.controller)

	s.mockCreateManager.EXPECT().dispatchForNewWorkflow(
		ctx, now, targetWorkflow,
	).Return(nil).Times(1)

	err := s.transactionManager.createWorkflow(ctx, now, targetWorkflow)
	s.NoError(err)
}

func (s *transactionManagerSuite) TestUpdateWorkflow() {
	ctx := ctx.Background()
	now := time.Now()
	isWorkflowRebuilt := true
	targetWorkflow := execution.NewMockWorkflow(s.controller)
	newWorkflow := execution.NewMockWorkflow(s.controller)

	s.mockUpdateManager.EXPECT().dispatchForExistingWorkflow(
		ctx, now, isWorkflowRebuilt, targetWorkflow, newWorkflow,
	).Return(nil).Times(1)

	err := s.transactionManager.updateWorkflow(ctx, now, isWorkflowRebuilt, targetWorkflow, newWorkflow)
	s.NoError(err)
}

func (s *transactionManagerSuite) TestBackfillWorkflow_CurrentWorkflow_Active_Open() {
	ctx := ctx.Background()
	now := time.Now()
	releaseCalled := false
	runID := uuid.New()

	workflow := execution.NewMockWorkflow(s.controller)
	context := execution.NewMockContext(s.controller)
	mutableState := execution.NewMockMutableState(s.controller)
	var releaseFn execution.ReleaseFunc = func(error) { releaseCalled = true }

	workflowEvents := &persistence.WorkflowEvents{
		Events: []*shared.HistoryEvent{{EventId: common.Int64Ptr(1)}},
	}

	workflow.EXPECT().GetContext().Return(context).AnyTimes()
	workflow.EXPECT().GetMutableState().Return(mutableState).AnyTimes()
	workflow.EXPECT().GetReleaseFn().Return(releaseFn).AnyTimes()

	s.mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestCurrentClusterName).AnyTimes()
	s.mockClusterMetadata.EXPECT().ClusterNameForFailoverVersion(s.domainEntry.GetFailoverVersion()).Return(cluster.TestCurrentClusterName).AnyTimes()

	s.mockEventsReapplier.EXPECT().ReapplyEvents(ctx, mutableState, workflowEvents.Events, runID).Return(workflowEvents.Events, nil).Times(1)

	mutableState.EXPECT().IsCurrentWorkflowGuaranteed().Return(true).AnyTimes()
	mutableState.EXPECT().IsWorkflowExecutionRunning().Return(true).AnyTimes()
	mutableState.EXPECT().GetDomainEntry().Return(s.domainEntry).AnyTimes()
	mutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{RunID: runID}).Times(1)
	context.EXPECT().PersistNonFirstWorkflowEvents(gomock.Any(), workflowEvents).Return(int64(0), nil).Times(1)
	context.EXPECT().UpdateWorkflowExecutionWithNew(
		gomock.Any(), now, persistence.UpdateWorkflowModeUpdateCurrent, nil, nil, execution.TransactionPolicyActive, (*execution.TransactionPolicy)(nil),
	).Return(nil).Times(1)
	err := s.transactionManager.backfillWorkflow(ctx, now, workflow, workflowEvents)
	s.NoError(err)
	s.True(releaseCalled)
}

func (s *transactionManagerSuite) TestBackfillWorkflow_CurrentWorkflow_Active_Closed() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	runID := "some random run ID"
	lastDecisionTaskStartedEventID := int64(9999)
	nextEventID := lastDecisionTaskStartedEventID * 2
	lastDecisionTaskStartedVersion := s.domainEntry.GetFailoverVersion()
	versionHistory := persistence.NewVersionHistory([]byte("branch token"), []*persistence.VersionHistoryItem{
		{EventID: lastDecisionTaskStartedEventID, Version: lastDecisionTaskStartedVersion},
	})
	histories := persistence.NewVersionHistories(versionHistory)

	releaseCalled := false

	workflow := execution.NewMockWorkflow(s.controller)
	context := execution.NewMockContext(s.controller)
	mutableState := execution.NewMockMutableState(s.controller)
	var releaseFn execution.ReleaseFunc = func(error) { releaseCalled = true }

	workflowEvents := &persistence.WorkflowEvents{}

	workflow.EXPECT().GetContext().Return(context).AnyTimes()
	workflow.EXPECT().GetMutableState().Return(mutableState).AnyTimes()
	workflow.EXPECT().GetReleaseFn().Return(releaseFn).AnyTimes()

	s.mockClusterMetadata.EXPECT().ClusterNameForFailoverVersion(s.domainEntry.GetFailoverVersion()).Return(cluster.TestCurrentClusterName).AnyTimes()
	s.mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestCurrentClusterName).AnyTimes()

	mutableState.EXPECT().IsCurrentWorkflowGuaranteed().Return(false).AnyTimes()
	mutableState.EXPECT().IsWorkflowExecutionRunning().Return(false).AnyTimes()
	mutableState.EXPECT().GetDomainEntry().Return(s.domainEntry).AnyTimes()
	mutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      runID,
	}).AnyTimes()
	mutableState.EXPECT().GetNextEventID().Return(nextEventID).AnyTimes()
	mutableState.EXPECT().GetPreviousStartedEventID().Return(lastDecisionTaskStartedEventID).Times(1)
	mutableState.EXPECT().GetVersionHistories().Return(histories).Times(1)

	s.mockWorkflowResetter.EXPECT().ResetWorkflow(
		ctx,
		domainID,
		workflowID,
		runID,
		versionHistory.GetBranchToken(),
		lastDecisionTaskStartedEventID,
		lastDecisionTaskStartedVersion,
		nextEventID,
		gomock.Any(),
		gomock.Any(),
		workflow,
		EventsReapplicationResetWorkflowReason,
		workflowEvents.Events,
		false,
	).Return(nil).Times(1)

	s.mockExecutionManager.On("GetCurrentExecution", mock.Anything, &persistence.GetCurrentExecutionRequest{
		DomainID:   domainID,
		WorkflowID: workflowID,
	}).Return(&persistence.GetCurrentExecutionResponse{RunID: runID}, nil).Once()

	context.EXPECT().PersistNonFirstWorkflowEvents(gomock.Any(), workflowEvents).Return(int64(0), nil).Times(1)
	context.EXPECT().UpdateWorkflowExecutionWithNew(
		gomock.Any(), now, persistence.UpdateWorkflowModeBypassCurrent, nil, nil, execution.TransactionPolicyPassive, (*execution.TransactionPolicy)(nil),
	).Return(nil).Times(1)

	err := s.transactionManager.backfillWorkflow(ctx, now, workflow, workflowEvents)
	s.NoError(err)
	s.True(releaseCalled)
}

func (s *transactionManagerSuite) TestBackfillWorkflow_CurrentWorkflow_Passive_Open() {
	ctx := ctx.Background()
	now := time.Now()
	releaseCalled := false

	workflow := execution.NewMockWorkflow(s.controller)
	context := execution.NewMockContext(s.controller)
	mutableState := execution.NewMockMutableState(s.controller)
	var releaseFn execution.ReleaseFunc = func(error) { releaseCalled = true }

	workflowEvents := &persistence.WorkflowEvents{
		Events: []*shared.HistoryEvent{{EventId: common.Int64Ptr(1)}},
	}

	workflow.EXPECT().GetContext().Return(context).AnyTimes()
	workflow.EXPECT().GetMutableState().Return(mutableState).AnyTimes()
	workflow.EXPECT().GetReleaseFn().Return(releaseFn).AnyTimes()

	s.mockClusterMetadata.EXPECT().ClusterNameForFailoverVersion(s.domainEntry.GetFailoverVersion()).Return(cluster.TestCurrentClusterName).AnyTimes()
	s.mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestAlternativeClusterName).AnyTimes()

	mutableState.EXPECT().IsCurrentWorkflowGuaranteed().Return(true).AnyTimes()
	mutableState.EXPECT().IsWorkflowExecutionRunning().Return(true).AnyTimes()
	mutableState.EXPECT().GetDomainEntry().Return(s.domainEntry).AnyTimes()
	context.EXPECT().ReapplyEvents([]*persistence.WorkflowEvents{workflowEvents}).Times(1)
	context.EXPECT().PersistNonFirstWorkflowEvents(gomock.Any(), workflowEvents).Return(int64(0), nil).Times(1)
	context.EXPECT().UpdateWorkflowExecutionWithNew(
		gomock.Any(), now, persistence.UpdateWorkflowModeUpdateCurrent, nil, nil, execution.TransactionPolicyPassive, (*execution.TransactionPolicy)(nil),
	).Return(nil).Times(1)
	err := s.transactionManager.backfillWorkflow(ctx, now, workflow, workflowEvents)
	s.NoError(err)
	s.True(releaseCalled)
}

func (s *transactionManagerSuite) TestBackfillWorkflow_CurrentWorkflow_Passive_Closed() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	runID := "some random run ID"

	releaseCalled := false

	workflow := execution.NewMockWorkflow(s.controller)
	context := execution.NewMockContext(s.controller)
	mutableState := execution.NewMockMutableState(s.controller)
	var releaseFn execution.ReleaseFunc = func(error) { releaseCalled = true }

	workflowEvents := &persistence.WorkflowEvents{}

	workflow.EXPECT().GetContext().Return(context).AnyTimes()
	workflow.EXPECT().GetMutableState().Return(mutableState).AnyTimes()
	workflow.EXPECT().GetReleaseFn().Return(releaseFn).AnyTimes()

	s.mockClusterMetadata.EXPECT().ClusterNameForFailoverVersion(s.domainEntry.GetFailoverVersion()).Return(cluster.TestCurrentClusterName).AnyTimes()
	s.mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestAlternativeClusterName).AnyTimes()

	mutableState.EXPECT().IsCurrentWorkflowGuaranteed().Return(false).AnyTimes()
	mutableState.EXPECT().IsWorkflowExecutionRunning().Return(false).AnyTimes()
	mutableState.EXPECT().GetDomainEntry().Return(s.domainEntry).AnyTimes()
	mutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      runID,
	}).AnyTimes()

	s.mockExecutionManager.On("GetCurrentExecution", mock.Anything, &persistence.GetCurrentExecutionRequest{
		DomainID:   domainID,
		WorkflowID: workflowID,
	}).Return(&persistence.GetCurrentExecutionResponse{RunID: runID}, nil).Once()
	context.EXPECT().ReapplyEvents([]*persistence.WorkflowEvents{workflowEvents}).Times(1)
	context.EXPECT().PersistNonFirstWorkflowEvents(gomock.Any(), workflowEvents).Return(int64(0), nil).Times(1)
	context.EXPECT().UpdateWorkflowExecutionWithNew(
		gomock.Any(), now, persistence.UpdateWorkflowModeUpdateCurrent, nil, nil, execution.TransactionPolicyPassive, (*execution.TransactionPolicy)(nil),
	).Return(nil).Times(1)

	err := s.transactionManager.backfillWorkflow(ctx, now, workflow, workflowEvents)
	s.NoError(err)
	s.True(releaseCalled)
}

func (s *transactionManagerSuite) TestBackfillWorkflow_NotCurrentWorkflow_Active() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	runID := "some random run ID"
	currentRunID := "other random run ID"

	releaseCalled := false

	workflow := execution.NewMockWorkflow(s.controller)
	context := execution.NewMockContext(s.controller)
	mutableState := execution.NewMockMutableState(s.controller)
	var releaseFn execution.ReleaseFunc = func(error) { releaseCalled = true }

	workflowEvents := &persistence.WorkflowEvents{
		Events: []*shared.HistoryEvent{{
			EventType: common.EventTypePtr(shared.EventTypeWorkflowExecutionSignaled),
		}},
		DomainID:   domainID,
		WorkflowID: workflowID,
	}

	workflow.EXPECT().GetContext().Return(context).AnyTimes()
	workflow.EXPECT().GetMutableState().Return(mutableState).AnyTimes()
	workflow.EXPECT().GetReleaseFn().Return(releaseFn).AnyTimes()

	s.mockClusterMetadata.EXPECT().ClusterNameForFailoverVersion(s.domainEntry.GetFailoverVersion()).Return(cluster.TestCurrentClusterName).AnyTimes()
	s.mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestCurrentClusterName).AnyTimes()

	mutableState.EXPECT().IsCurrentWorkflowGuaranteed().Return(false).AnyTimes()
	mutableState.EXPECT().IsWorkflowExecutionRunning().Return(false).AnyTimes()
	mutableState.EXPECT().GetDomainEntry().Return(s.domainEntry).AnyTimes()
	mutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      runID,
	}).AnyTimes()

	s.mockExecutionManager.On("GetCurrentExecution", mock.Anything, &persistence.GetCurrentExecutionRequest{
		DomainID:   domainID,
		WorkflowID: workflowID,
	}).Return(&persistence.GetCurrentExecutionResponse{RunID: currentRunID}, nil).Once()
	context.EXPECT().ReapplyEvents([]*persistence.WorkflowEvents{workflowEvents}).Times(1)
	context.EXPECT().PersistNonFirstWorkflowEvents(gomock.Any(), workflowEvents).Return(int64(0), nil).Times(1)
	context.EXPECT().UpdateWorkflowExecutionWithNew(
		gomock.Any(), now, persistence.UpdateWorkflowModeBypassCurrent, nil, nil, execution.TransactionPolicyPassive, (*execution.TransactionPolicy)(nil),
	).Return(nil).Times(1)
	err := s.transactionManager.backfillWorkflow(ctx, now, workflow, workflowEvents)
	s.NoError(err)
	s.True(releaseCalled)
}

func (s *transactionManagerSuite) TestBackfillWorkflow_NotCurrentWorkflow_Passive() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	runID := "some random run ID"
	currentRunID := "other random run ID"

	releaseCalled := false

	workflow := execution.NewMockWorkflow(s.controller)
	context := execution.NewMockContext(s.controller)
	mutableState := execution.NewMockMutableState(s.controller)
	var releaseFn execution.ReleaseFunc = func(error) { releaseCalled = true }

	workflowEvents := &persistence.WorkflowEvents{
		Events: []*shared.HistoryEvent{{
			EventType: common.EventTypePtr(shared.EventTypeWorkflowExecutionSignaled),
		}},
		DomainID:   domainID,
		WorkflowID: workflowID,
	}

	workflow.EXPECT().GetContext().Return(context).AnyTimes()
	workflow.EXPECT().GetMutableState().Return(mutableState).AnyTimes()
	workflow.EXPECT().GetReleaseFn().Return(releaseFn).AnyTimes()

	s.mockClusterMetadata.EXPECT().ClusterNameForFailoverVersion(s.domainEntry.GetFailoverVersion()).Return(cluster.TestCurrentClusterName).AnyTimes()
	s.mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(cluster.TestAlternativeClusterName).AnyTimes()

	mutableState.EXPECT().IsCurrentWorkflowGuaranteed().Return(false).AnyTimes()
	mutableState.EXPECT().IsWorkflowExecutionRunning().Return(false).AnyTimes()
	mutableState.EXPECT().GetDomainEntry().Return(s.domainEntry).AnyTimes()
	mutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      runID,
	}).AnyTimes()

	s.mockExecutionManager.On("GetCurrentExecution", mock.Anything, &persistence.GetCurrentExecutionRequest{
		DomainID:   domainID,
		WorkflowID: workflowID,
	}).Return(&persistence.GetCurrentExecutionResponse{RunID: currentRunID}, nil).Once()
	context.EXPECT().ReapplyEvents([]*persistence.WorkflowEvents{workflowEvents}).Times(1)
	context.EXPECT().PersistNonFirstWorkflowEvents(gomock.Any(), workflowEvents).Return(int64(0), nil).Times(1)
	context.EXPECT().UpdateWorkflowExecutionWithNew(
		gomock.Any(), now, persistence.UpdateWorkflowModeBypassCurrent, nil, nil, execution.TransactionPolicyPassive, (*execution.TransactionPolicy)(nil),
	).Return(nil).Times(1)
	err := s.transactionManager.backfillWorkflow(ctx, now, workflow, workflowEvents)
	s.NoError(err)
	s.True(releaseCalled)
}

func (s *transactionManagerSuite) TestCheckWorkflowExists_DoesNotExists() {
	ctx := ctx.Background()
	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	runID := "some random run ID"

	s.mockExecutionManager.On("GetWorkflowExecution", mock.Anything, &persistence.GetWorkflowExecutionRequest{
		DomainID: domainID,
		Execution: shared.WorkflowExecution{
			WorkflowId: common.StringPtr(workflowID),
			RunId:      common.StringPtr(runID),
		},
	}).Return(nil, &shared.EntityNotExistsError{}).Once()

	exists, err := s.transactionManager.checkWorkflowExists(ctx, domainID, workflowID, runID)
	s.NoError(err)
	s.False(exists)
}

func (s *transactionManagerSuite) TestCheckWorkflowExists_DoesExists() {
	ctx := ctx.Background()
	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	runID := "some random run ID"

	s.mockExecutionManager.On("GetWorkflowExecution", mock.Anything, &persistence.GetWorkflowExecutionRequest{
		DomainID: domainID,
		Execution: shared.WorkflowExecution{
			WorkflowId: common.StringPtr(workflowID),
			RunId:      common.StringPtr(runID),
		},
	}).Return(&persistence.GetWorkflowExecutionResponse{}, nil).Once()

	exists, err := s.transactionManager.checkWorkflowExists(ctx, domainID, workflowID, runID)
	s.NoError(err)
	s.True(exists)
}

func (s *transactionManagerSuite) TestGetWorkflowCurrentRunID_Missing() {
	ctx := ctx.Background()
	domainID := "some random domain ID"
	workflowID := "some random workflow ID"

	s.mockExecutionManager.On("GetCurrentExecution", mock.Anything, &persistence.GetCurrentExecutionRequest{
		DomainID:   domainID,
		WorkflowID: workflowID,
	}).Return(nil, &shared.EntityNotExistsError{}).Once()

	currentRunID, err := s.transactionManager.getCurrentWorkflowRunID(ctx, domainID, workflowID)
	s.NoError(err)
	s.Equal("", currentRunID)
}

func (s *transactionManagerSuite) TestGetWorkflowCurrentRunID_Exists() {
	ctx := ctx.Background()
	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	runID := "some random run ID"

	s.mockExecutionManager.On("GetCurrentExecution", mock.Anything, &persistence.GetCurrentExecutionRequest{
		DomainID:   domainID,
		WorkflowID: workflowID,
	}).Return(&persistence.GetCurrentExecutionResponse{RunID: runID}, nil).Once()

	currentRunID, err := s.transactionManager.getCurrentWorkflowRunID(ctx, domainID, workflowID)
	s.NoError(err)
	s.Equal(runID, currentRunID)
}
