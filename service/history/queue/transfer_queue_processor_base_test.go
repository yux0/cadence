// Copyright (c) 2017-2020 Uber Technologies Inc.

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uber-go/tally"

	"github.com/uber/cadence/common/collection"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/loggerimpl"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/service/history/config"
	"github.com/uber/cadence/service/history/shard"
	"github.com/uber/cadence/service/history/task"
)

type (
	transferQueueProcessorBaseSuite struct {
		suite.Suite
		*require.Assertions

		controller           *gomock.Controller
		mockShard            *shard.TestContext
		mockTaskProcessor    *task.MockProcessor
		mockQueueSplitPolicy *MockProcessingQueueSplitPolicy

		redispatchQueue collection.Queue
		logger          log.Logger
		metricsClient   metrics.Client
		metricsScope    metrics.Scope
	}
)

func TestTransferQueueProcessorBaseSuite(t *testing.T) {
	s := new(transferQueueProcessorBaseSuite)
	suite.Run(t, s)
}

func (s *transferQueueProcessorBaseSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	s.controller = gomock.NewController(s.T())
	s.mockShard = shard.NewTestContext(
		s.controller,
		&persistence.ShardInfo{
			ShardID:          10,
			RangeID:          1,
			TransferAckLevel: 0,
		},
		config.NewForTest(),
	)
	s.mockQueueSplitPolicy = NewMockProcessingQueueSplitPolicy(s.controller)
	s.mockTaskProcessor = task.NewMockProcessor(s.controller)

	s.redispatchQueue = collection.NewConcurrentQueue()
	s.logger = loggerimpl.NewDevelopmentForTest(s.Suite)
	s.metricsClient = metrics.NewClient(tally.NoopScope, metrics.History)
	s.metricsScope = s.metricsClient.Scope(metrics.TransferQueueProcessorScope)
}

func (s *transferQueueProcessorBaseSuite) TearDownTest() {
	s.controller.Finish()
	s.mockShard.Finish(s.T())
}

func (s *transferQueueProcessorBaseSuite) TestUpdateAckLevel_ProcessedFinished() {
	processingQueueStates := []ProcessingQueueState{
		NewProcessingQueueState(
			2,
			newTransferTaskKey(100),
			newTransferTaskKey(100),
			NewDomainFilter(map[string]struct{}{"testDomain1": {}}, false),
		),
		NewProcessingQueueState(
			0,
			newTransferTaskKey(1000),
			newTransferTaskKey(1000),
			NewDomainFilter(map[string]struct{}{"testDomain1": {}, "testDomain2": {}}, true),
		),
	}
	queueShutdown := false
	queueShutdownFn := func() error {
		queueShutdown = true
		return nil
	}

	processorBase := s.newTestTransferQueueProcessBase(
		processingQueueStates,
		nil,
		nil,
		queueShutdownFn,
		nil,
	)

	processFinished, err := processorBase.updateAckLevel()
	s.NoError(err)
	s.True(processFinished)
	s.True(queueShutdown)
}

func (s *transferQueueProcessorBaseSuite) TestUpdateAckLevel_ProcessNotFinished() {
	processingQueueStates := []ProcessingQueueState{
		NewProcessingQueueState(
			2,
			newTransferTaskKey(5),
			newTransferTaskKey(100),
			NewDomainFilter(map[string]struct{}{"testDomain1": {}}, false),
		),
		NewProcessingQueueState(
			1,
			newTransferTaskKey(2),
			newTransferTaskKey(100),
			NewDomainFilter(map[string]struct{}{"testDomain1": {}}, false),
		),
		NewProcessingQueueState(
			0,
			newTransferTaskKey(100),
			newTransferTaskKey(1000),
			NewDomainFilter(map[string]struct{}{"testDomain1": {}, "testDomain2": {}}, true),
		),
	}
	updateAckLevel := int64(0)
	updateTransferAckLevelFn := func(ackLevel task.Key) error {
		updateAckLevel = ackLevel.(transferTaskKey).taskID
		return nil
	}

	processorBase := s.newTestTransferQueueProcessBase(
		processingQueueStates,
		nil,
		updateTransferAckLevelFn,
		nil,
		nil,
	)

	processFinished, err := processorBase.updateAckLevel()
	s.NoError(err)
	s.False(processFinished)
	s.Equal(int64(2), updateAckLevel)
}

func (s *transferQueueProcessorBaseSuite) TestProcessBatch_NoNextPage_FullRead() {
	ackLevel := newTransferTaskKey(0)
	maxLevel := newTransferTaskKey(1000)
	processingQueueStates := []ProcessingQueueState{
		NewProcessingQueueState(
			0,
			ackLevel,
			maxLevel,
			NewDomainFilter(map[string]struct{}{"testDomain1": {}}, false),
		),
		NewProcessingQueueState(
			0,
			newTransferTaskKey(1000),
			newTransferTaskKey(10000),
			NewDomainFilter(map[string]struct{}{"testDomain1": {}}, false),
		),
	}
	taskInitializer := func(taskInfo task.Info) task.Task {
		return task.NewTransferTask(s.mockShard, taskInfo, task.QueueTypeActiveTransfer, s.metricsScope, nil, nil, nil, nil, s.mockShard.GetTimeSource(), nil, nil)
	}
	updateMaxReadLevel := func() task.Key {
		return newTransferTaskKey(10000)
	}
	taskInfos := []*persistence.TransferTaskInfo{
		{
			TaskID:   1,
			DomainID: "testDomain1",
		},
		{
			TaskID:   10,
			DomainID: "testDomain2",
		},
		{
			TaskID:   100,
			DomainID: "testDomain1",
		},
	}
	mockExecutionManager := s.mockShard.Resource.ExecutionMgr
	mockExecutionManager.On("GetTransferTasks", &persistence.GetTransferTasksRequest{
		ReadLevel:    ackLevel.(transferTaskKey).taskID,
		MaxReadLevel: maxLevel.(transferTaskKey).taskID,
		BatchSize:    s.mockShard.GetConfig().TransferTaskBatchSize(),
	}).Return(&persistence.GetTransferTasksResponse{
		Tasks:         taskInfos,
		NextPageToken: nil,
	}, nil).Once()

	s.mockTaskProcessor.EXPECT().TrySubmit(gomock.Any()).Return(true, nil).AnyTimes()

	processorBase := s.newTestTransferQueueProcessBase(
		processingQueueStates,
		updateMaxReadLevel,
		nil,
		nil,
		taskInitializer,
	)

	processorBase.processBatch()

	queueCollection := processorBase.processingQueueCollections[0]
	s.NotNil(queueCollection.ActiveQueue())
	s.True(taskKeyEquals(maxLevel, queueCollection.Queues()[0].State().ReadLevel()))

	newTasks := false
	select {
	case <-processorBase.notifyCh:
		newTasks = true
	default:
	}

	s.True(newTasks)
}

func (s *transferQueueProcessorBaseSuite) TestProcessBatch_NoNextPage_PartialRead() {
	ackLevel := newTransferTaskKey(0)
	maxLevel := newTransferTaskKey(1000)
	shardMaxLevel := newTransferTaskKey(500)
	processingQueueStates := []ProcessingQueueState{
		NewProcessingQueueState(
			0,
			ackLevel,
			maxLevel,
			NewDomainFilter(map[string]struct{}{"testDomain1": {}}, false),
		),
	}
	taskInitializer := func(taskInfo task.Info) task.Task {
		return task.NewTransferTask(s.mockShard, taskInfo, task.QueueTypeActiveTransfer, s.metricsScope, nil, nil, nil, nil, s.mockShard.GetTimeSource(), nil, nil)
	}
	updateMaxReadLevel := func() task.Key {
		return shardMaxLevel
	}
	taskInfos := []*persistence.TransferTaskInfo{
		{
			TaskID:   1,
			DomainID: "testDomain1",
		},
		{
			TaskID:   10,
			DomainID: "testDomain2",
		},
		{
			TaskID:   100,
			DomainID: "testDomain1",
		},
	}
	mockExecutionManager := s.mockShard.Resource.ExecutionMgr
	mockExecutionManager.On("GetTransferTasks", &persistence.GetTransferTasksRequest{
		ReadLevel:    ackLevel.(transferTaskKey).taskID,
		MaxReadLevel: shardMaxLevel.(transferTaskKey).taskID,
		BatchSize:    s.mockShard.GetConfig().TransferTaskBatchSize(),
	}).Return(&persistence.GetTransferTasksResponse{
		Tasks:         taskInfos,
		NextPageToken: nil,
	}, nil).Once()

	s.mockTaskProcessor.EXPECT().TrySubmit(gomock.Any()).Return(true, nil).AnyTimes()

	processorBase := s.newTestTransferQueueProcessBase(
		processingQueueStates,
		updateMaxReadLevel,
		nil,
		nil,
		taskInitializer,
	)

	processorBase.processBatch()

	queueCollection := processorBase.processingQueueCollections[0]
	s.NotNil(queueCollection.ActiveQueue())
	s.True(taskKeyEquals(shardMaxLevel, queueCollection.Queues()[0].State().ReadLevel()))

	newTasks := false
	select {
	case <-processorBase.notifyCh:
		newTasks = true
	default:
	}

	s.False(newTasks)
}

func (s *transferQueueProcessorBaseSuite) TestProcessBatch_WithNextPage() {
	ackLevel := newTransferTaskKey(0)
	maxLevel := newTransferTaskKey(1000)
	processingQueueStates := []ProcessingQueueState{
		NewProcessingQueueState(
			0,
			ackLevel,
			maxLevel,
			NewDomainFilter(map[string]struct{}{"testDomain1": {}}, false),
		),
	}
	taskInitializer := func(taskInfo task.Info) task.Task {
		return task.NewTransferTask(s.mockShard, taskInfo, task.QueueTypeActiveTransfer, s.metricsScope, nil, nil, nil, nil, s.mockShard.GetTimeSource(), nil, nil)
	}
	updateMaxReadLevel := func() task.Key {
		return newTransferTaskKey(10000)
	}
	taskInfos := []*persistence.TransferTaskInfo{
		{
			TaskID:   1,
			DomainID: "testDomain1",
		},
		{
			TaskID:   10,
			DomainID: "testDomain2",
		},
		{
			TaskID:   100,
			DomainID: "testDomain1",
		},
		{
			TaskID:   500,
			DomainID: "testDomain2",
		},
	}
	mockExecutionManager := s.mockShard.Resource.ExecutionMgr
	mockExecutionManager.On("GetTransferTasks", &persistence.GetTransferTasksRequest{
		ReadLevel:    ackLevel.(transferTaskKey).taskID,
		MaxReadLevel: maxLevel.(transferTaskKey).taskID,
		BatchSize:    s.mockShard.GetConfig().TransferTaskBatchSize(),
	}).Return(&persistence.GetTransferTasksResponse{
		Tasks:         taskInfos,
		NextPageToken: []byte{1, 2, 3},
	}, nil).Once()

	s.mockTaskProcessor.EXPECT().TrySubmit(gomock.Any()).Return(true, nil).AnyTimes()

	processorBase := s.newTestTransferQueueProcessBase(
		processingQueueStates,
		updateMaxReadLevel,
		nil,
		nil,
		taskInitializer,
	)

	processorBase.processBatch()

	queueCollection := processorBase.processingQueueCollections[0]
	s.NotNil(queueCollection.ActiveQueue())
	s.True(taskKeyEquals(newTransferTaskKey(500), queueCollection.Queues()[0].State().ReadLevel()))

	newTasks := false
	select {
	case <-processorBase.notifyCh:
		newTasks = true
	default:
	}

	s.True(newTasks)
}

func (s *transferQueueProcessorBaseSuite) TestReadTasks_NoNextPage() {
	readLevel := newTransferTaskKey(3)
	maxReadLevel := newTransferTaskKey(100)

	mockExecutionManager := s.mockShard.Resource.ExecutionMgr
	getTransferTaskResponse := &persistence.GetTransferTasksResponse{
		Tasks:         []*persistence.TransferTaskInfo{{}, {}, {}},
		NextPageToken: nil,
	}
	mockExecutionManager.On("GetTransferTasks", &persistence.GetTransferTasksRequest{
		ReadLevel:    readLevel.(transferTaskKey).taskID,
		MaxReadLevel: maxReadLevel.(transferTaskKey).taskID,
		BatchSize:    s.mockShard.GetConfig().TransferTaskBatchSize(),
	}).Return(getTransferTaskResponse, nil).Once()

	processorBase := s.newTestTransferQueueProcessBase(
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	tasks, more, err := processorBase.readTasks(readLevel, maxReadLevel)
	s.NoError(err)
	s.Len(tasks, len(getTransferTaskResponse.Tasks))
	s.False(more)
}

func (s *transferQueueProcessorBaseSuite) TestReadTasks_WithNextPage() {
	readLevel := newTransferTaskKey(3)
	maxReadLevel := newTransferTaskKey(10)

	mockExecutionManager := s.mockShard.Resource.ExecutionMgr
	getTransferTaskResponse := &persistence.GetTransferTasksResponse{
		Tasks:         []*persistence.TransferTaskInfo{{}, {}, {}},
		NextPageToken: []byte{1, 2, 3},
	}
	mockExecutionManager.On("GetTransferTasks", &persistence.GetTransferTasksRequest{
		ReadLevel:    readLevel.(transferTaskKey).taskID,
		MaxReadLevel: maxReadLevel.(transferTaskKey).taskID,
		BatchSize:    s.mockShard.GetConfig().TransferTaskBatchSize(),
	}).Return(getTransferTaskResponse, nil).Once()

	processorBase := s.newTestTransferQueueProcessBase(
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	tasks, more, err := processorBase.readTasks(readLevel, maxReadLevel)
	s.NoError(err)
	s.Len(tasks, len(getTransferTaskResponse.Tasks))
	s.True(more)
}

func (s *transferQueueProcessorBaseSuite) newTestTransferQueueProcessBase(
	processingQueueStates []ProcessingQueueState,
	maxReadLevel updateMaxReadLevelFn,
	updateTransferAckLevel updateClusterAckLevelFn,
	transferQueueShutdown queueShutdownFn,
	taskInitializer task.Initializer,
) *transferQueueProcessorBase {
	return newTransferQueueProcessorBase(
		s.mockShard,
		processingQueueStates,
		s.mockTaskProcessor,
		s.redispatchQueue,
		newTransferQueueProcessorOptions(s.mockShard.GetConfig(), true, false),
		maxReadLevel,
		updateTransferAckLevel,
		transferQueueShutdown,
		taskInitializer,
		s.logger,
		s.metricsClient,
	)
}
