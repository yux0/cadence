// Copyright (c) 2020 Uber Technologies Inc.
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
	"testing"

	"github.com/uber/cadence/.gen/go/health"
	hist "github.com/uber/cadence/.gen/go/history"
	"github.com/uber/cadence/.gen/go/replicator"
	"github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common/types"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestThriftHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := NewMockHandler(ctrl)
	th := NewThriftHandler(h)
	ctx := context.Background()
	internalErr := &types.InternalServiceError{Message: "test"}
	expectedErr := &shared.InternalServiceError{Message: "test"}

	t.Run("Health", func(t *testing.T) {
		h.EXPECT().Health(ctx).Return(&health.HealthStatus{}, internalErr).Times(1)
		resp, err := th.Health(ctx)
		assert.Equal(t, health.HealthStatus{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("CloseShard", func(t *testing.T) {
		h.EXPECT().CloseShard(ctx, &shared.CloseShardRequest{}).Return(internalErr).Times(1)
		err := th.CloseShard(ctx, &shared.CloseShardRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("DescribeHistoryHost", func(t *testing.T) {
		h.EXPECT().DescribeHistoryHost(ctx, &shared.DescribeHistoryHostRequest{}).Return(&shared.DescribeHistoryHostResponse{}, internalErr).Times(1)
		resp, err := th.DescribeHistoryHost(ctx, &shared.DescribeHistoryHostRequest{})
		assert.Equal(t, shared.DescribeHistoryHostResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("DescribeMutableState", func(t *testing.T) {
		h.EXPECT().DescribeMutableState(ctx, &hist.DescribeMutableStateRequest{}).Return(&hist.DescribeMutableStateResponse{}, internalErr).Times(1)
		resp, err := th.DescribeMutableState(ctx, &hist.DescribeMutableStateRequest{})
		assert.Equal(t, hist.DescribeMutableStateResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("DescribeQueue", func(t *testing.T) {
		h.EXPECT().DescribeQueue(ctx, &shared.DescribeQueueRequest{}).Return(&shared.DescribeQueueResponse{}, internalErr).Times(1)
		resp, err := th.DescribeQueue(ctx, &shared.DescribeQueueRequest{})
		assert.Equal(t, shared.DescribeQueueResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("DescribeWorkflowExecution", func(t *testing.T) {
		h.EXPECT().DescribeWorkflowExecution(ctx, &hist.DescribeWorkflowExecutionRequest{}).Return(&shared.DescribeWorkflowExecutionResponse{}, internalErr).Times(1)
		resp, err := th.DescribeWorkflowExecution(ctx, &hist.DescribeWorkflowExecutionRequest{})
		assert.Equal(t, shared.DescribeWorkflowExecutionResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("GetDLQReplicationMessages", func(t *testing.T) {
		h.EXPECT().GetDLQReplicationMessages(ctx, &replicator.GetDLQReplicationMessagesRequest{}).Return(&replicator.GetDLQReplicationMessagesResponse{}, internalErr).Times(1)
		resp, err := th.GetDLQReplicationMessages(ctx, &replicator.GetDLQReplicationMessagesRequest{})
		assert.Equal(t, replicator.GetDLQReplicationMessagesResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("GetMutableState", func(t *testing.T) {
		h.EXPECT().GetMutableState(ctx, &hist.GetMutableStateRequest{}).Return(&hist.GetMutableStateResponse{}, internalErr).Times(1)
		resp, err := th.GetMutableState(ctx, &hist.GetMutableStateRequest{})
		assert.Equal(t, hist.GetMutableStateResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("GetReplicationMessages", func(t *testing.T) {
		h.EXPECT().GetReplicationMessages(ctx, &replicator.GetReplicationMessagesRequest{}).Return(&replicator.GetReplicationMessagesResponse{}, internalErr).Times(1)
		resp, err := th.GetReplicationMessages(ctx, &replicator.GetReplicationMessagesRequest{})
		assert.Equal(t, replicator.GetReplicationMessagesResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("MergeDLQMessages", func(t *testing.T) {
		h.EXPECT().MergeDLQMessages(ctx, &replicator.MergeDLQMessagesRequest{}).Return(&replicator.MergeDLQMessagesResponse{}, internalErr).Times(1)
		resp, err := th.MergeDLQMessages(ctx, &replicator.MergeDLQMessagesRequest{})
		assert.Equal(t, replicator.MergeDLQMessagesResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("NotifyFailoverMarkers", func(t *testing.T) {
		h.EXPECT().NotifyFailoverMarkers(ctx, &hist.NotifyFailoverMarkersRequest{}).Return(internalErr).Times(1)
		err := th.NotifyFailoverMarkers(ctx, &hist.NotifyFailoverMarkersRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("PollMutableState", func(t *testing.T) {
		h.EXPECT().PollMutableState(ctx, &hist.PollMutableStateRequest{}).Return(&hist.PollMutableStateResponse{}, internalErr).Times(1)
		resp, err := th.PollMutableState(ctx, &hist.PollMutableStateRequest{})
		assert.Equal(t, hist.PollMutableStateResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("PurgeDLQMessages", func(t *testing.T) {
		h.EXPECT().PurgeDLQMessages(ctx, &replicator.PurgeDLQMessagesRequest{}).Return(internalErr).Times(1)
		err := th.PurgeDLQMessages(ctx, &replicator.PurgeDLQMessagesRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("QueryWorkflow", func(t *testing.T) {
		h.EXPECT().QueryWorkflow(ctx, &hist.QueryWorkflowRequest{}).Return(&hist.QueryWorkflowResponse{}, internalErr).Times(1)
		resp, err := th.QueryWorkflow(ctx, &hist.QueryWorkflowRequest{})
		assert.Equal(t, hist.QueryWorkflowResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("ReadDLQMessages", func(t *testing.T) {
		h.EXPECT().ReadDLQMessages(ctx, &replicator.ReadDLQMessagesRequest{}).Return(&replicator.ReadDLQMessagesResponse{}, internalErr).Times(1)
		resp, err := th.ReadDLQMessages(ctx, &replicator.ReadDLQMessagesRequest{})
		assert.Equal(t, replicator.ReadDLQMessagesResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("ReapplyEvents", func(t *testing.T) {
		h.EXPECT().ReapplyEvents(ctx, &hist.ReapplyEventsRequest{}).Return(internalErr).Times(1)
		err := th.ReapplyEvents(ctx, &hist.ReapplyEventsRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RecordActivityTaskHeartbeat", func(t *testing.T) {
		h.EXPECT().RecordActivityTaskHeartbeat(ctx, &hist.RecordActivityTaskHeartbeatRequest{}).Return(&shared.RecordActivityTaskHeartbeatResponse{}, internalErr).Times(1)
		resp, err := th.RecordActivityTaskHeartbeat(ctx, &hist.RecordActivityTaskHeartbeatRequest{})
		assert.Equal(t, shared.RecordActivityTaskHeartbeatResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RecordActivityTaskStarted", func(t *testing.T) {
		h.EXPECT().RecordActivityTaskStarted(ctx, &hist.RecordActivityTaskStartedRequest{}).Return(&hist.RecordActivityTaskStartedResponse{}, internalErr).Times(1)
		resp, err := th.RecordActivityTaskStarted(ctx, &hist.RecordActivityTaskStartedRequest{})
		assert.Equal(t, hist.RecordActivityTaskStartedResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RecordChildExecutionCompleted", func(t *testing.T) {
		h.EXPECT().RecordChildExecutionCompleted(ctx, &hist.RecordChildExecutionCompletedRequest{}).Return(internalErr).Times(1)
		err := th.RecordChildExecutionCompleted(ctx, &hist.RecordChildExecutionCompletedRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RecordDecisionTaskStarted", func(t *testing.T) {
		h.EXPECT().RecordDecisionTaskStarted(ctx, &hist.RecordDecisionTaskStartedRequest{}).Return(&hist.RecordDecisionTaskStartedResponse{}, internalErr).Times(1)
		resp, err := th.RecordDecisionTaskStarted(ctx, &hist.RecordDecisionTaskStartedRequest{})
		assert.Equal(t, hist.RecordDecisionTaskStartedResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RefreshWorkflowTasks", func(t *testing.T) {
		h.EXPECT().RefreshWorkflowTasks(ctx, &hist.RefreshWorkflowTasksRequest{}).Return(internalErr).Times(1)
		err := th.RefreshWorkflowTasks(ctx, &hist.RefreshWorkflowTasksRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RemoveSignalMutableState", func(t *testing.T) {
		h.EXPECT().RemoveSignalMutableState(ctx, &hist.RemoveSignalMutableStateRequest{}).Return(internalErr).Times(1)
		err := th.RemoveSignalMutableState(ctx, &hist.RemoveSignalMutableStateRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RemoveTask", func(t *testing.T) {
		h.EXPECT().RemoveTask(ctx, &shared.RemoveTaskRequest{}).Return(internalErr).Times(1)
		err := th.RemoveTask(ctx, &shared.RemoveTaskRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("ReplicateEventsV2", func(t *testing.T) {
		h.EXPECT().ReplicateEventsV2(ctx, &hist.ReplicateEventsV2Request{}).Return(internalErr).Times(1)
		err := th.ReplicateEventsV2(ctx, &hist.ReplicateEventsV2Request{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RequestCancelWorkflowExecution", func(t *testing.T) {
		h.EXPECT().RequestCancelWorkflowExecution(ctx, &hist.RequestCancelWorkflowExecutionRequest{}).Return(internalErr).Times(1)
		err := th.RequestCancelWorkflowExecution(ctx, &hist.RequestCancelWorkflowExecutionRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("ResetQueue", func(t *testing.T) {
		h.EXPECT().ResetQueue(ctx, &shared.ResetQueueRequest{}).Return(internalErr).Times(1)
		err := th.ResetQueue(ctx, &shared.ResetQueueRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("ResetStickyTaskList", func(t *testing.T) {
		h.EXPECT().ResetStickyTaskList(ctx, &hist.ResetStickyTaskListRequest{}).Return(&hist.ResetStickyTaskListResponse{}, internalErr).Times(1)
		resp, err := th.ResetStickyTaskList(ctx, &hist.ResetStickyTaskListRequest{})
		assert.Equal(t, hist.ResetStickyTaskListResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("ResetWorkflowExecution", func(t *testing.T) {
		h.EXPECT().ResetWorkflowExecution(ctx, &hist.ResetWorkflowExecutionRequest{}).Return(&shared.ResetWorkflowExecutionResponse{}, internalErr).Times(1)
		resp, err := th.ResetWorkflowExecution(ctx, &hist.ResetWorkflowExecutionRequest{})
		assert.Equal(t, shared.ResetWorkflowExecutionResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RespondActivityTaskCanceled", func(t *testing.T) {
		h.EXPECT().RespondActivityTaskCanceled(ctx, &hist.RespondActivityTaskCanceledRequest{}).Return(internalErr).Times(1)
		err := th.RespondActivityTaskCanceled(ctx, &hist.RespondActivityTaskCanceledRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RespondActivityTaskCompleted", func(t *testing.T) {
		h.EXPECT().RespondActivityTaskCompleted(ctx, &hist.RespondActivityTaskCompletedRequest{}).Return(internalErr).Times(1)
		err := th.RespondActivityTaskCompleted(ctx, &hist.RespondActivityTaskCompletedRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RespondActivityTaskFailed", func(t *testing.T) {
		h.EXPECT().RespondActivityTaskFailed(ctx, &hist.RespondActivityTaskFailedRequest{}).Return(internalErr).Times(1)
		err := th.RespondActivityTaskFailed(ctx, &hist.RespondActivityTaskFailedRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RespondDecisionTaskCompleted", func(t *testing.T) {
		h.EXPECT().RespondDecisionTaskCompleted(ctx, &hist.RespondDecisionTaskCompletedRequest{}).Return(&hist.RespondDecisionTaskCompletedResponse{}, internalErr).Times(1)
		resp, err := th.RespondDecisionTaskCompleted(ctx, &hist.RespondDecisionTaskCompletedRequest{})
		assert.Equal(t, hist.RespondDecisionTaskCompletedResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("RespondDecisionTaskFailed", func(t *testing.T) {
		h.EXPECT().RespondDecisionTaskFailed(ctx, &hist.RespondDecisionTaskFailedRequest{}).Return(internalErr).Times(1)
		err := th.RespondDecisionTaskFailed(ctx, &hist.RespondDecisionTaskFailedRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("ScheduleDecisionTask", func(t *testing.T) {
		h.EXPECT().ScheduleDecisionTask(ctx, &hist.ScheduleDecisionTaskRequest{}).Return(internalErr).Times(1)
		err := th.ScheduleDecisionTask(ctx, &hist.ScheduleDecisionTaskRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("SignalWithStartWorkflowExecution", func(t *testing.T) {
		h.EXPECT().SignalWithStartWorkflowExecution(ctx, &hist.SignalWithStartWorkflowExecutionRequest{}).Return(&shared.StartWorkflowExecutionResponse{}, internalErr).Times(1)
		resp, err := th.SignalWithStartWorkflowExecution(ctx, &hist.SignalWithStartWorkflowExecutionRequest{})
		assert.Equal(t, shared.StartWorkflowExecutionResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("SignalWorkflowExecution", func(t *testing.T) {
		h.EXPECT().SignalWorkflowExecution(ctx, &hist.SignalWorkflowExecutionRequest{}).Return(internalErr).Times(1)
		err := th.SignalWorkflowExecution(ctx, &hist.SignalWorkflowExecutionRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("StartWorkflowExecution", func(t *testing.T) {
		h.EXPECT().StartWorkflowExecution(ctx, &hist.StartWorkflowExecutionRequest{}).Return(&shared.StartWorkflowExecutionResponse{}, internalErr).Times(1)
		resp, err := th.StartWorkflowExecution(ctx, &hist.StartWorkflowExecutionRequest{})
		assert.Equal(t, shared.StartWorkflowExecutionResponse{}, *resp)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("SyncActivity", func(t *testing.T) {
		h.EXPECT().SyncActivity(ctx, &hist.SyncActivityRequest{}).Return(internalErr).Times(1)
		err := th.SyncActivity(ctx, &hist.SyncActivityRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("SyncShardStatus", func(t *testing.T) {
		h.EXPECT().SyncShardStatus(ctx, &hist.SyncShardStatusRequest{}).Return(internalErr).Times(1)
		err := th.SyncShardStatus(ctx, &hist.SyncShardStatusRequest{})
		assert.Equal(t, expectedErr, err)
	})
	t.Run("TerminateWorkflowExecution", func(t *testing.T) {
		h.EXPECT().TerminateWorkflowExecution(ctx, &hist.TerminateWorkflowExecutionRequest{}).Return(internalErr).Times(1)
		err := th.TerminateWorkflowExecution(ctx, &hist.TerminateWorkflowExecutionRequest{})
		assert.Equal(t, expectedErr, err)
	})
}
