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

//go:generate mockgen -package $GOPACKAGE -source $GOFILE -destination dlqMessageHandler_mock.go

package domain

import (
	"context"

	"github.com/uber/cadence/.gen/go/replicator"
	"github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
)

type (
	// DLQMessageHandler is the interface handles domain DLQ messages
	DLQMessageHandler interface {
		Read(ctx context.Context, lastMessageID int64, pageSize int, pageToken []byte) ([]*replicator.ReplicationTask, []byte, error)
		Purge(ctx context.Context, lastMessageID int64) error
		Merge(ctx context.Context, lastMessageID int64, pageSize int, pageToken []byte) ([]byte, error)
	}

	dlqMessageHandlerImpl struct {
		replicationHandler ReplicationTaskExecutor
		replicationQueue   ReplicationQueue
		logger             log.Logger
	}
)

// NewDLQMessageHandler returns a DLQTaskHandler instance
func NewDLQMessageHandler(
	replicationHandler ReplicationTaskExecutor,
	replicationQueue ReplicationQueue,
	logger log.Logger,
) DLQMessageHandler {
	return &dlqMessageHandlerImpl{
		replicationHandler: replicationHandler,
		replicationQueue:   replicationQueue,
		logger:             logger,
	}
}

// ReadMessages reads domain replication DLQ messages
func (d *dlqMessageHandlerImpl) Read(
	ctx context.Context,
	lastMessageID int64,
	pageSize int,
	pageToken []byte,
) ([]*replicator.ReplicationTask, []byte, error) {

	ackLevel, err := d.replicationQueue.GetDLQAckLevel(ctx)
	if err != nil {
		return nil, nil, err
	}

	return d.replicationQueue.GetMessagesFromDLQ(
		ctx,
		ackLevel,
		lastMessageID,
		pageSize,
		pageToken,
	)
}

// PurgeMessages purges domain replication DLQ messages
func (d *dlqMessageHandlerImpl) Purge(
	ctx context.Context,
	lastMessageID int64,
) error {

	ackLevel, err := d.replicationQueue.GetDLQAckLevel(ctx)
	if err != nil {
		return err
	}

	if err := d.replicationQueue.RangeDeleteMessagesFromDLQ(
		ctx,
		ackLevel,
		lastMessageID,
	); err != nil {
		return err
	}

	if err := d.replicationQueue.UpdateDLQAckLevel(
		ctx,
		lastMessageID,
	); err != nil {
		d.logger.Error("Failed to update DLQ ack level after purging messages", tag.Error(err))
	}

	return nil
}

// MergeMessages merges domain replication DLQ messages
func (d *dlqMessageHandlerImpl) Merge(
	ctx context.Context,
	lastMessageID int64,
	pageSize int,
	pageToken []byte,
) ([]byte, error) {

	ackLevel, err := d.replicationQueue.GetDLQAckLevel(ctx)
	if err != nil {
		return nil, err
	}

	messages, token, err := d.replicationQueue.GetMessagesFromDLQ(
		ctx,
		ackLevel,
		lastMessageID,
		pageSize,
		pageToken,
	)
	if err != nil {
		return nil, err
	}

	var ackedMessageID int64
	for _, message := range messages {
		domainTask := message.GetDomainTaskAttributes()
		if domainTask == nil {
			return nil, &shared.InternalServiceError{Message: "Encounter non domain replication task in domain replication queue."}
		}

		// TODO:
		if err := d.replicationHandler.Execute(
			domainTask,
		); err != nil {
			return nil, err
		}
		ackedMessageID = *message.SourceTaskId
	}

	if err := d.replicationQueue.RangeDeleteMessagesFromDLQ(
		ctx,
		ackLevel,
		ackedMessageID,
	); err != nil {
		d.logger.Error("failed to delete merged tasks on merging domain DLQ message", tag.Error(err))
		return nil, err
	}
	if err := d.replicationQueue.UpdateDLQAckLevel(ctx, ackedMessageID); err != nil {
		d.logger.Error("failed to update ack level on merging domain DLQ message", tag.Error(err))
	}

	return token, nil
}
