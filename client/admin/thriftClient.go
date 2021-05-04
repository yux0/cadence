// Copyright (c) 2020 Uber Technologies, Inc.
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

package admin

import (
	"context"

	"go.uber.org/yarpc"

	"github.com/uber/cadence/.gen/go/admin/adminserviceclient"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/common/types/mapper/thrift"
)

type thriftClient struct {
	c adminserviceclient.Interface
}

// NewThriftClient creates a new instance of Client with thrift protocol
func NewThriftClient(c adminserviceclient.Interface) Client {
	return thriftClient{c}
}

func (t thriftClient) AddSearchAttribute(ctx context.Context, request *types.AddSearchAttributeRequest, opts ...yarpc.CallOption) error {
	err := t.c.AddSearchAttribute(ctx, thrift.FromAddSearchAttributeRequest(request), opts...)
	return thrift.ToError(err)
}

func (t thriftClient) CloseShard(ctx context.Context, request *types.CloseShardRequest, opts ...yarpc.CallOption) error {
	err := t.c.CloseShard(ctx, thrift.FromCloseShardRequest(request), opts...)
	return thrift.ToError(err)
}

func (t thriftClient) DescribeCluster(ctx context.Context, opts ...yarpc.CallOption) (*types.DescribeClusterResponse, error) {
	response, err := t.c.DescribeCluster(ctx, opts...)
	return thrift.ToDescribeClusterResponse(response), thrift.ToError(err)
}

func (t thriftClient) DescribeHistoryHost(ctx context.Context, request *types.DescribeHistoryHostRequest, opts ...yarpc.CallOption) (*types.DescribeHistoryHostResponse, error) {
	response, err := t.c.DescribeHistoryHost(ctx, thrift.FromDescribeHistoryHostRequest(request), opts...)
	return thrift.ToDescribeHistoryHostResponse(response), thrift.ToError(err)
}

func (t thriftClient) DescribeQueue(ctx context.Context, request *types.DescribeQueueRequest, opts ...yarpc.CallOption) (*types.DescribeQueueResponse, error) {
	response, err := t.c.DescribeQueue(ctx, thrift.FromDescribeQueueRequest(request), opts...)
	return thrift.ToDescribeQueueResponse(response), thrift.ToError(err)
}

func (t thriftClient) DescribeWorkflowExecution(ctx context.Context, request *types.AdminDescribeWorkflowExecutionRequest, opts ...yarpc.CallOption) (*types.AdminDescribeWorkflowExecutionResponse, error) {
	response, err := t.c.DescribeWorkflowExecution(ctx, thrift.FromAdminDescribeWorkflowExecutionRequest(request), opts...)
	return thrift.ToAdminDescribeWorkflowExecutionResponse(response), thrift.ToError(err)
}

func (t thriftClient) GetDLQReplicationMessages(ctx context.Context, request *types.GetDLQReplicationMessagesRequest, opts ...yarpc.CallOption) (*types.GetDLQReplicationMessagesResponse, error) {
	response, err := t.c.GetDLQReplicationMessages(ctx, thrift.FromGetDLQReplicationMessagesRequest(request), opts...)
	return thrift.ToGetDLQReplicationMessagesResponse(response), thrift.ToError(err)
}

func (t thriftClient) GetDomainReplicationMessages(ctx context.Context, request *types.GetDomainReplicationMessagesRequest, opts ...yarpc.CallOption) (*types.GetDomainReplicationMessagesResponse, error) {
	response, err := t.c.GetDomainReplicationMessages(ctx, thrift.FromGetDomainReplicationMessagesRequest(request), opts...)
	return thrift.ToGetDomainReplicationMessagesResponse(response), thrift.ToError(err)
}

func (t thriftClient) GetReplicationMessages(ctx context.Context, request *types.GetReplicationMessagesRequest, opts ...yarpc.CallOption) (*types.GetReplicationMessagesResponse, error) {
	response, err := t.c.GetReplicationMessages(ctx, thrift.FromGetReplicationMessagesRequest(request), opts...)
	return thrift.ToGetReplicationMessagesResponse(response), thrift.ToError(err)
}

func (t thriftClient) GetWorkflowExecutionRawHistoryV2(ctx context.Context, request *types.GetWorkflowExecutionRawHistoryV2Request, opts ...yarpc.CallOption) (*types.GetWorkflowExecutionRawHistoryV2Response, error) {
	response, err := t.c.GetWorkflowExecutionRawHistoryV2(ctx, thrift.FromGetWorkflowExecutionRawHistoryV2Request(request), opts...)
	return thrift.ToGetWorkflowExecutionRawHistoryV2Response(response), thrift.ToError(err)
}

func (t thriftClient) MergeDLQMessages(ctx context.Context, request *types.MergeDLQMessagesRequest, opts ...yarpc.CallOption) (*types.MergeDLQMessagesResponse, error) {
	response, err := t.c.MergeDLQMessages(ctx, thrift.FromMergeDLQMessagesRequest(request), opts...)
	return thrift.ToMergeDLQMessagesResponse(response), thrift.ToError(err)
}

func (t thriftClient) PurgeDLQMessages(ctx context.Context, request *types.PurgeDLQMessagesRequest, opts ...yarpc.CallOption) error {
	err := t.c.PurgeDLQMessages(ctx, thrift.FromPurgeDLQMessagesRequest(request), opts...)
	return thrift.ToError(err)
}

func (t thriftClient) ReadDLQMessages(ctx context.Context, request *types.ReadDLQMessagesRequest, opts ...yarpc.CallOption) (*types.ReadDLQMessagesResponse, error) {
	response, err := t.c.ReadDLQMessages(ctx, thrift.FromReadDLQMessagesRequest(request), opts...)
	return thrift.ToReadDLQMessagesResponse(response), thrift.ToError(err)
}

func (t thriftClient) ReapplyEvents(ctx context.Context, request *types.ReapplyEventsRequest, opts ...yarpc.CallOption) error {
	err := t.c.ReapplyEvents(ctx, thrift.FromReapplyEventsRequest(request), opts...)
	return thrift.ToError(err)
}

func (t thriftClient) RefreshWorkflowTasks(ctx context.Context, request *types.RefreshWorkflowTasksRequest, opts ...yarpc.CallOption) error {
	err := t.c.RefreshWorkflowTasks(ctx, thrift.FromRefreshWorkflowTasksRequest(request), opts...)
	return thrift.ToError(err)
}

func (t thriftClient) RemoveTask(ctx context.Context, request *types.RemoveTaskRequest, opts ...yarpc.CallOption) error {
	err := t.c.RemoveTask(ctx, thrift.FromRemoveTaskRequest(request), opts...)
	return thrift.ToError(err)
}

func (t thriftClient) ResendReplicationTasks(ctx context.Context, request *types.ResendReplicationTasksRequest, opts ...yarpc.CallOption) error {
	err := t.c.ResendReplicationTasks(ctx, thrift.FromResendReplicationTasksRequest(request), opts...)
	return thrift.ToError(err)
}

func (t thriftClient) ResetQueue(ctx context.Context, request *types.ResetQueueRequest, opts ...yarpc.CallOption) error {
	err := t.c.ResetQueue(ctx, thrift.FromResetQueueRequest(request), opts...)
	return thrift.ToError(err)
}
