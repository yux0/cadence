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
	transactionManagerForExistingWorkflowSuite struct {
		suite.Suite
		*require.Assertions

		controller         *gomock.Controller
		mockTransactionMgr *MocktransactionManager

		updateMgr *transactionManagerForExistingWorkflowImpl
	}
)

func TestTransactionMgrForExistingWorkflowSuite(t *testing.T) {
	s := new(transactionManagerForExistingWorkflowSuite)
	suite.Run(t, s)
}

func (s *transactionManagerForExistingWorkflowSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	s.controller = gomock.NewController(s.T())
	s.mockTransactionMgr = NewMocktransactionManager(s.controller)

	s.updateMgr = newTransactionManagerForExistingWorkflow(
		s.mockTransactionMgr,
	).(*transactionManagerForExistingWorkflowImpl)
}

func (s *transactionManagerForExistingWorkflowSuite) TearDownTest() {
	s.controller.Finish()
}

func (s *transactionManagerForExistingWorkflowSuite) TestDispatchForExistingWorkflow_NoRebuild_CurrentWorkflowGuaranteed() {
	ctx := ctx.Background()
	now := time.Now()

	isWorkflowRebuilt := false

	targetReleaseCalled := false
	newReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	newWorkflow := execution.NewMockWorkflow(s.controller)
	newContext := execution.NewMockContext(s.controller)
	newMutableState := execution.NewMockMutableState(s.controller)
	var newReleaseFn execution.ReleaseFunc = func(error) { newReleaseCalled = true }
	newWorkflow.EXPECT().GetContext().Return(newContext).AnyTimes()
	newWorkflow.EXPECT().GetMutableState().Return(newMutableState).AnyTimes()
	newWorkflow.EXPECT().GetReleaseFn().Return(newReleaseFn).AnyTimes()

	targetMutableState.EXPECT().IsCurrentWorkflowGuaranteed().Return(true).AnyTimes()

	targetContext.EXPECT().UpdateWorkflowExecutionWithNewAsPassive(
		now,
		newContext,
		newMutableState,
	).Return(nil).Times(1)

	err := s.updateMgr.dispatchForExistingWorkflow(ctx, now, isWorkflowRebuilt, targetWorkflow, newWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(newReleaseCalled)
}

func (s *transactionManagerForExistingWorkflowSuite) TestDispatchForExistingWorkflow_NoRebuild_CurrentWorkflowNotGuaranteed_IsCurrent() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"

	isWorkflowRebuilt := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()

	newWorkflow := execution.NewMockWorkflow(s.controller)

	targetMutableState.EXPECT().IsCurrentWorkflowGuaranteed().Return(false).AnyTimes()
	targetMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      targetRunID,
	}).AnyTimes()
	s.mockTransactionMgr.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(targetRunID, nil).Times(1)

	err := s.updateMgr.dispatchForExistingWorkflow(ctx, now, isWorkflowRebuilt, targetWorkflow, newWorkflow)
	s.Error(err)
}

func (s *transactionManagerForExistingWorkflowSuite) TestDispatchForExistingWorkflow_NoRebuild_CurrentWorkflowNotGuaranteed_NotCurrent_CurrentRunning_UpdateAsCurrent() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"
	currentRunID := "other random runID"

	isWorkflowRebuilt := false

	targetReleaseCalled := false
	newReleaseCalled := false
	currentReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	newWorkflow := execution.NewMockWorkflow(s.controller)
	newContext := execution.NewMockContext(s.controller)
	newMutableState := execution.NewMockMutableState(s.controller)
	var newReleaseFn execution.ReleaseFunc = func(error) { newReleaseCalled = true }
	newWorkflow.EXPECT().GetContext().Return(newContext).AnyTimes()
	newWorkflow.EXPECT().GetMutableState().Return(newMutableState).AnyTimes()
	newWorkflow.EXPECT().GetReleaseFn().Return(newReleaseFn).AnyTimes()
	newWorkflow.EXPECT().Revive().Return(nil).Times(1)

	currentWorkflow := execution.NewMockWorkflow(s.controller)
	currentContext := execution.NewMockContext(s.controller)
	currentMutableState := execution.NewMockMutableState(s.controller)
	var currentReleaseFn execution.ReleaseFunc = func(error) { currentReleaseCalled = true }
	currentWorkflow.EXPECT().GetContext().Return(currentContext).AnyTimes()
	currentWorkflow.EXPECT().GetMutableState().Return(currentMutableState).AnyTimes()
	currentWorkflow.EXPECT().GetReleaseFn().Return(currentReleaseFn).AnyTimes()

	targetMutableState.EXPECT().IsCurrentWorkflowGuaranteed().Return(false).AnyTimes()
	targetMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      targetRunID,
	}).AnyTimes()
	s.mockTransactionMgr.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(currentRunID, nil).Times(1)
	s.mockTransactionMgr.EXPECT().loadNDCWorkflow(ctx, domainID, workflowID, currentRunID).Return(currentWorkflow, nil).Times(1)

	targetWorkflow.EXPECT().HappensAfter(currentWorkflow).Return(true, nil)
	currentWorkflowPolicy := execution.TransactionPolicyPassive
	currentMutableState.EXPECT().IsWorkflowExecutionRunning().Return(true).AnyTimes()
	currentWorkflow.EXPECT().SuppressBy(targetWorkflow).Return(currentWorkflowPolicy, nil).Times(1)
	targetWorkflow.EXPECT().Revive().Return(nil).Times(1)

	targetContext.EXPECT().ConflictResolveWorkflowExecution(
		now,
		persistence.ConflictResolveWorkflowModeUpdateCurrent,
		targetMutableState,
		newContext,
		newMutableState,
		currentContext,
		currentMutableState,
		currentWorkflowPolicy.Ptr(),
	).Return(nil).Times(1)

	err := s.updateMgr.dispatchForExistingWorkflow(ctx, now, isWorkflowRebuilt, targetWorkflow, newWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(newReleaseCalled)
	s.True(currentReleaseCalled)
}

func (s *transactionManagerForExistingWorkflowSuite) TestDispatchForExistingWorkflow_NoRebuild_CurrentWorkflowNotGuaranteed_NotCurrent_CurrentComplete_UpdateAsCurrent() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"
	currentRunID := "other random runID"

	isWorkflowRebuilt := false

	targetReleaseCalled := false
	newReleaseCalled := false
	currentReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	newWorkflow := execution.NewMockWorkflow(s.controller)
	newContext := execution.NewMockContext(s.controller)
	newMutableState := execution.NewMockMutableState(s.controller)
	var newReleaseFn execution.ReleaseFunc = func(error) { newReleaseCalled = true }
	newWorkflow.EXPECT().GetContext().Return(newContext).AnyTimes()
	newWorkflow.EXPECT().GetMutableState().Return(newMutableState).AnyTimes()
	newWorkflow.EXPECT().GetReleaseFn().Return(newReleaseFn).AnyTimes()
	newWorkflow.EXPECT().Revive().Return(nil).Times(1)

	currentWorkflow := execution.NewMockWorkflow(s.controller)
	currentContext := execution.NewMockContext(s.controller)
	currentMutableState := execution.NewMockMutableState(s.controller)
	var currentReleaseFn execution.ReleaseFunc = func(error) { currentReleaseCalled = true }
	currentWorkflow.EXPECT().GetContext().Return(currentContext).AnyTimes()
	currentWorkflow.EXPECT().GetMutableState().Return(currentMutableState).AnyTimes()
	currentWorkflow.EXPECT().GetReleaseFn().Return(currentReleaseFn).AnyTimes()

	targetMutableState.EXPECT().IsCurrentWorkflowGuaranteed().Return(false).AnyTimes()
	targetMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      targetRunID,
	}).AnyTimes()
	s.mockTransactionMgr.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(currentRunID, nil).Times(1)
	s.mockTransactionMgr.EXPECT().loadNDCWorkflow(ctx, domainID, workflowID, currentRunID).Return(currentWorkflow, nil).Times(1)

	targetWorkflow.EXPECT().HappensAfter(currentWorkflow).Return(true, nil)
	currentWorkflowPolicy := execution.TransactionPolicyPassive
	currentMutableState.EXPECT().IsWorkflowExecutionRunning().Return(false).AnyTimes()
	currentWorkflow.EXPECT().SuppressBy(targetWorkflow).Return(currentWorkflowPolicy, nil).Times(0)
	targetWorkflow.EXPECT().Revive().Return(nil).Times(1)

	targetContext.EXPECT().ConflictResolveWorkflowExecution(
		now,
		persistence.ConflictResolveWorkflowModeUpdateCurrent,
		targetMutableState,
		newContext,
		newMutableState,
		currentContext,
		currentMutableState,
		currentWorkflowPolicy.Ptr(),
	).Return(nil).Times(1)

	err := s.updateMgr.dispatchForExistingWorkflow(ctx, now, isWorkflowRebuilt, targetWorkflow, newWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(newReleaseCalled)
	s.True(currentReleaseCalled)
}

func (s *transactionManagerForExistingWorkflowSuite) TestDispatchForExistingWorkflow_NoRebuild_CurrentWorkflowNotGuaranteed_NotCurrent_UpdateAsZombie_NewRunDoesNotExists() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"
	newRunID := "some random new run ID"
	currentRunID := "other random runID"

	isWorkflowRebuilt := false

	targetReleaseCalled := false
	newReleaseCalled := false
	currentReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	newWorkflow := execution.NewMockWorkflow(s.controller)
	newContext := execution.NewMockContext(s.controller)
	newMutableState := execution.NewMockMutableState(s.controller)
	var newReleaseFn execution.ReleaseFunc = func(error) { newReleaseCalled = true }
	newWorkflow.EXPECT().GetContext().Return(newContext).AnyTimes()
	newWorkflow.EXPECT().GetMutableState().Return(newMutableState).AnyTimes()
	newWorkflow.EXPECT().GetReleaseFn().Return(newReleaseFn).AnyTimes()

	currentWorkflow := execution.NewMockWorkflow(s.controller)
	var currentReleaseFn execution.ReleaseFunc = func(error) { currentReleaseCalled = true }
	currentWorkflow.EXPECT().GetReleaseFn().Return(currentReleaseFn).AnyTimes()

	targetMutableState.EXPECT().IsCurrentWorkflowGuaranteed().Return(false).AnyTimes()
	targetMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      targetRunID,
	}).AnyTimes()
	newMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      newRunID,
	}).AnyTimes()
	s.mockTransactionMgr.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(currentRunID, nil).Times(1)
	s.mockTransactionMgr.EXPECT().loadNDCWorkflow(ctx, domainID, workflowID, currentRunID).Return(currentWorkflow, nil).Times(1)
	s.mockTransactionMgr.EXPECT().checkWorkflowExists(ctx, domainID, workflowID, newRunID).Return(false, nil).Times(1)

	targetWorkflow.EXPECT().HappensAfter(currentWorkflow).Return(false, nil)
	targetWorkflow.EXPECT().SuppressBy(currentWorkflow).Return(execution.TransactionPolicyPassive, nil).Times(1)
	newWorkflow.EXPECT().SuppressBy(currentWorkflow).Return(execution.TransactionPolicyPassive, nil).Times(1)

	targetContext.EXPECT().UpdateWorkflowExecutionWithNew(
		now,
		persistence.UpdateWorkflowModeBypassCurrent,
		newContext,
		newMutableState,
		execution.TransactionPolicyPassive,
		execution.TransactionPolicyPassive.Ptr(),
	).Return(nil).Times(1)

	err := s.updateMgr.dispatchForExistingWorkflow(ctx, now, isWorkflowRebuilt, targetWorkflow, newWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(newReleaseCalled)
	s.True(currentReleaseCalled)
}

func (s *transactionManagerForExistingWorkflowSuite) TestDispatchForExistingWorkflow_NoRebuild_CurrentWorkflowNotGuaranteed_NotCurrent_UpdateAsZombie_NewRunDoesExists() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"
	newRunID := "some random new run ID"
	currentRunID := "other random runID"

	isWorkflowRebuilt := false

	targetReleaseCalled := false
	newReleaseCalled := false
	currentReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	newWorkflow := execution.NewMockWorkflow(s.controller)
	newContext := execution.NewMockContext(s.controller)
	newMutableState := execution.NewMockMutableState(s.controller)
	var newReleaseFn execution.ReleaseFunc = func(error) { newReleaseCalled = true }
	newWorkflow.EXPECT().GetContext().Return(newContext).AnyTimes()
	newWorkflow.EXPECT().GetMutableState().Return(newMutableState).AnyTimes()
	newWorkflow.EXPECT().GetReleaseFn().Return(newReleaseFn).AnyTimes()

	currentWorkflow := execution.NewMockWorkflow(s.controller)
	var currentReleaseFn execution.ReleaseFunc = func(error) { currentReleaseCalled = true }
	currentWorkflow.EXPECT().GetReleaseFn().Return(currentReleaseFn).AnyTimes()

	targetMutableState.EXPECT().IsCurrentWorkflowGuaranteed().Return(false).AnyTimes()
	targetMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      targetRunID,
	}).AnyTimes()
	newMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      newRunID,
	}).AnyTimes()
	s.mockTransactionMgr.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(currentRunID, nil).Times(1)
	s.mockTransactionMgr.EXPECT().loadNDCWorkflow(ctx, domainID, workflowID, currentRunID).Return(currentWorkflow, nil).Times(1)
	s.mockTransactionMgr.EXPECT().checkWorkflowExists(ctx, domainID, workflowID, newRunID).Return(true, nil).Times(1)

	targetWorkflow.EXPECT().HappensAfter(currentWorkflow).Return(false, nil)
	targetWorkflow.EXPECT().SuppressBy(currentWorkflow).Return(execution.TransactionPolicyPassive, nil).Times(1)
	newWorkflow.EXPECT().SuppressBy(currentWorkflow).Return(execution.TransactionPolicyPassive, nil).Times(1)

	targetContext.EXPECT().UpdateWorkflowExecutionWithNew(
		now,
		persistence.UpdateWorkflowModeBypassCurrent,
		(execution.Context)(nil),
		(execution.MutableState)(nil),
		execution.TransactionPolicyPassive,
		(*execution.TransactionPolicy)(nil),
	).Return(nil).Times(1)

	err := s.updateMgr.dispatchForExistingWorkflow(ctx, now, isWorkflowRebuilt, targetWorkflow, newWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(newReleaseCalled)
	s.True(currentReleaseCalled)
}

func (s *transactionManagerForExistingWorkflowSuite) TestDispatchForExistingWorkflow_Rebuild_IsCurrent() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"

	isWorkflowRebuilt := true

	targetReleaseCalled := false
	newReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	newWorkflow := execution.NewMockWorkflow(s.controller)
	newContext := execution.NewMockContext(s.controller)
	newMutableState := execution.NewMockMutableState(s.controller)
	var newReleaseFn execution.ReleaseFunc = func(error) { newReleaseCalled = true }
	newWorkflow.EXPECT().GetContext().Return(newContext).AnyTimes()
	newWorkflow.EXPECT().GetMutableState().Return(newMutableState).AnyTimes()
	newWorkflow.EXPECT().GetReleaseFn().Return(newReleaseFn).AnyTimes()

	targetMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      targetRunID,
	}).AnyTimes()
	s.mockTransactionMgr.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(targetRunID, nil).Times(1)

	targetContext.EXPECT().ConflictResolveWorkflowExecution(
		now,
		persistence.ConflictResolveWorkflowModeUpdateCurrent,
		targetMutableState,
		newContext,
		newMutableState,
		(execution.Context)(nil),
		(execution.MutableState)(nil),
		(*execution.TransactionPolicy)(nil),
	).Return(nil).Times(1)

	err := s.updateMgr.dispatchForExistingWorkflow(ctx, now, isWorkflowRebuilt, targetWorkflow, newWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(newReleaseCalled)
}

func (s *transactionManagerForExistingWorkflowSuite) TestDispatchForExistingWorkflow_Rebuild_NotCurrent_UpdateAsCurrent() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"
	currentRunID := "other random runID"

	isWorkflowRebuilt := true

	targetReleaseCalled := false
	newReleaseCalled := false
	currentReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	newWorkflow := execution.NewMockWorkflow(s.controller)
	newContext := execution.NewMockContext(s.controller)
	newMutableState := execution.NewMockMutableState(s.controller)
	var newReleaseFn execution.ReleaseFunc = func(error) { newReleaseCalled = true }
	newWorkflow.EXPECT().GetContext().Return(newContext).AnyTimes()
	newWorkflow.EXPECT().GetMutableState().Return(newMutableState).AnyTimes()
	newWorkflow.EXPECT().GetReleaseFn().Return(newReleaseFn).AnyTimes()
	newWorkflow.EXPECT().Revive().Return(nil).Times(1)

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
	s.mockTransactionMgr.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(currentRunID, nil).Times(1)
	s.mockTransactionMgr.EXPECT().loadNDCWorkflow(ctx, domainID, workflowID, currentRunID).Return(currentWorkflow, nil).Times(1)

	targetWorkflow.EXPECT().HappensAfter(currentWorkflow).Return(true, nil)
	currentWorkflowPolicy := execution.TransactionPolicyActive
	currentMutableState.EXPECT().IsWorkflowExecutionRunning().Return(true).AnyTimes()
	currentWorkflow.EXPECT().SuppressBy(targetWorkflow).Return(currentWorkflowPolicy, nil).Times(1)
	targetWorkflow.EXPECT().Revive().Return(nil).Times(1)

	targetContext.EXPECT().ConflictResolveWorkflowExecution(
		now,
		persistence.ConflictResolveWorkflowModeUpdateCurrent,
		targetMutableState,
		newContext,
		newMutableState,
		currentContext,
		currentMutableState,
		currentWorkflowPolicy.Ptr(),
	).Return(nil).Times(1)

	err := s.updateMgr.dispatchForExistingWorkflow(ctx, now, isWorkflowRebuilt, targetWorkflow, newWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(newReleaseCalled)
	s.True(currentReleaseCalled)
}

func (s *transactionManagerForExistingWorkflowSuite) TestDispatchForExistingWorkflow_Rebuild_NotCurrent_UpdateAsZombie_NewRunDoesNotExists() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"
	newRunID := "some random new run ID"
	currentRunID := "other random runID"

	isWorkflowRebuilt := true

	targetReleaseCalled := false
	newReleaseCalled := false
	currentReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	newWorkflow := execution.NewMockWorkflow(s.controller)
	newContext := execution.NewMockContext(s.controller)
	newMutableState := execution.NewMockMutableState(s.controller)
	var newReleaseFn execution.ReleaseFunc = func(error) { newReleaseCalled = true }
	newWorkflow.EXPECT().GetContext().Return(newContext).AnyTimes()
	newWorkflow.EXPECT().GetMutableState().Return(newMutableState).AnyTimes()
	newWorkflow.EXPECT().GetReleaseFn().Return(newReleaseFn).AnyTimes()

	currentWorkflow := execution.NewMockWorkflow(s.controller)
	var currentReleaseFn execution.ReleaseFunc = func(error) { currentReleaseCalled = true }
	currentWorkflow.EXPECT().GetReleaseFn().Return(currentReleaseFn).AnyTimes()

	targetMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      targetRunID,
	}).AnyTimes()
	newMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      newRunID,
	}).AnyTimes()
	s.mockTransactionMgr.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(currentRunID, nil).Times(1)
	s.mockTransactionMgr.EXPECT().loadNDCWorkflow(ctx, domainID, workflowID, currentRunID).Return(currentWorkflow, nil).Times(1)
	s.mockTransactionMgr.EXPECT().checkWorkflowExists(ctx, domainID, workflowID, newRunID).Return(false, nil).Times(1)

	targetWorkflow.EXPECT().HappensAfter(currentWorkflow).Return(false, nil)
	targetWorkflow.EXPECT().SuppressBy(currentWorkflow).Return(execution.TransactionPolicyPassive, nil).Times(1)
	newWorkflow.EXPECT().SuppressBy(currentWorkflow).Return(execution.TransactionPolicyPassive, nil).Times(1)

	targetContext.EXPECT().ConflictResolveWorkflowExecution(
		now,
		persistence.ConflictResolveWorkflowModeBypassCurrent,
		targetMutableState,
		newContext,
		newMutableState,
		(execution.Context)(nil),
		(execution.MutableState)(nil),
		(*execution.TransactionPolicy)(nil),
	).Return(nil).Times(1)

	err := s.updateMgr.dispatchForExistingWorkflow(ctx, now, isWorkflowRebuilt, targetWorkflow, newWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(newReleaseCalled)
	s.True(currentReleaseCalled)
}

func (s *transactionManagerForExistingWorkflowSuite) TestDispatchForExistingWorkflow_Rebuild_NotCurrent_UpdateAsZombie_NewRunDoesExists() {
	ctx := ctx.Background()
	now := time.Now()

	domainID := "some random domain ID"
	workflowID := "some random workflow ID"
	targetRunID := "some random run ID"
	newRunID := "some random new run ID"
	currentRunID := "other random runID"

	isWorkflowRebuilt := true

	targetReleaseCalled := false
	newReleaseCalled := false
	currentReleaseCalled := false

	targetWorkflow := execution.NewMockWorkflow(s.controller)
	targetContext := execution.NewMockContext(s.controller)
	targetMutableState := execution.NewMockMutableState(s.controller)
	var targetReleaseFn execution.ReleaseFunc = func(error) { targetReleaseCalled = true }
	targetWorkflow.EXPECT().GetContext().Return(targetContext).AnyTimes()
	targetWorkflow.EXPECT().GetMutableState().Return(targetMutableState).AnyTimes()
	targetWorkflow.EXPECT().GetReleaseFn().Return(targetReleaseFn).AnyTimes()

	newWorkflow := execution.NewMockWorkflow(s.controller)
	newContext := execution.NewMockContext(s.controller)
	newMutableState := execution.NewMockMutableState(s.controller)
	var newReleaseFn execution.ReleaseFunc = func(error) { newReleaseCalled = true }
	newWorkflow.EXPECT().GetContext().Return(newContext).AnyTimes()
	newWorkflow.EXPECT().GetMutableState().Return(newMutableState).AnyTimes()
	newWorkflow.EXPECT().GetReleaseFn().Return(newReleaseFn).AnyTimes()

	currentWorkflow := execution.NewMockWorkflow(s.controller)
	var currentReleaseFn execution.ReleaseFunc = func(error) { currentReleaseCalled = true }
	currentWorkflow.EXPECT().GetReleaseFn().Return(currentReleaseFn).AnyTimes()

	targetMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      targetRunID,
	}).AnyTimes()
	newMutableState.EXPECT().GetExecutionInfo().Return(&persistence.WorkflowExecutionInfo{
		DomainID:   domainID,
		WorkflowID: workflowID,
		RunID:      newRunID,
	}).AnyTimes()
	s.mockTransactionMgr.EXPECT().getCurrentWorkflowRunID(ctx, domainID, workflowID).Return(currentRunID, nil).Times(1)
	s.mockTransactionMgr.EXPECT().loadNDCWorkflow(ctx, domainID, workflowID, currentRunID).Return(currentWorkflow, nil).Times(1)
	s.mockTransactionMgr.EXPECT().checkWorkflowExists(ctx, domainID, workflowID, newRunID).Return(true, nil).Times(1)

	targetWorkflow.EXPECT().HappensAfter(currentWorkflow).Return(false, nil)
	targetWorkflow.EXPECT().SuppressBy(currentWorkflow).Return(execution.TransactionPolicyPassive, nil).Times(1)
	newWorkflow.EXPECT().SuppressBy(currentWorkflow).Return(execution.TransactionPolicyPassive, nil).Times(1)

	targetContext.EXPECT().ConflictResolveWorkflowExecution(
		now,
		persistence.ConflictResolveWorkflowModeBypassCurrent,
		targetMutableState,
		(execution.Context)(nil),
		(execution.MutableState)(nil),
		(execution.Context)(nil),
		(execution.MutableState)(nil),
		(*execution.TransactionPolicy)(nil),
	).Return(nil).Times(1)

	err := s.updateMgr.dispatchForExistingWorkflow(ctx, now, isWorkflowRebuilt, targetWorkflow, newWorkflow)
	s.NoError(err)
	s.True(targetReleaseCalled)
	s.True(newReleaseCalled)
	s.True(currentReleaseCalled)
}
