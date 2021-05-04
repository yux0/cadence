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

//go:generate mockgen -package $GOPACKAGE -source $GOFILE -destination dlq_handler_mock.go

package replication

import (
	"context"

	"github.com/uber/cadence/.gen/go/replicator"
	workflow "github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/common/types/mapper/thrift"
	"github.com/uber/cadence/service/history/shard"
)

const (
	defaultBeginningMessageID = -1
)

var (
	errInvalidCluster = &workflow.BadRequestError{Message: "Invalid target cluster name."}
)

type (
	// DLQHandler is the interface handles replication DLQ messages
	DLQHandler interface {
		ReadMessages(
			ctx context.Context,
			sourceCluster string,
			lastMessageID int64,
			pageSize int,
			pageToken []byte,
		) ([]*replicator.ReplicationTask, []byte, error)
		PurgeMessages(
			ctx context.Context,
			sourceCluster string,
			lastMessageID int64,
		) error
		MergeMessages(
			ctx context.Context,
			sourceCluster string,
			lastMessageID int64,
			pageSize int,
			pageToken []byte,
		) ([]byte, error)
	}

	dlqHandlerImpl struct {
		taskExecutors map[string]TaskExecutor
		shard         shard.Context
		logger        log.Logger
	}
)

var _ DLQHandler = (*dlqHandlerImpl)(nil)

// NewDLQHandler initialize the replication message DLQ handler
func NewDLQHandler(
	shard shard.Context,
	taskExecutors map[string]TaskExecutor,
) DLQHandler {

	if taskExecutors == nil {
		panic("Failed to initialize replication DLQ handler due to nil task executors")
	}

	return &dlqHandlerImpl{
		shard:         shard,
		taskExecutors: taskExecutors,
		logger:        shard.GetLogger(),
	}
}

func (r *dlqHandlerImpl) ReadMessages(
	ctx context.Context,
	sourceCluster string,
	lastMessageID int64,
	pageSize int,
	pageToken []byte,
) ([]*replicator.ReplicationTask, []byte, error) {

	return r.readMessagesWithAckLevel(
		ctx,
		sourceCluster,
		lastMessageID,
		pageSize,
		pageToken,
	)
}

func (r *dlqHandlerImpl) readMessagesWithAckLevel(
	ctx context.Context,
	sourceCluster string,
	lastMessageID int64,
	pageSize int,
	pageToken []byte,
) ([]*replicator.ReplicationTask, []byte, error) {

	resp, err := r.shard.GetExecutionManager().GetReplicationTasksFromDLQ(
		ctx,
		&persistence.GetReplicationTasksFromDLQRequest{
			SourceClusterName: sourceCluster,
			GetReplicationTasksRequest: persistence.GetReplicationTasksRequest{
				ReadLevel:     defaultBeginningMessageID,
				MaxReadLevel:  lastMessageID,
				BatchSize:     pageSize,
				NextPageToken: pageToken,
			},
		},
	)
	if err != nil {
		return nil, nil, err
	}

	remoteAdminClient := r.shard.GetService().GetClientBean().GetRemoteAdminClient(sourceCluster)
	if remoteAdminClient == nil {
		return nil, nil, errInvalidCluster
	}

	taskInfo := make([]*types.ReplicationTaskInfo, 0, len(resp.Tasks))
	for _, task := range resp.Tasks {
		taskInfo = append(taskInfo, &types.ReplicationTaskInfo{
			DomainID:     common.StringPtr(task.GetDomainID()),
			WorkflowID:   common.StringPtr(task.GetWorkflowID()),
			RunID:        common.StringPtr(task.GetRunID()),
			TaskType:     common.Int16Ptr(int16(task.GetTaskType())),
			TaskID:       common.Int64Ptr(task.GetTaskID()),
			Version:      common.Int64Ptr(task.GetVersion()),
			FirstEventID: common.Int64Ptr(task.FirstEventID),
			NextEventID:  common.Int64Ptr(task.NextEventID),
			ScheduledID:  common.Int64Ptr(task.ScheduledID),
		})
	}
	response := &replicator.GetDLQReplicationMessagesResponse{}
	if len(taskInfo) > 0 {
		clientResp, err := remoteAdminClient.GetDLQReplicationMessages(
			ctx,
			&types.GetDLQReplicationMessagesRequest{
				TaskInfos: taskInfo,
			},
		)
		response = thrift.FromGetDLQReplicationMessagesResponse(clientResp)
		err = thrift.FromError(err)
		if err != nil {
			return nil, nil, err
		}
	}

	return response.ReplicationTasks, resp.NextPageToken, nil
}

func (r *dlqHandlerImpl) PurgeMessages(
	ctx context.Context,
	sourceCluster string,
	lastMessageID int64,
) error {

	err := r.shard.GetExecutionManager().RangeDeleteReplicationTaskFromDLQ(
		ctx,
		&persistence.RangeDeleteReplicationTaskFromDLQRequest{
			SourceClusterName:    sourceCluster,
			ExclusiveBeginTaskID: defaultBeginningMessageID,
			InclusiveEndTaskID:   lastMessageID,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *dlqHandlerImpl) MergeMessages(
	ctx context.Context,
	sourceCluster string,
	lastMessageID int64,
	pageSize int,
	pageToken []byte,
) ([]byte, error) {

	if _, ok := r.taskExecutors[sourceCluster]; !ok {
		return nil, errInvalidCluster
	}

	tasks, token, err := r.readMessagesWithAckLevel(
		ctx,
		sourceCluster,
		lastMessageID,
		pageSize,
		pageToken,
	)

	lastMessageID = defaultBeginningMessageID
	for _, task := range tasks {
		if _, err := r.taskExecutors[sourceCluster].execute(
			task,
			true,
		); err != nil {
			return nil, err
		}

		if lastMessageID < task.GetSourceTaskId() {
			lastMessageID = task.GetSourceTaskId()
		}
	}

	err = r.shard.GetExecutionManager().RangeDeleteReplicationTaskFromDLQ(
		ctx,
		&persistence.RangeDeleteReplicationTaskFromDLQRequest{
			SourceClusterName:    sourceCluster,
			ExclusiveBeginTaskID: defaultBeginningMessageID,
			InclusiveEndTaskID:   lastMessageID,
		},
	)
	if err != nil {
		return nil, err
	}
	return token, nil
}
