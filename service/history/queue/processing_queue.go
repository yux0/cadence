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

package queue

import (
	"fmt"
	"sort"

	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/metrics"
	t "github.com/uber/cadence/common/task"
	"github.com/uber/cadence/service/history/task"
)

type (
	processingQueueStateImpl struct {
		level        int
		ackLevel     task.Key
		readLevel    task.Key
		maxLevel     task.Key
		domainFilter DomainFilter
	}

	processingQueueImpl struct {
		state            *processingQueueStateImpl
		outstandingTasks map[task.Key]task.Task

		logger        log.Logger
		metricsClient metrics.Client // TODO: emit metrics
	}
)

// NewProcessingQueueState creates a new state instance for processing queue
// readLevel will be set to the same value as ackLevel
func NewProcessingQueueState(
	level int,
	ackLevel task.Key,
	maxLevel task.Key,
	domainFilter DomainFilter,
) ProcessingQueueState {
	return newProcessingQueueState(
		level,
		ackLevel,
		ackLevel,
		maxLevel,
		domainFilter,
	)
}

func newProcessingQueueState(
	level int,
	ackLevel task.Key,
	readLevel task.Key,
	maxLevel task.Key,
	domainFilter DomainFilter,
) *processingQueueStateImpl {
	return &processingQueueStateImpl{
		level:        level,
		ackLevel:     ackLevel,
		readLevel:    readLevel,
		maxLevel:     maxLevel,
		domainFilter: domainFilter,
	}
}

// NewProcessingQueue creates a new processing queue based on its state
func NewProcessingQueue(
	state ProcessingQueueState,
	logger log.Logger,
	metricsClient metrics.Client,
) ProcessingQueue {
	return newProcessingQueue(
		state,
		nil,
		logger,
		metricsClient,
	)
}

func newProcessingQueue(
	state ProcessingQueueState,
	outstandingTasks map[task.Key]task.Task,
	logger log.Logger,
	metricsClient metrics.Client,
) *processingQueueImpl {
	if outstandingTasks == nil {
		outstandingTasks = make(map[task.Key]task.Task)
	}

	queue := &processingQueueImpl{
		outstandingTasks: outstandingTasks,
		logger:           logger,
		metricsClient:    metricsClient,
	}

	// convert state to *processingQueueStateImpl type so that
	// queue implementation can change the state value
	if stateImpl, ok := state.(*processingQueueStateImpl); ok {
		queue.state = stateImpl
	} else {
		queue.state = copyQueueState(state)
	}

	return queue
}

func (s *processingQueueStateImpl) Level() int {
	return s.level
}

func (s *processingQueueStateImpl) MaxLevel() task.Key {
	return s.maxLevel
}

func (s *processingQueueStateImpl) AckLevel() task.Key {
	return s.ackLevel
}

func (s *processingQueueStateImpl) ReadLevel() task.Key {
	return s.readLevel
}

func (s *processingQueueStateImpl) DomainFilter() DomainFilter {
	return s.domainFilter
}

func (s *processingQueueStateImpl) String() string {
	return fmt.Sprintf("&{level: %+v, ackLevel: %+v, readLevel: %+v, maxLevel: %+v, domainFilter: %+v}",
		s.level, s.ackLevel, s.readLevel, s.maxLevel, s.domainFilter,
	)
}

func (q *processingQueueImpl) State() ProcessingQueueState {
	return q.state
}

func (q *processingQueueImpl) Split(
	policy ProcessingQueueSplitPolicy,
) []ProcessingQueue {
	newQueueStates := policy.Evaluate(q)
	if len(newQueueStates) == 0 {
		// no need to split, return self
		return []ProcessingQueue{q}
	}

	return splitProcessingQueue([]*processingQueueImpl{q}, newQueueStates, q.logger, q.metricsClient)
}

func (q *processingQueueImpl) Merge(
	queue ProcessingQueue,
) []ProcessingQueue {
	q1, q2 := q, queue.(*processingQueueImpl)

	if q1.State().Level() != q2.State().Level() {
		errMsg := "Processing queue encountered a queue from different level during merge"
		q.logger.Error(errMsg, tag.Error(
			fmt.Errorf("current queue level: %v, incoming queue level: %v", q1.state.level, q2.state.level),
		))
		panic(errMsg)
	}

	if !q1.state.ackLevel.Less(q2.state.maxLevel) ||
		!q2.state.ackLevel.Less(q1.state.maxLevel) {
		// one queue's ackLevel is larger or equal than the other one's maxLevel
		// this means there's no overlap between two queues
		return []ProcessingQueue{q1, q2}
	}

	// generate new queue states for merged queues
	newQueueStates := []ProcessingQueueState{}
	if !taskKeyEquals(q1.state.ackLevel, q2.state.ackLevel) {
		if q2.state.ackLevel.Less(q1.state.ackLevel) {
			q1, q2 = q2, q1
		}

		newQueueStates = append(newQueueStates, newProcessingQueueState(
			q1.state.level,
			q1.state.ackLevel,
			minTaskKey(q1.state.readLevel, q2.state.ackLevel),
			q2.state.ackLevel,
			q1.state.domainFilter.copy(),
		))
	}

	if !taskKeyEquals(q1.state.maxLevel, q2.state.maxLevel) {
		if q1.state.maxLevel.Less(q2.state.maxLevel) {
			q1, q2 = q2, q1
		}

		newQueueStates = append(newQueueStates, newProcessingQueueState(
			q1.state.level,
			q2.state.maxLevel,
			maxTaskKey(q1.state.readLevel, q2.state.maxLevel),
			q1.state.maxLevel,
			q1.state.domainFilter.copy(),
		))
	}

	overlappingQueueAckLevel := maxTaskKey(q1.state.ackLevel, q2.state.ackLevel)
	newQueueStates = append(newQueueStates, newProcessingQueueState(
		q1.state.level,
		overlappingQueueAckLevel,
		maxTaskKey(minTaskKey(q1.state.readLevel, q2.state.readLevel), overlappingQueueAckLevel),
		minTaskKey(q1.state.maxLevel, q2.state.maxLevel),
		q1.state.domainFilter.Merge(q2.state.domainFilter),
	))

	return splitProcessingQueue([]*processingQueueImpl{q1, q2}, newQueueStates, q.logger, q.metricsClient)
}

func (q *processingQueueImpl) AddTasks(
	tasks map[task.Key]task.Task,
	newReadLevel task.Key,
) {
	for key, task := range tasks {
		if _, loaded := q.outstandingTasks[key]; loaded {
			q.logger.Debug(fmt.Sprintf("Skipping task: %+v. DomainID: %v, WorkflowID: %v, RunID: %v, Type: %v",
				key, task.GetDomainID(), task.GetWorkflowID(), task.GetRunID(), task.GetTaskType()))
			continue
		}

		if !taskBelongsToProcessQueue(q.state, key, task) {
			errMsg := "Processing queue encountered a task doesn't belong to its scope"
			q.logger.Error(errMsg, tag.Error(
				fmt.Errorf("Processing queue state: %+v, task: %+v", q.state, key),
			))
			panic(errMsg)
		}

		q.outstandingTasks[key] = task
	}

	q.state.readLevel = newReadLevel
}

func (q *processingQueueImpl) UpdateAckLevel() {
	keys := make([]task.Key, 0, len(q.outstandingTasks))
	for key := range q.outstandingTasks {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Less(keys[j])
	})

	for _, key := range keys {
		if q.outstandingTasks[key].State() != t.TaskStateAcked {
			break
		}

		q.state.ackLevel = key
		delete(q.outstandingTasks, key)
	}

	if len(q.outstandingTasks) == 0 {
		q.state.ackLevel = q.state.readLevel
	}

	if timerKey, ok := q.state.ackLevel.(timerTaskKey); ok {
		q.state.ackLevel = newTimerTaskKey(timerKey.visibilityTimestamp, 0)
	}
}

func splitProcessingQueue(
	queues []*processingQueueImpl,
	newQueueStates []ProcessingQueueState,
	logger log.Logger,
	metricsClient metrics.Client,
) []ProcessingQueue {
	newQueueTasks := make([]map[task.Key]task.Task, 0, len(newQueueStates))
	for i := 0; i != len(newQueueStates); i++ {
		newQueueTasks = append(newQueueTasks, make(map[task.Key]task.Task))
	}

	for _, queue := range queues {
	SplitTaskLoop:
		for key, task := range queue.outstandingTasks {
			for i, state := range newQueueStates {
				if taskBelongsToProcessQueue(state, key, task) {
					newQueueTasks[i][key] = task
					continue SplitTaskLoop
				}
			}

			// if code reaches there it means the task doesn't belongs to any new queue.
			// there's must be a bug in the code for generating the newQueueStates
			// log error, skip the split and return current queues as result
			currentQueues := make([]ProcessingQueue, 0, len(newQueueStates))
			currentQueueStates := make([]ProcessingQueueState, 0, len(newQueueStates))
			for _, q := range queues {
				currentQueues = append(currentQueues, q)
				currentQueueStates = append(currentQueueStates, queue.State())
			}
			logger.Error("Processing queue encountered an error during split or merge.", tag.Error(
				fmt.Errorf("current queue state: %+v, new queue state: %+v", currentQueueStates, newQueueStates),
			))
			return currentQueues
		}
	}

	newQueues := make([]ProcessingQueue, 0, len(newQueueStates))
	for i, state := range newQueueStates {
		queue := newProcessingQueue(
			state,
			newQueueTasks[i],
			logger,
			metricsClient,
		)
		newQueues = append(newQueues, queue)
	}

	return newQueues
}

func taskBelongsToProcessQueue(
	state ProcessingQueueState,
	key task.Key,
	task task.Task,
) bool {
	return state.DomainFilter().Filter(task.GetDomainID()) &&
		state.AckLevel().Less(key) &&
		!state.MaxLevel().Less(key)
}

func taskKeyEquals(
	key1 task.Key,
	key2 task.Key,
) bool {
	return !key1.Less(key2) && !key2.Less(key1)
}

func minTaskKey(
	key1 task.Key,
	key2 task.Key,
) task.Key {
	if key1.Less(key2) {
		return key1
	}
	return key2
}

func maxTaskKey(
	key1 task.Key,
	key2 task.Key,
) task.Key {
	if key1.Less(key2) {
		return key2
	}
	return key1
}

func copyQueueState(
	state ProcessingQueueState,
) *processingQueueStateImpl {
	return newProcessingQueueState(
		state.Level(),
		state.AckLevel(),
		state.ReadLevel(),
		state.MaxLevel(),
		state.DomainFilter(),
	)
}
