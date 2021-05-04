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

package sql

import (
	"context"
	"fmt"

	"database/sql"

	workflow "github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/persistence/sql/sqlplugin"
)

const (
	emptyMessageID = -1
)

type (
	sqlQueue struct {
		queueType persistence.QueueType
		logger    log.Logger
		sqlStore
	}
)

func newQueue(
	db sqlplugin.DB,
	logger log.Logger,
	queueType persistence.QueueType,
) (persistence.Queue, error) {
	return &sqlQueue{
		sqlStore: sqlStore{
			db:     db,
			logger: logger,
		},
		queueType: queueType,
		logger:    logger,
	}, nil
}

func (q *sqlQueue) EnqueueMessage(
	ctx context.Context,
	messagePayload []byte,
) error {

	err := q.txExecute(ctx, "EnqueueMessage", func(tx sqlplugin.Tx) error {
		lastMessageID, err := tx.GetLastEnqueuedMessageIDForUpdate(ctx, q.queueType)
		if err != nil {
			if err == sql.ErrNoRows {
				lastMessageID = -1
			} else {
				return fmt.Errorf("failed to get last enqueued message id: %v", err)
			}
		}

		_, err = tx.InsertIntoQueue(ctx, newQueueRow(q.queueType, lastMessageID+1, messagePayload))
		return err
	})
	if err != nil {
		return &workflow.InternalServiceError{Message: err.Error()}
	}
	return nil
}

func (q *sqlQueue) ReadMessages(
	ctx context.Context,
	lastMessageID int64,
	maxCount int,
) ([]*persistence.InternalQueueMessage, error) {

	rows, err := q.db.GetMessagesFromQueue(ctx, q.queueType, lastMessageID, maxCount)
	if err != nil {
		return nil, err
	}

	var messages []*persistence.InternalQueueMessage
	for _, row := range rows {
		messages = append(messages, &persistence.InternalQueueMessage{ID: row.MessageID, Payload: row.MessagePayload})
	}
	return messages, nil
}

func newQueueRow(
	queueType persistence.QueueType,
	messageID int64,
	payload []byte,
) *sqlplugin.QueueRow {

	return &sqlplugin.QueueRow{QueueType: queueType, MessageID: messageID, MessagePayload: payload}
}

func (q *sqlQueue) DeleteMessagesBefore(
	ctx context.Context,
	messageID int64,
) error {

	_, err := q.db.DeleteMessagesBefore(ctx, q.queueType, messageID)
	if err != nil {
		return &workflow.InternalServiceError{
			Message: fmt.Sprintf("DeleteMessagesBefore operation failed. Error %v", err),
		}
	}
	return nil
}

func (q *sqlQueue) UpdateAckLevel(
	ctx context.Context,
	messageID int64,
	clusterName string,
) error {

	err := q.txExecute(ctx, "UpdateAckLevel", func(tx sqlplugin.Tx) error {
		clusterAckLevels, err := tx.GetAckLevels(ctx, q.queueType, true)
		if err != nil {
			return &workflow.InternalServiceError{
				Message: fmt.Sprintf("UpdateAckLevel operation failed. Error %v", err),
			}
		}

		if clusterAckLevels == nil {
			err := tx.InsertAckLevel(ctx, q.queueType, messageID, clusterName)
			if err != nil {
				return &workflow.InternalServiceError{
					Message: fmt.Sprintf("UpdateAckLevel operation failed. Error %v", err),
				}
			}
			return nil
		}

		// Ignore possibly delayed message
		if clusterAckLevels[clusterName] > messageID {
			return nil
		}

		clusterAckLevels[clusterName] = messageID
		err = tx.UpdateAckLevels(ctx, q.queueType, clusterAckLevels)
		if err != nil {
			return &workflow.InternalServiceError{
				Message: fmt.Sprintf("UpdateAckLevel operation failed. Error %v", err),
			}
		}
		return nil
	})

	if err != nil {
		return &workflow.InternalServiceError{Message: err.Error()}
	}
	return nil
}

func (q *sqlQueue) GetAckLevels(
	ctx context.Context,
) (map[string]int64, error) {
	return q.db.GetAckLevels(ctx, q.queueType, false)
}

func (q *sqlQueue) EnqueueMessageToDLQ(
	ctx context.Context,
	messagePayload []byte,
) error {

	err := q.txExecute(ctx, "EnqueueMessageToDLQ", func(tx sqlplugin.Tx) error {
		var err error
		lastMessageID, err := tx.GetLastEnqueuedMessageIDForUpdate(ctx, q.getDLQTypeFromQueueType())
		if err != nil {
			if err == sql.ErrNoRows {
				lastMessageID = -1
			} else {
				return fmt.Errorf("failed to get last enqueued message id from DLQ: %v", err)
			}
		}
		_, err = tx.InsertIntoQueue(ctx, newQueueRow(q.getDLQTypeFromQueueType(), lastMessageID+1, messagePayload))
		return err
	})
	if err != nil {
		return &workflow.InternalServiceError{Message: err.Error()}
	}
	return nil
}

func (q *sqlQueue) ReadMessagesFromDLQ(
	ctx context.Context,
	firstMessageID int64,
	lastMessageID int64,
	pageSize int,
	pageToken []byte,
) ([]*persistence.InternalQueueMessage, []byte, error) {

	if pageToken != nil && len(pageToken) != 0 {
		lastReadMessageID, err := deserializePageToken(pageToken)
		if err != nil {
			return nil, nil, &workflow.InternalServiceError{
				Message: fmt.Sprintf("invalid next page token %v", pageToken)}
		}
		firstMessageID = lastReadMessageID
	}

	rows, err := q.db.GetMessagesBetween(ctx, q.getDLQTypeFromQueueType(), firstMessageID, lastMessageID, pageSize)
	if err != nil {
		return nil, nil, &workflow.InternalServiceError{
			Message: fmt.Sprintf("ReadMessagesFromDLQ operation failed. Error %v", err),
		}
	}

	var messages []*persistence.InternalQueueMessage
	for _, row := range rows {
		messages = append(messages, &persistence.InternalQueueMessage{ID: row.MessageID, Payload: row.MessagePayload})
	}

	var newPagingToken []byte
	if messages != nil && len(messages) >= pageSize {
		lastReadMessageID := messages[len(messages)-1].ID
		newPagingToken = serializePageToken(int64(lastReadMessageID))
	}
	return messages, newPagingToken, nil
}

func (q *sqlQueue) DeleteMessageFromDLQ(
	ctx context.Context,
	messageID int64,
) error {

	_, err := q.db.DeleteMessage(ctx, q.getDLQTypeFromQueueType(), messageID)
	if err != nil {
		return &workflow.InternalServiceError{
			Message: fmt.Sprintf("DeleteMessageFromDLQ operation failed. Error %v", err),
		}
	}
	return nil
}

func (q *sqlQueue) RangeDeleteMessagesFromDLQ(
	ctx context.Context,
	firstMessageID int64,
	lastMessageID int64,
) error {

	_, err := q.db.RangeDeleteMessages(ctx, q.getDLQTypeFromQueueType(), firstMessageID, lastMessageID)
	if err != nil {
		return &workflow.InternalServiceError{
			Message: fmt.Sprintf("RangeDeleteMessagesFromDLQ operation failed. Error %v", err),
		}
	}
	return nil
}

func (q *sqlQueue) UpdateDLQAckLevel(
	ctx context.Context,
	messageID int64,
	clusterName string,
) error {

	err := q.txExecute(ctx, "UpdateDLQAckLevel", func(tx sqlplugin.Tx) error {
		clusterAckLevels, err := tx.GetAckLevels(ctx, q.getDLQTypeFromQueueType(), true)
		if err != nil {
			return &workflow.InternalServiceError{
				Message: fmt.Sprintf("UpdateDLQAckLevel operation failed. Error %v", err),
			}
		}

		if clusterAckLevels == nil {
			err := tx.InsertAckLevel(ctx, q.getDLQTypeFromQueueType(), messageID, clusterName)
			if err != nil {
				return &workflow.InternalServiceError{
					Message: fmt.Sprintf("UpdateDLQAckLevel operation failed. Error %v", err),
				}
			}
			return nil
		}

		// Ignore possibly delayed message
		if clusterAckLevels[clusterName] > messageID {
			return nil
		}

		clusterAckLevels[clusterName] = messageID
		err = tx.UpdateAckLevels(ctx, q.getDLQTypeFromQueueType(), clusterAckLevels)
		if err != nil {
			return &workflow.InternalServiceError{
				Message: fmt.Sprintf("UpdateDLQAckLevel operation failed. Error %v", err),
			}
		}
		return nil
	})

	if err != nil {
		return &workflow.InternalServiceError{Message: err.Error()}
	}
	return nil
}

func (q *sqlQueue) GetDLQAckLevels(
	ctx context.Context,
) (map[string]int64, error) {

	return q.db.GetAckLevels(ctx, q.getDLQTypeFromQueueType(), false)
}

func (q *sqlQueue) GetDLQSize(
	ctx context.Context,
) (int64, error) {

	return q.db.GetQueueSize(ctx, q.getDLQTypeFromQueueType())
}

func (q *sqlQueue) getDLQTypeFromQueueType() persistence.QueueType {
	return -q.queueType
}
