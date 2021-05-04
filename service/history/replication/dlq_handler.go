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

//go:generate mockgen -copyright_file ../../../LICENSE -package $GOPACKAGE -source $GOFILE -destination dlq_handler_mock.go

package replication

import (
	"context"

	"github.com/uber/cadence/.gen/go/replicator"
	workflow "github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/service/history/shard"
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

	tasks, _, token, err := r.readMessagesWithAckLevel(
		ctx,
		sourceCluster,
		lastMessageID,
		pageSize,
		pageToken,
	)
	return tasks, token, err
}

func (r *dlqHandlerImpl) readMessagesWithAckLevel(
	ctx context.Context,
	sourceCluster string,
	lastMessageID int64,
	pageSize int,
	pageToken []byte,
) ([]*replicator.ReplicationTask, int64, []byte, error) {

	ackLevel := r.shard.GetReplicatorDLQAckLevel(sourceCluster)
	resp, err := r.shard.GetExecutionManager().GetReplicationTasksFromDLQ(&persistence.GetReplicationTasksFromDLQRequest{
		SourceClusterName: sourceCluster,
		GetReplicationTasksRequest: persistence.GetReplicationTasksRequest{
			ReadLevel:     ackLevel,
			MaxReadLevel:  lastMessageID,
			BatchSize:     pageSize,
			NextPageToken: pageToken,
		},
	})
	if err != nil {
		return nil, ackLevel, nil, err
	}

	remoteAdminClient := r.shard.GetService().GetClientBean().GetRemoteAdminClient(sourceCluster)
	taskInfo := make([]*replicator.ReplicationTaskInfo, 0, len(resp.Tasks))
	for _, task := range resp.Tasks {
		taskInfo = append(taskInfo, &replicator.ReplicationTaskInfo{
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
		response, err = remoteAdminClient.GetDLQReplicationMessages(
			ctx,
			&replicator.GetDLQReplicationMessagesRequest{
				TaskInfos: taskInfo,
			},
		)
		if err != nil {
			return nil, ackLevel, nil, err
		}
	}

	return response.ReplicationTasks, ackLevel, resp.NextPageToken, nil
}

func (r *dlqHandlerImpl) PurgeMessages(
	sourceCluster string,
	lastMessageID int64,
) error {

	ackLevel := r.shard.GetReplicatorDLQAckLevel(sourceCluster)
	err := r.shard.GetExecutionManager().RangeDeleteReplicationTaskFromDLQ(
		&persistence.RangeDeleteReplicationTaskFromDLQRequest{
			SourceClusterName:    sourceCluster,
			ExclusiveBeginTaskID: ackLevel,
			InclusiveEndTaskID:   lastMessageID,
		},
	)
	if err != nil {
		return err
	}

	if err = r.shard.UpdateReplicatorDLQAckLevel(
		sourceCluster,
		lastMessageID,
	); err != nil {
		r.logger.Error("Failed to purge history replication message", tag.Error(err))
		// The update ack level should not block the call. Ignore the error.
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

	tasks, ackLevel, token, err := r.readMessagesWithAckLevel(
		ctx,
		sourceCluster,
		lastMessageID,
		pageSize,
		pageToken,
	)

	for _, task := range tasks {
		if _, err := r.taskExecutors[sourceCluster].execute(
			task,
			true,
		); err != nil {
			return nil, err
		}
	}

	err = r.shard.GetExecutionManager().RangeDeleteReplicationTaskFromDLQ(
		&persistence.RangeDeleteReplicationTaskFromDLQRequest{
			SourceClusterName:    sourceCluster,
			ExclusiveBeginTaskID: ackLevel,
			InclusiveEndTaskID:   lastMessageID,
		},
	)
	if err != nil {
		return nil, err
	}

	if err = r.shard.UpdateReplicatorDLQAckLevel(
		sourceCluster,
		lastMessageID,
	); err != nil {
		r.logger.Error("Failed to merge history replication message", tag.Error(err))
		// The update ack level should not block the call. Ignore the error.
	}
	return token, nil
}
