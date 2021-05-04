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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/service/history/execution"
)

type (
	transactionManagerForNewWorkflowSuite struct {
		suite.Suite
		*require.Assertions

		controller             *gomock.Controller
		mockTransactionManager *MocktransactionManager

		createManager *transactionManagerForNewWorkflowImpl
	}
)

func TestTransactionManagerForNewWorkflowSuite(t *testing.T) {
	s := new(transactionManagerForNewWorkflowSuite)
	suite.Run(t, s)
}

func (s *transactionManagerForNewWorkflowSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	s.controller = gomock.NewController(s.T())
	s.mockTransactionManager = NewMocktransactionManager(s.controller)

	s.createManager = newTransactionManagerForNewWorkflow(
		s.mockTransactionManager,
	).(*transactionManagerForNewWorkflowImpl)
}

func (s *transactionManagerForNewWorkflowSuite) TearDownTest() {
	s.controller.Finish()
}

func (s *transactionManagerForNewWorkflowSuite) TestDispatchForNewWorkflow_Dup() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	runID := "some random run ID"

	workflow := execution.NewMockWorkflow(s.controller)
	mutableState := execution.NewMockMutableState(s.controller)
	workflow.EXPECT().GetMutableState().Return(mutableState).AnyTimes()

	mutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      runID,
	}).AnyTimes()

	s.mockTransactionManager.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(runID, nil).Times(1)

	err := s.createManager.dispatchForNewWorkflow(ctx, now, workflow)
	s.NoError(err)
}

func (s *transactionManagerForNewWorkflowSuite) TestDispatchForNewWorkflow_BrandNew() {
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
	workflow.EXPECT().GetContext().Return(context).AnyTimes()
	workflow.EXPECT().GetMutableState().Return(mutableState).AnyTimes()
	workflow.EXPECT().GetReleaseFn().Return(releaseFn).AnyTimes()

	workflowSnapshot := &persistence.WorkflowSnapshot{}
	workflowEventsSeq := []*persistence.WorkflowEvents{&persistence.WorkflowEvents{}}
	workflowHistorySize := int64(12345)
	mutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      runID,
	}).AnyTimes()
	mutableState.EXPECT().CloseTransactionAsSnapshot(now, execution.TransactionPolicyPassive).Return(
		workflowSnapshot, workflowEventsSeq, nil,
	).Times(1)

	s.mockTransactionManager.EXPECT().getCurrentWorkflowRunID(
		ctx, domainID, workflowID,
	).Return("", nil).Times(1)

	context.EXPECT().PersistFirstWorkflowEvents(
		gomock.Any(),
		workflowEventsSeq[0],
	).Return(workflowHistorySize, nil).Times(1)
	context.EXPECT().CreateWorkflowExecution(
		gomock.Any(),
		workflowSnapshot,
		workflowHistorySize,
		now,
		persistence.CreateWorkflowModeBrandNew,
		"",
		int64(0),
	).Return(nil).Times(1)

	err := s.createManager.dispatchForNewWorkflow(ctx, now, workflow)
	s.NoError(err)
	s.True(releaseCalled)
}

func (s *transactionManagerForNewWorkflowSuite) TestDispatchForNewWorkflow_CreateAsCurrent() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"
	currentRunID := "other random runID"
	currentLastWriteVersion := int64(4321)

	targetReleaseCalled := false
	currentReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	currentWorkflow := execution.NewMockWorkflow(s.controller)
	currentMutableState := execution.NewMockMutableState(s.controller)
	var currentReleaseFn execution.ReleaseFunc = func(error) { currentReleaseCalled = true }
	currentWorkflow.EXPECT().GetMutableState().Return(currentMutableState).AnyTimes()
	currentWorkflow.EXPECT().GetReleaseFn().Return(currentReleaseFn).AnyTimes()

	targetWorkflowSnapshot := &persistence.WorkflowSnapshot{}
	targetWorkflowEventsSeq := []*persistence.WorkflowEvents{&persistence.WorkflowEvents{}}
	targetWorkflowHistorySize := int64(12345)
	targetMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      targetRunID,
	}).AnyTimes()
	targetMutableState.EXPECT().CloseTransactionAsSnapshot(now, execution.TransactionPolicyPassive).Return(
		targetWorkflowSnapshot, targetWorkflowEventsSeq, nil,
	).Times(1)

	s.mockTransactionManager.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(currentRunID, nil).Times(1)
	s.mockTransactionManager.EXPECT().loadNDCWorkflow(ctx, domainID, workflowID, currentRunID).Return(currentWorkflow, nil).Times(1)

	targetWorkflow.EXPECT().HappensAfter(currentWorkflow).Return(true, nil)
	currentMutableState.EXPECT().IsWorkflowExecutionRunning().Return(false).AnyTimes()
	currentMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      currentRunID,
	}).AnyTimes()
	currentWorkflow.EXPECT().GetVectorClock().Return(currentLastWriteVersion, int64(0), nil)

	targetContext.EXPECT().PersistFirstWorkflowEvents(
		gomock.Any(),
		targetWorkflowEventsSeq[0],
	).Return(targetWorkflowHistorySize, nil).Times(1)
	targetContext.EXPECT().CreateWorkflowExecution(
		gomock.Any(),
		targetWorkflowSnapshot,
		targetWorkflowHistorySize,
		now,
		persistence.CreateWorkflowModeWorkflowIDReuse,
		currentRunID,
		currentLastWriteVersion,
	).Return(nil).Times(1)

	err := s.createManager.dispatchForNewWorkflow(ctx, now, targetWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(currentReleaseCalled)
}

func (s *transactionManagerForNewWorkflowSuite) TestDispatchForNewWorkflow_CreateAsZombie() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"
	currentRunID := "other random runID"

	targetReleaseCalled := false
	currentReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	currentWorkflow := execution.NewMockWorkflow(s.controller)
	var currentReleaseFn execution.ReleaseFunc = func(error) { currentReleaseCalled = true }
	currentWorkflow.EXPECT().GetReleaseFn().Return(currentReleaseFn).AnyTimes()

	targetWorkflowSnapshot := &persistence.WorkflowSnapshot{
		ExecutionInfo: &persistence.WorkflowExecutionInfo{
			DomainID:   domainID,
			WorkflowID: workflowID,
		},
	}
	targetWorkflowEventsSeq := []*persistence.WorkflowEvents{&persistence.WorkflowEvents{}}
	targetWorkflowHistorySize := int64(12345)
	targetMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      targetRunID,
	}).AnyTimes()
	targetMutableState.EXPECT().CloseTransactionAsSnapshot(now, execution.TransactionPolicyPassive).Return(
		targetWorkflowSnapshot, targetWorkflowEventsSeq, nil,
	).Times(1)

	s.mockTransactionManager.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(currentRunID, nil).Times(1)
	s.mockTransactionManager.EXPECT().loadNDCWorkflow(ctx, domainID, workflowID, currentRunID).Return(currentWorkflow, nil).Times(1)

	targetWorkflow.EXPECT().HappensAfter(currentWorkflow).Return(false, nil)
	targetWorkflow.EXPECT().SuppressBy(currentWorkflow).Return(execution.TransactionPolicyPassive, nil).Times(1)

	targetContext.EXPECT().PersistFirstWorkflowEvents(
		gomock.Any(),
		targetWorkflowEventsSeq[0],
	).Return(targetWorkflowHistorySize, nil).Times(1)
	targetContext.EXPECT().CreateWorkflowExecution(
		gomock.Any(),
		targetWorkflowSnapshot,
		targetWorkflowHistorySize,
		now,
		persistence.CreateWorkflowModeZombie,
		"",
		int64(0),
	).Return(nil).Times(1)
	targetContext.EXPECT().ReapplyEvents(targetWorkflowEventsSeq).Return(nil).Times(1)

	err := s.createManager.dispatchForNewWorkflow(ctx, now, targetWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(currentReleaseCalled)
}

func (s *transactionManagerForNewWorkflowSuite) TestDispatchForNewWorkflow_CreateAsZombie_Dedup() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"
	currentRunID := "other random runID"

	targetReleaseCalled := false
	currentReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	currentWorkflow := execution.NewMockWorkflow(s.controller)
	var currentReleaseFn execution.ReleaseFunc = func(error) { currentReleaseCalled = true }
	currentWorkflow.EXPECT().GetReleaseFn().Return(currentReleaseFn).AnyTimes()

	targetWorkflowSnapshot := &persistence.WorkflowSnapshot{
		ExecutionInfo: &persistence.WorkflowExecutionInfo{
			DomainID:   domainID,
			WorkflowID: workflowID,
		},
	}
	targetWorkflowEventsSeq := []*persistence.WorkflowEvents{&persistence.WorkflowEvents{}}
	targetWorkflowHistorySize := int64(12345)
	targetMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      targetRunID,
	}).AnyTimes()
	targetMutableState.EXPECT().CloseTransactionAsSnapshot(now, execution.TransactionPolicyPassive).Return(
		targetWorkflowSnapshot, targetWorkflowEventsSeq, nil,
	).Times(1)

	s.mockTransactionManager.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(currentRunID, nil).Times(1)
	s.mockTransactionManager.EXPECT().loadNDCWorkflow(ctx, domainID, workflowID, currentRunID).Return(currentWorkflow, nil).Times(1)

	targetWorkflow.EXPECT().HappensAfter(currentWorkflow).Return(false, nil)
	targetWorkflow.EXPECT().SuppressBy(currentWorkflow).Return(execution.TransactionPolicyPassive, nil).Times(1)

	targetContext.EXPECT().PersistFirstWorkflowEvents(
		gomock.Any(),
		targetWorkflowEventsSeq[0],
	).Return(targetWorkflowHistorySize, nil).Times(1)
	targetContext.EXPECT().CreateWorkflowExecution(
		gomock.Any(),
		targetWorkflowSnapshot,
		targetWorkflowHistorySize,
		now,
		persistence.CreateWorkflowModeZombie,
		"",
		int64(0),
	).Return(&persistence.WorkflowExecutionAlreadyStartedError{}).Times(1)
	targetContext.EXPECT().ReapplyEvents(targetWorkflowEventsSeq).Return(nil).Times(1)

	err := s.createManager.dispatchForNewWorkflow(ctx, now, targetWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(currentReleaseCalled)
}

func (s *transactionManagerForNewWorkflowSuite) TestDispatchForNewWorkflow_SuppressCurrentAndCreateAsCurrent() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"
	currentRunID := "other random runID"

	targetReleaseCalled := false
	currentReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	currentWorkflow := execution.NewMockWorkflow(s.controller)
	currentContext := execution.NewMockContext(s.controller)
	currentMutableState := execution.NewMockMutableState(s.controller)
	var currentReleaseFn execution.ReleaseFunc = func(error) { currentReleaseCalled = true }
	currentWorkflow.EXPECT().GetContext().Return(currentContext).AnyTimes()
	currentWorkflow.EXPECT().GetMutableState().Return(currentMutableState).AnyTimes()
	currentWorkflow.EXPECT().GetReleaseFn().Return(currentReleaseFn).AnyTimes()

	targetMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      targetRunID,
	}).AnyTimes()

	s.mockTransactionManager.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(currentRunID, nil).Times(1)
	s.mockTransactionManager.EXPECT().loadNDCWorkflow(ctx, domainID, workflowID, currentRunID).Return(currentWorkflow, nil).Times(1)

	targetWorkflow.EXPECT().HappensAfter(currentWorkflow).Return(true, nil)
	currentMutableState.EXPECT().IsWorkflowExecutionRunning().Return(true).AnyTimes()
	currentWorkflowPolicy := execution.TransactionPolicyActive
	currentWorkflow.EXPECT().SuppressBy(targetWorkflow).Return(currentWorkflowPolicy, nil).Times(1)
	targetWorkflow.EXPECT().Revive().Return(nil).Times(1)

	currentContext.EXPECT().UpdateWorkflowExecutionWithNew(
		gomock.Any(),
		now,
		persistence.UpdateWorkflowModeUpdateCurrent,
		targetContext,
		targetMutableState,
		currentWorkflowPolicy,
		execution.TransactionPolicyPassive.Ptr(),
	).Return(nil).Times(1)

	err := s.createManager.dispatchForNewWorkflow(ctx, now, targetWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(currentReleaseCalled)
}
