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

	"go.uber.org/yarpc"

	"github.com/uber/cadence/common/types"
)

//go:generate mockgen -package $GOPACKAGE -source $GOFILE -destination interface_mock.go -package history github.com/uber/cadence/client/history Client

// Client is the interface exposed by history service client
type Client interface {
	CloseShard(context.Context, *types.CloseShardRequest, ...yarpc.CallOption) error
	DescribeHistoryHost(context.Context, *types.DescribeHistoryHostRequest, ...yarpc.CallOption) (*types.DescribeHistoryHostResponse, error)
	DescribeMutableState(context.Context, *types.DescribeMutableStateRequest, ...yarpc.CallOption) (*types.DescribeMutableStateResponse, error)
	DescribeQueue(context.Context, *types.DescribeQueueRequest, ...yarpc.CallOption) (*types.DescribeQueueResponse, error)
	DescribeWorkflowExecution(context.Context, *types.HistoryDescribeWorkflowExecutionRequest, ...yarpc.CallOption) (*types.DescribeWorkflowExecutionResponse, error)
	GetDLQReplicationMessages(context.Context, *types.GetDLQReplicationMessagesRequest, ...yarpc.CallOption) (*types.GetDLQReplicationMessagesResponse, error)
	GetMutableState(context.Context, *types.GetMutableStateRequest, ...yarpc.CallOption) (*types.GetMutableStateResponse, error)
	GetReplicationMessages(context.Context, *types.GetReplicationMessagesRequest, ...yarpc.CallOption) (*types.GetReplicationMessagesResponse, error)
	MergeDLQMessages(context.Context, *types.MergeDLQMessagesRequest, ...yarpc.CallOption) (*types.MergeDLQMessagesResponse, error)
	NotifyFailoverMarkers(context.Context, *types.NotifyFailoverMarkersRequest, ...yarpc.CallOption) error
	PollMutableState(context.Context, *types.PollMutableStateRequest, ...yarpc.CallOption) (*types.PollMutableStateResponse, error)
	PurgeDLQMessages(context.Context, *types.PurgeDLQMessagesRequest, ...yarpc.CallOption) error
	QueryWorkflow(context.Context, *types.HistoryQueryWorkflowRequest, ...yarpc.CallOption) (*types.HistoryQueryWorkflowResponse, error)
	ReadDLQMessages(context.Context, *types.ReadDLQMessagesRequest, ...yarpc.CallOption) (*types.ReadDLQMessagesResponse, error)
	ReapplyEvents(context.Context, *types.HistoryReapplyEventsRequest, ...yarpc.CallOption) error
	RecordActivityTaskHeartbeat(context.Context, *types.HistoryRecordActivityTaskHeartbeatRequest, ...yarpc.CallOption) (*types.RecordActivityTaskHeartbeatResponse, error)
	RecordActivityTaskStarted(context.Context, *types.RecordActivityTaskStartedRequest, ...yarpc.CallOption) (*types.RecordActivityTaskStartedResponse, error)
	RecordChildExecutionCompleted(context.Context, *types.RecordChildExecutionCompletedRequest, ...yarpc.CallOption) error
	RecordDecisionTaskStarted(context.Context, *types.RecordDecisionTaskStartedRequest, ...yarpc.CallOption) (*types.RecordDecisionTaskStartedResponse, error)
	RefreshWorkflowTasks(context.Context, *types.HistoryRefreshWorkflowTasksRequest, ...yarpc.CallOption) error
	RemoveSignalMutableState(context.Context, *types.RemoveSignalMutableStateRequest, ...yarpc.CallOption) error
	RemoveTask(context.Context, *types.RemoveTaskRequest, ...yarpc.CallOption) error
	ReplicateEventsV2(context.Context, *types.ReplicateEventsV2Request, ...yarpc.CallOption) error
	RequestCancelWorkflowExecution(context.Context, *types.HistoryRequestCancelWorkflowExecutionRequest, ...yarpc.CallOption) error
	ResetQueue(context.Context, *types.ResetQueueRequest, ...yarpc.CallOption) error
	ResetStickyTaskList(context.Context, *types.HistoryResetStickyTaskListRequest, ...yarpc.CallOption) (*types.HistoryResetStickyTaskListResponse, error)
	ResetWorkflowExecution(context.Context, *types.HistoryResetWorkflowExecutionRequest, ...yarpc.CallOption) (*types.ResetWorkflowExecutionResponse, error)
	RespondActivityTaskCanceled(context.Context, *types.HistoryRespondActivityTaskCanceledRequest, ...yarpc.CallOption) error
	RespondActivityTaskCompleted(context.Context, *types.HistoryRespondActivityTaskCompletedRequest, ...yarpc.CallOption) error
	RespondActivityTaskFailed(context.Context, *types.HistoryRespondActivityTaskFailedRequest, ...yarpc.CallOption) error
	RespondDecisionTaskCompleted(context.Context, *types.HistoryRespondDecisionTaskCompletedRequest, ...yarpc.CallOption) (*types.HistoryRespondDecisionTaskCompletedResponse, error)
	RespondDecisionTaskFailed(context.Context, *types.HistoryRespondDecisionTaskFailedRequest, ...yarpc.CallOption) error
	ScheduleDecisionTask(context.Context, *types.ScheduleDecisionTaskRequest, ...yarpc.CallOption) error
	SignalWithStartWorkflowExecution(context.Context, *types.HistorySignalWithStartWorkflowExecutionRequest, ...yarpc.CallOption) (*types.StartWorkflowExecutionResponse, error)
	SignalWorkflowExecution(context.Context, *types.HistorySignalWorkflowExecutionRequest, ...yarpc.CallOption) error
	StartWorkflowExecution(context.Context, *types.HistoryStartWorkflowExecutionRequest, ...yarpc.CallOption) (*types.StartWorkflowExecutionResponse, error)
	SyncActivity(context.Context, *types.SyncActivityRequest, ...yarpc.CallOption) error
	SyncShardStatus(context.Context, *types.SyncShardStatusRequest, ...yarpc.CallOption) error
	TerminateWorkflowExecution(context.Context, *types.HistoryTerminateWorkflowExecutionRequest, ...yarpc.CallOption) error
}
