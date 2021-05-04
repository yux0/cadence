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

//go:generate mockgen -package $GOPACKAGE -source $GOFILE -destination state_rebuilder_mock.go -self_package github.com/uber/cadence/service/history/execution

package execution

import (
	"context"
	ctx "context"
	"fmt"
	"time"

	"github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/cluster"
	"github.com/uber/cadence/common/collection"
	"github.com/uber/cadence/common/definition"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/service/history/shard"
)

const (
	// NDCDefaultPageSize is the default pagination size for ndc
	NDCDefaultPageSize = 100
)

type (
	// StateRebuilder is a mutable state builder to ndc state rebuild
	StateRebuilder interface {
		Rebuild(
			ctx ctx.Context,
			now time.Time,
			baseWorkflowIdentifier definition.WorkflowIdentifier,
			baseBranchToken []byte,
			baseLastEventID int64,
			baseLastEventVersion int64,
			targetWorkflowIdentifier definition.WorkflowIdentifier,
			targetBranchToken []byte,
			requestID string,
		) (MutableState, int64, error)
	}

	stateRebuilderImpl struct {
		shard           shard.Context
		domainCache     cache.DomainCache
		clusterMetadata cluster.Metadata
		historyV2Mgr    persistence.HistoryManager
		taskRefresher   MutableStateTaskRefresher

		rebuiltHistorySize int64
		logger             log.Logger
	}
)

var _ StateRebuilder = (*stateRebuilderImpl)(nil)

// NewStateRebuilder creates a state rebuilder
func NewStateRebuilder(
	shard shard.Context,
	logger log.Logger,
) StateRebuilder {

	return &stateRebuilderImpl{
		shard:           shard,
		domainCache:     shard.GetDomainCache(),
		clusterMetadata: shard.GetService().GetClusterMetadata(),
		historyV2Mgr:    shard.GetHistoryManager(),
		taskRefresher: NewMutableStateTaskRefresher(
			shard.GetConfig(),
			shard.GetDomainCache(),
			shard.GetEventsCache(),
			logger,
			shard.GetShardID(),
		),
		rebuiltHistorySize: 0,
		logger:             logger,
	}
}

func (r *stateRebuilderImpl) Rebuild(
	ctx ctx.Context,
	now time.Time,
	baseWorkflowIdentifier definition.WorkflowIdentifier,
	baseBranchToken []byte,
	baseLastEventID int64,
	baseLastEventVersion int64,
	targetWorkflowIdentifier definition.WorkflowIdentifier,
	targetBranchToken []byte,
	requestID string,
) (MutableState, int64, error) {

	iter := collection.NewPagingIterator(r.getPaginationFn(
		ctx,
		baseWorkflowIdentifier,
		common.FirstEventID,
		baseLastEventID+1,
		baseBranchToken,
	))

	domainEntry, err := r.domainCache.GetDomainByID(targetWorkflowIdentifier.DomainID)
	if err != nil {
		return nil, 0, err
	}

	// need to specially handling the first batch, to initialize mutable state & state builder
	batch, err := iter.Next()
	if err != nil {
		return nil, 0, err
	}
	firstEventBatch := batch.(*shared.History).Events
	rebuiltMutableState, stateBuilder := r.initializeBuilders(
		domainEntry,
	)
	if err := r.applyEvents(targetWorkflowIdentifier, stateBuilder, firstEventBatch, requestID); err != nil {
		return nil, 0, err
	}

	for iter.HasNext() {
		batch, err := iter.Next()
		if err != nil {
			return nil, 0, err
		}
		events := batch.(*shared.History).Events
		if err := r.applyEvents(targetWorkflowIdentifier, stateBuilder, events, requestID); err != nil {
			return nil, 0, err
		}
	}

	if err := rebuiltMutableState.SetCurrentBranchToken(targetBranchToken); err != nil {
		return nil, 0, err
	}
	rebuildVersionHistories := rebuiltMutableState.GetVersionHistories()
	if rebuildVersionHistories != nil {
		currentVersionHistory, err := rebuildVersionHistories.GetCurrentVersionHistory()
		if err != nil {
			return nil, 0, err
		}
		lastItem, err := currentVersionHistory.GetLastItem()
		if err != nil {
			return nil, 0, err
		}
		if !lastItem.Equals(persistence.NewVersionHistoryItem(
			baseLastEventID,
			baseLastEventVersion,
		)) {
			return nil, 0, &shared.BadRequestError{Message: fmt.Sprintf(
				"nDCStateRebuilder unable to rebuild mutable state to event ID: %v, version: %v, "+
					"baseLastEventID + baseLastEventVersion is not the same as the last event of the last "+
					"batch, event ID: %v, version :%v ,typicaly because of attemptting to rebuild to a middle of a batch",
				baseLastEventID,
				baseLastEventVersion,
				lastItem.EventID,
				lastItem.Version,
			)}
		}
	}

	// close rebuilt mutable state transaction clearing all generated tasks, etc.
	_, _, err = rebuiltMutableState.CloseTransactionAsSnapshot(now, TransactionPolicyPassive)
	if err != nil {
		return nil, 0, err
	}

	// refresh tasks to be generated
	if err := r.taskRefresher.RefreshTasks(ctx, now, rebuiltMutableState); err != nil {
		return nil, 0, err
	}

	// mutable state rebuild should use the same time stamp
	rebuiltMutableState.GetExecutionInfo().StartTimestamp = now
	return rebuiltMutableState, r.rebuiltHistorySize, nil
}

func (r *stateRebuilderImpl) initializeBuilders(
	domainEntry *cache.DomainCacheEntry,
) (MutableState, StateBuilder) {
	resetMutableStateBuilder := NewMutableStateBuilderWithVersionHistories(
		r.shard,
		r.logger,
		domainEntry,
	)
	stateBuilder := NewStateBuilder(
		r.shard,
		r.logger,
		resetMutableStateBuilder,
		func(mutableState MutableState) MutableStateTaskGenerator {
			return NewMutableStateTaskGenerator(r.shard.GetDomainCache(), r.logger, mutableState)
		},
	)
	return resetMutableStateBuilder, stateBuilder
}

func (r *stateRebuilderImpl) applyEvents(
	workflowIdentifier definition.WorkflowIdentifier,
	stateBuilder StateBuilder,
	events []*shared.HistoryEvent,
	requestID string,
) error {

	_, err := stateBuilder.ApplyEvents(
		workflowIdentifier.DomainID,
		requestID,
		shared.WorkflowExecution{
			WorkflowId: common.StringPtr(workflowIdentifier.WorkflowID),
			RunId:      common.StringPtr(workflowIdentifier.RunID),
		},
		events,
		nil, // no new run history when rebuilding mutable state
	)
	if err != nil {
		r.logger.Error("nDCStateRebuilder unable to rebuild mutable state.", tag.Error(err))
		return err
	}
	return nil
}

func (r *stateRebuilderImpl) getPaginationFn(
	ctx context.Context,
	workflowIdentifier definition.WorkflowIdentifier,
	firstEventID int64,
	nextEventID int64,
	branchToken []byte,
) collection.PaginationFn {

	return func(paginationToken []byte) ([]interface{}, []byte, error) {

		_, historyBatches, token, size, err := persistence.PaginateHistory(
			ctx,
			r.historyV2Mgr,
			true,
			branchToken,
			firstEventID,
			nextEventID,
			paginationToken,
			NDCDefaultPageSize,
			common.IntPtr(r.shard.GetShardID()),
		)
		if err != nil {
			return nil, nil, err
		}
		r.rebuiltHistorySize += int64(size)

		var paginateItems []interface{}
		for _, history := range historyBatches {
			paginateItems = append(paginateItems, history)
		}
		return paginateItems, token, nil
	}
}
