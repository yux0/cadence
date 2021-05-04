// Copyright (c) 2017-2021 Uber Technologies Inc.

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package queue

import (
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/uber/cadence/common/clock"
	"github.com/uber/cadence/common/dynamicconfig"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/metrics/mocks"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/service/history/config"
	"github.com/uber/cadence/service/history/shard"
	"github.com/uber/cadence/service/history/task"
)

type (
	transferQueueValidatorSuite struct {
		suite.Suite
		*require.Assertions

		controller      *gomock.Controller
		mockShard       *shard.TestContext
		mockLogger      *log.MockLogger
		mockMetricScope *mocks.Scope

		timeSource clock.TimeSource
		processor  *transferQueueProcessorBase
		validator  *transferQueueValidator
	}
)

const (
	testValidationInterval = 100 * time.Millisecond
)

func TestTransferQueueValidatorSuite(t *testing.T) {
	s := new(transferQueueValidatorSuite)
	suite.Run(t, s)
}

func (s *transferQueueValidatorSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	s.controller = gomock.NewController(s.T())
	s.mockShard = shard.NewTestContext(
		s.controller,
		&persistence.ShardInfo{
			RangeID:          1,
			TransferAckLevel: 0,
		},
		config.NewForTest(),
	)
	s.mockLogger = &log.MockLogger{}
	s.mockMetricScope = &mocks.Scope{}

	s.timeSource = clock.NewRealTimeSource()
	s.processor = &transferQueueProcessorBase{
		processorBase: &processorBase{
			processingQueueCollections: newProcessingQueueCollections(
				[]ProcessingQueueState{
					NewProcessingQueueState(
						defaultProcessingQueueLevel,
						newTransferTaskKey(0),
						maximumTransferTaskKey,
						NewDomainFilter(map[string]struct{}{}, true),
					),
				},
				nil,
				nil,
			),
		},
	}
	s.validator = newTransferQueueValidator(
		s.processor,
		s.timeSource,
		dynamicconfig.GetDurationPropertyFn(testValidationInterval),
		s.mockLogger,
		s.mockMetricScope,
	)
}

func (s *transferQueueValidatorSuite) TearDownTest() {
	s.controller.Finish()
	s.mockLogger.AssertExpectations(s.T())
	s.mockMetricScope.AssertExpectations(s.T())
}

func (s *transferQueueValidatorSuite) TestAddTasks_NoTaskDropped() {
	executionInfo := &persistence.WorkflowExecutionInfo{}
	tasks := []persistence.Task{
		&persistence.DecisionTask{TaskID: 0},
		&persistence.DecisionTask{TaskID: 1},
		&persistence.DecisionTask{TaskID: 2},
	}
	expectedPendingTasksLen := len(tasks)

	s.validator.addTasks(executionInfo, tasks)
	s.Len(s.validator.pendingTaskInfos, expectedPendingTasksLen)

	tasks = []persistence.Task{
		&persistence.DecisionTask{TaskID: 4},
		&persistence.DecisionTask{TaskID: 5},
	}
	expectedPendingTasksLen += len(tasks)

	s.validator.addTasks(executionInfo, tasks)
	s.Len(s.validator.pendingTaskInfos, expectedPendingTasksLen)
}

func (s *transferQueueValidatorSuite) TestAddTasks_TaskDropped() {
	executionInfo := &persistence.WorkflowExecutionInfo{}
	tasks := []persistence.Task{
		&persistence.DecisionTask{TaskID: 0},
		&persistence.DecisionTask{TaskID: 1},
		&persistence.DecisionTask{TaskID: 2},
	}
	expectedPendingTasksLen := len(tasks)

	s.validator.addTasks(executionInfo, tasks)
	s.Len(s.validator.pendingTaskInfos, expectedPendingTasksLen)

	tasks = []persistence.Task{}
	for i := 0; i != defaultMaxPendingTasksSize; i++ {
		tasks = append(tasks, &persistence.DecisionTask{TaskID: int64(i + expectedPendingTasksLen)})
	}

	numDroppedTasks := expectedPendingTasksLen + len(tasks) - defaultMaxPendingTasksSize
	s.mockLogger.On("Warn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Times(1)
	s.mockMetricScope.On("AddCounter", metrics.QueueValidatorDropTaskCounter, int64(numDroppedTasks)).Times(1)

	s.validator.addTasks(executionInfo, tasks)
	s.Len(s.validator.pendingTaskInfos, defaultMaxPendingTasksSize)
}

func (s *transferQueueValidatorSuite) TestAckTasks_NoTaskLost() {
	executionInfo := &persistence.WorkflowExecutionInfo{}
	pendingTasks := []persistence.Task{
		&persistence.DecisionTask{TaskID: 0},
		&persistence.DecisionTask{TaskID: 1},
		&persistence.DecisionTask{TaskID: 2},
		&persistence.DecisionTask{TaskID: 100},
	}
	s.validator.addTasks(executionInfo, pendingTasks)

	loadedTasks := make(map[task.Key]task.Task, len(pendingTasks))
	for _, pendingTask := range pendingTasks[:len(pendingTasks)-1] {
		loadedTasks[newTransferTaskKey(pendingTask.GetTaskID())] = task.NewTransferTask(
			s.mockShard,
			&persistence.TransferTaskInfo{
				TaskID: pendingTask.GetTaskID(),
			},
			task.QueueTypeActiveTransfer,
			nil, nil, nil, nil, nil,
			s.timeSource,
			nil,
		)
	}

	time.Sleep(testValidationInterval)
	s.mockMetricScope.On("IncCounter", metrics.QueueValidatorValidationCounter).Times(1)

	readLevel := newTransferTaskKey(0)
	maxReadLevel := newTransferTaskKey(10)
	s.processor.processingQueueCollections[0].ActiveQueue().State().(*processingQueueStateImpl).readLevel = maxReadLevel
	s.validator.ackTasks(defaultProcessingQueueLevel, readLevel, maxReadLevel, loadedTasks)
	s.Len(s.validator.pendingTaskInfos, 1)
}

func (s *transferQueueValidatorSuite) TestAckTasks_TaskLost() {
	executionInfo := &persistence.WorkflowExecutionInfo{}
	pendingTasks := []persistence.Task{
		&persistence.DecisionTask{TaskID: 0},
		&persistence.DecisionTask{TaskID: 1},
		&persistence.DecisionTask{TaskID: 2},
	}
	s.validator.addTasks(executionInfo, pendingTasks)

	loadedTasks := make(map[task.Key]task.Task, len(pendingTasks))
	for _, pendingTask := range pendingTasks[1:] {
		loadedTasks[newTransferTaskKey(pendingTask.GetTaskID())] = task.NewTransferTask(
			s.mockShard,
			&persistence.TransferTaskInfo{
				TaskID: pendingTask.GetTaskID(),
			},
			task.QueueTypeActiveTransfer,
			nil, nil, nil, nil, nil,
			s.timeSource,
			nil,
		)
	}

	time.Sleep(testValidationInterval)
	s.mockLogger.On("Error", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Times(1)
	s.mockMetricScope.On("IncCounter", metrics.QueueValidatorValidationCounter).Times(1)
	s.mockMetricScope.On("IncCounter", metrics.QueueValidatorLostTaskCounter).Times(1)

	readLevel := newTransferTaskKey(0)
	maxReadLevel := newTransferTaskKey(10)
	s.processor.processingQueueCollections[0].ActiveQueue().State().(*processingQueueStateImpl).readLevel = maxReadLevel
	s.validator.ackTasks(defaultProcessingQueueLevel, readLevel, maxReadLevel, loadedTasks)
	s.Empty(s.validator.pendingTaskInfos)
}

func (s *transferQueueValidatorSuite) TestAckTasks_LostRequestNotContinuous() {
	readLevel := newTransferTaskKey(0)
	maxReadLevel := newTransferTaskKey(10)
	s.validator.ackTasks(defaultProcessingQueueLevel, readLevel, maxReadLevel, nil)

	readLevel = newTransferTaskKey(10)
	maxReadLevel = newTransferTaskKey(15)
	s.validator.ackTasks(defaultProcessingQueueLevel, readLevel, maxReadLevel, nil)

	s.mockLogger.On("Error", mock.Anything, mock.Anything).Times(1)
	s.mockMetricScope.On("IncCounter", metrics.QueueValidatorInvalidLoadCounter).Times(1)

	readLevel = newTransferTaskKey(16)
	maxReadLevel = newTransferTaskKey(25)
	s.validator.ackTasks(defaultProcessingQueueLevel, readLevel, maxReadLevel, nil)
}
