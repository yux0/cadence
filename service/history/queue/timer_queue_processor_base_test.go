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
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uber-go/tally"

	"github.com/uber/cadence/common/cluster"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/loggerimpl"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/service/history/config"
	"github.com/uber/cadence/service/history/constants"
	"github.com/uber/cadence/service/history/shard"
	"github.com/uber/cadence/service/history/task"
)

type (
	timerQueueProcessorBaseSuite struct {
		suite.Suite
		*require.Assertions

		controller           *gomock.Controller
		mockShard            *shard.TestContext
		mockTaskProcessor    *task.MockProcessor
		mockQueueSplitPolicy *MockProcessingQueueSplitPolicy

		clusterName   string
		logger        log.Logger
		metricsClient metrics.Client
		metricsScope  metrics.Scope
	}
)

func TestTimerQueueProcessorBaseSuite(t *testing.T) {
	s := new(timerQueueProcessorBaseSuite)
	suite.Run(t, s)
}

func (s *timerQueueProcessorBaseSuite) SetupTest() {
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
	s.mockShard.Resource.DomainCache.EXPECT().GetDomainName(gomock.Any()).Return(constants.TestDomainName, nil).AnyTimes()
	s.mockQueueSplitPolicy = NewMockProcessingQueueSplitPolicy(s.controller)
	s.mockTaskProcessor = task.NewMockProcessor(s.controller)

	s.clusterName = cluster.TestCurrentClusterName
	s.logger = loggerimpl.NewDevelopmentForTest(s.Suite)
	s.metricsClient = metrics.NewClient(tally.NoopScope, metrics.History)
	s.metricsScope = s.metricsClient.Scope(metrics.TimerQueueProcessorScope)
}

func (s *timerQueueProcessorBaseSuite) TearDownTest() {
	s.controller.Finish()
	s.mockShard.Finish(s.T())
}

func (s *timerQueueProcessorBaseSuite) TestIsProcessNow() {
	mockClusterMetadata := s.mockShard.Resource.ClusterMetadata
	mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(s.clusterName).AnyTimes()

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(nil, nil, nil, nil, nil)
	s.True(timerQueueProcessBase.isProcessNow(time.Time{}))

	now := s.mockShard.GetCurrentTime(s.clusterName)
	s.True(timerQueueProcessBase.isProcessNow(now))

	timeBefore := now.Add(-10 * time.Second)
	s.True(timerQueueProcessBase.isProcessNow(timeBefore))

	timeAfter := now.Add(10 * time.Second)
	s.False(timerQueueProcessBase.isProcessNow(timeAfter))
}

func (s *timerQueueProcessorBaseSuite) TestGetTimerTasks_More() {
	readLevel := newTimerTaskKey(time.Now().Add(-10*time.Second), 0)
	maxReadLevel := newTimerTaskKey(time.Now().Add(10*time.Second), 0)
	batchSize := 10

	request := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  readLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  maxReadLevel.(timerTaskKey).visibilityTimestamp,
		BatchSize:     batchSize,
		NextPageToken: []byte("some random input next page token"),
	}

	response := &persistence.GetTimerIndexTasksResponse{
		Timers: []*persistence.TimerTaskInfo{
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: time.Now().Add(-5 * time.Second),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
		},
		NextPageToken: []byte("some random output next page token"),
	}

	mockExecutionMgr := s.mockShard.Resource.ExecutionMgr
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, request).Return(response, nil).Once()

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(nil, nil, nil, nil, nil)
	timers, token, err := timerQueueProcessBase.getTimerTasks(readLevel, maxReadLevel, request.NextPageToken, batchSize)
	s.Nil(err)
	s.Equal(response.Timers, timers)
	s.Equal(response.NextPageToken, token)
}

func (s *timerQueueProcessorBaseSuite) TestGetTimerTasks_NoMore() {
	readLevel := newTimerTaskKey(time.Now().Add(-10*time.Second), 0)
	maxReadLevel := newTimerTaskKey(time.Now().Add(10*time.Second), 0)
	batchSize := 10

	request := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  readLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  maxReadLevel.(timerTaskKey).visibilityTimestamp,
		BatchSize:     batchSize,
		NextPageToken: nil,
	}

	response := &persistence.GetTimerIndexTasksResponse{
		Timers: []*persistence.TimerTaskInfo{
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: time.Now().Add(-5 * time.Second),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
		},
		NextPageToken: nil,
	}

	mockExecutionMgr := s.mockShard.Resource.ExecutionMgr
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, request).Return(response, nil).Once()

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(nil, nil, nil, nil, nil)
	timers, token, err := timerQueueProcessBase.getTimerTasks(readLevel, maxReadLevel, request.NextPageToken, batchSize)
	s.Nil(err)
	s.Equal(response.Timers, timers)
	s.Empty(token)
}

func (s *timerQueueProcessorBaseSuite) TestReadLookAheadTask() {
	mockClusterMetadata := s.mockShard.Resource.ClusterMetadata
	mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(s.clusterName).AnyTimes()

	shardMaxReadLevel := s.mockShard.UpdateTimerMaxReadLevel(s.clusterName)
	readLevel := newTimerTaskKey(shardMaxReadLevel, 0)
	maxReadLevel := newTimerTaskKey(shardMaxReadLevel.Add(10*time.Second), 0)

	request := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  readLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  maxReadLevel.(timerTaskKey).visibilityTimestamp,
		BatchSize:     1,
		NextPageToken: nil,
	}

	response := &persistence.GetTimerIndexTasksResponse{
		Timers: []*persistence.TimerTaskInfo{
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: shardMaxReadLevel,
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
				Version:             int64(79),
			},
		},
		NextPageToken: []byte("some random next page token"),
	}

	mockExecutionMgr := s.mockShard.Resource.ExecutionMgr
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, request).Return(response, nil).Once()

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(nil, nil, nil, nil, nil)
	lookAheadTask, err := timerQueueProcessBase.readLookAheadTask(readLevel, maxReadLevel)
	s.Nil(err)
	s.Equal(response.Timers[0], lookAheadTask)
}

func (s *timerQueueProcessorBaseSuite) TestReadAndFilterTasks_NoLookAhead_NoNextPage() {
	mockClusterMetadata := s.mockShard.Resource.ClusterMetadata
	mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(s.clusterName).AnyTimes()

	readLevel := newTimerTaskKey(time.Now().Add(-10*time.Second), 0)
	maxReadLevel := newTimerTaskKey(time.Now().Add(1*time.Second), 0)

	request := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  readLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  maxReadLevel.(timerTaskKey).visibilityTimestamp,
		BatchSize:     s.mockShard.GetConfig().TimerTaskBatchSize(),
		NextPageToken: []byte("some random input next page token"),
	}

	lookAheadRequest := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  maxReadLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  maximumTimerTaskKey.(timerTaskKey).visibilityTimestamp,
		BatchSize:     1,
		NextPageToken: nil,
	}

	response := &persistence.GetTimerIndexTasksResponse{
		Timers: []*persistence.TimerTaskInfo{
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: time.Now().Add(-5 * time.Second),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
		},
		NextPageToken: nil,
	}

	mockExecutionMgr := s.mockShard.Resource.ExecutionMgr
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, request).Return(response, nil).Once()
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, lookAheadRequest).Return(&persistence.GetTimerIndexTasksResponse{}, nil).Once()

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(nil, nil, nil, nil, nil)
	filteredTasks, lookAheadTask, nextPageToken, err := timerQueueProcessBase.readAndFilterTasks(readLevel, maxReadLevel, request.NextPageToken)
	s.Nil(err)
	s.Equal(response.Timers, filteredTasks)
	s.Nil(lookAheadTask)
	s.Nil(nextPageToken)
}

func (s *timerQueueProcessorBaseSuite) TestReadAndFilterTasks_NoLookAhead_HasNextPage() {
	mockClusterMetadata := s.mockShard.Resource.ClusterMetadata
	mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(s.clusterName).AnyTimes()

	readLevel := newTimerTaskKey(time.Now().Add(-10*time.Second), 0)
	maxReadLevel := newTimerTaskKey(time.Now().Add(1*time.Second), 0)

	request := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  readLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  maxReadLevel.(timerTaskKey).visibilityTimestamp,
		BatchSize:     s.mockShard.GetConfig().TimerTaskBatchSize(),
		NextPageToken: []byte("some random input next page token"),
	}

	response := &persistence.GetTimerIndexTasksResponse{
		Timers: []*persistence.TimerTaskInfo{
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: time.Now().Add(-5 * time.Second),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
		},
		NextPageToken: []byte("some random next page token"),
	}

	mockExecutionMgr := s.mockShard.Resource.ExecutionMgr
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, request).Return(response, nil).Once()

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(nil, nil, nil, nil, nil)
	filteredTasks, lookAheadTask, nextPageToken, err := timerQueueProcessBase.readAndFilterTasks(readLevel, maxReadLevel, request.NextPageToken)
	s.Nil(err)
	s.Equal(response.Timers, filteredTasks)
	s.Nil(lookAheadTask)
	s.Equal(response.NextPageToken, nextPageToken)
}

func (s *timerQueueProcessorBaseSuite) TestReadAndFilterTasks_HasLookAhead_NoNextPage() {
	mockClusterMetadata := s.mockShard.Resource.ClusterMetadata
	mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(s.clusterName).AnyTimes()

	readLevel := newTimerTaskKey(time.Now().Add(-10*time.Second), 0)
	maxReadLevel := newTimerTaskKey(time.Now().Add(1*time.Second), 0)

	request := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  readLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  maxReadLevel.(timerTaskKey).visibilityTimestamp,
		BatchSize:     s.mockShard.GetConfig().TimerTaskBatchSize(),
		NextPageToken: []byte("some random input next page token"),
	}

	response := &persistence.GetTimerIndexTasksResponse{
		Timers: []*persistence.TimerTaskInfo{
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: time.Now().Add(-5 * time.Second),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: time.Now().Add(500 * time.Millisecond),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
		},
		NextPageToken: nil,
	}

	mockExecutionMgr := s.mockShard.Resource.ExecutionMgr
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, request).Return(response, nil).Once()

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(nil, nil, nil, nil, nil)
	filteredTasks, lookAheadTask, nextPageToken, err := timerQueueProcessBase.readAndFilterTasks(readLevel, maxReadLevel, request.NextPageToken)
	s.Nil(err)
	s.Equal([]*persistence.TimerTaskInfo{response.Timers[0]}, filteredTasks)
	s.Equal(response.Timers[1], lookAheadTask)
	s.Nil(nextPageToken)
}

func (s *timerQueueProcessorBaseSuite) TestReadAndFilterTasks_HasLookAhead_HasNextPage() {
	mockClusterMetadata := s.mockShard.Resource.ClusterMetadata
	mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(s.clusterName).AnyTimes()

	readLevel := newTimerTaskKey(time.Now().Add(-10*time.Second), 0)
	maxReadLevel := newTimerTaskKey(time.Now().Add(1*time.Second), 0)

	request := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  readLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  maxReadLevel.(timerTaskKey).visibilityTimestamp,
		BatchSize:     s.mockShard.GetConfig().TimerTaskBatchSize(),
		NextPageToken: []byte("some random input next page token"),
	}

	response := &persistence.GetTimerIndexTasksResponse{
		Timers: []*persistence.TimerTaskInfo{
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: time.Now().Add(-5 * time.Second),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: time.Now().Add(500 * time.Millisecond),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
		},
		NextPageToken: []byte("some random next page token"),
	}

	mockExecutionMgr := s.mockShard.Resource.ExecutionMgr
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, request).Return(response, nil).Once()

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(nil, nil, nil, nil, nil)
	filteredTasks, lookAheadTask, nextPageToken, err := timerQueueProcessBase.readAndFilterTasks(readLevel, maxReadLevel, request.NextPageToken)
	s.Nil(err)
	s.Equal([]*persistence.TimerTaskInfo{response.Timers[0]}, filteredTasks)
	s.Equal(response.Timers[1], lookAheadTask)
	s.Nil(nextPageToken)
}

func (s *timerQueueProcessorBaseSuite) TestReadAndFilterTasks_LookAheadFailed_NoNextPage() {
	mockClusterMetadata := s.mockShard.Resource.ClusterMetadata
	mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(s.clusterName).AnyTimes()

	readLevel := newTimerTaskKey(time.Now().Add(-10*time.Second), 0)
	maxReadLevel := newTimerTaskKey(time.Now().Add(1*time.Second), 0)

	request := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  readLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  maxReadLevel.(timerTaskKey).visibilityTimestamp,
		BatchSize:     s.mockShard.GetConfig().TimerTaskBatchSize(),
		NextPageToken: []byte("some random input next page token"),
	}

	lookAheadRequest := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  maxReadLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  maximumTimerTaskKey.(timerTaskKey).visibilityTimestamp,
		BatchSize:     1,
		NextPageToken: nil,
	}

	response := &persistence.GetTimerIndexTasksResponse{
		Timers: []*persistence.TimerTaskInfo{
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: time.Now().Add(-5 * time.Second),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: time.Now().Add(-500 * time.Second),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
		},
		NextPageToken: nil,
	}

	mockExecutionMgr := s.mockShard.Resource.ExecutionMgr
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, request).Return(response, nil).Once()
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, lookAheadRequest).Return(nil, errors.New("some random error")).Times(s.mockShard.GetConfig().TimerProcessorGetFailureRetryCount())

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(nil, nil, nil, nil, nil)
	filteredTasks, lookAheadTask, nextPageToken, err := timerQueueProcessBase.readAndFilterTasks(readLevel, maxReadLevel, request.NextPageToken)
	s.Nil(err)
	s.Equal(response.Timers, filteredTasks)
	s.Equal(maxReadLevel.(timerTaskKey).visibilityTimestamp, lookAheadTask.VisibilityTimestamp)
	s.Nil(nextPageToken)
}

func (s *timerQueueProcessorBaseSuite) TestNotifyNewTimes() {
	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(nil, nil, nil, nil, nil)

	// assert the initial state
	s.True(timerQueueProcessBase.newTime.IsZero())
	select {
	case <-timerQueueProcessBase.newTimerCh:
	default:
	}

	now := time.Now()
	timerQueueProcessBase.notifyNewTimers([]persistence.Task{
		&persistence.UserTimerTask{
			VisibilityTimestamp: now.Add(5 * time.Second),
			TaskID:              int64(59),
			EventID:             int64(28),
		},
		&persistence.UserTimerTask{
			VisibilityTimestamp: now.Add(1 * time.Second),
			TaskID:              int64(59),
			EventID:             int64(28),
		},
	})
	select {
	case <-timerQueueProcessBase.newTimerCh:
		s.Equal(now.Add(1*time.Second), timerQueueProcessBase.newTime)
	default:
		s.Fail("should notify new timer")
	}

	timerQueueProcessBase.notifyNewTimers([]persistence.Task{
		&persistence.UserTimerTask{
			VisibilityTimestamp: now.Add(10 * time.Second),
			TaskID:              int64(59),
			EventID:             int64(28),
		},
	})
	select {
	case <-timerQueueProcessBase.newTimerCh:
		s.Fail("should not notify new timer")
	default:
		s.Equal(now.Add(1*time.Second), timerQueueProcessBase.newTime)
	}
}

func (s *timerQueueProcessorBaseSuite) TestProcessQueueCollections_SkipRead() {
	mockClusterMetadata := s.mockShard.Resource.ClusterMetadata
	mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(s.clusterName).AnyTimes()

	now := time.Now()
	queueLevel := 0
	shardMaxReadLevel := newTimerTaskKey(now, 0)
	ackLevel := newTimerTaskKey(now.Add(50*time.Millisecond), 0)
	maxLevel := newTimerTaskKey(now.Add(10*time.Second), 0)
	processingQueueStates := []ProcessingQueueState{
		NewProcessingQueueState(
			queueLevel,
			ackLevel,
			maxLevel,
			NewDomainFilter(map[string]struct{}{"testDomain1": {}}, false),
		),
	}
	updateMaxReadLevel := func() task.Key {
		return shardMaxReadLevel
	}

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(processingQueueStates, updateMaxReadLevel, nil, nil, nil)
	timerQueueProcessBase.processQueueCollections(map[int]struct{}{queueLevel: {}})

	s.Len(timerQueueProcessBase.processingQueueCollections, 1)
	s.Len(timerQueueProcessBase.processingQueueCollections[0].Queues(), 1)
	activeQueue := timerQueueProcessBase.processingQueueCollections[0].ActiveQueue()
	s.NotNil(activeQueue)
	s.Equal(ackLevel, activeQueue.State().AckLevel())
	s.Equal(ackLevel, activeQueue.State().ReadLevel())
	s.Equal(maxLevel, activeQueue.State().MaxLevel())

	s.Empty(timerQueueProcessBase.processingQueueReadProgress)

	s.Empty(timerQueueProcessBase.backoffTimer)
	time.Sleep(100 * time.Millisecond)
	s.True(timerQueueProcessBase.nextPollTime[queueLevel].Before(s.mockShard.GetTimeSource().Now()))
	select {
	case <-timerQueueProcessBase.timerGate.FireChan():
	default:
		s.Fail("timer gate should fire")
	}
}

func (s *timerQueueProcessorBaseSuite) TestProcessBatch_HasNextPage() {
	mockClusterMetadata := s.mockShard.Resource.ClusterMetadata
	mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(s.clusterName).AnyTimes()

	now := time.Now()
	queueLevel := 0
	ackLevel := newTimerTaskKey(now.Add(-5*time.Second), 0)
	shardMaxReadLevel := newTimerTaskKey(now.Add(1*time.Second), 0)
	maxLevel := newTimerTaskKey(now.Add(10*time.Second), 0)
	processingQueueStates := []ProcessingQueueState{
		NewProcessingQueueState(
			queueLevel,
			ackLevel,
			maxLevel,
			NewDomainFilter(map[string]struct{}{"excludedDomain": {}}, true),
		),
	}
	updateMaxReadLevel := func() task.Key {
		return shardMaxReadLevel
	}

	request := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  ackLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  shardMaxReadLevel.(timerTaskKey).visibilityTimestamp,
		BatchSize:     s.mockShard.GetConfig().TimerTaskBatchSize(),
		NextPageToken: nil,
	}

	response := &persistence.GetTimerIndexTasksResponse{
		Timers: []*persistence.TimerTaskInfo{
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: now.Add(-3 * time.Second),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
			{
				DomainID:            "excludedDomain",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: now.Add(-2 * time.Second),
				TaskID:              int64(60),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
		},
		NextPageToken: []byte("some random next page token"),
	}

	mockExecutionMgr := s.mockShard.Resource.ExecutionMgr
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, request).Return(response, nil).Once()

	s.mockTaskProcessor.EXPECT().TrySubmit(gomock.Any()).Return(true, nil).AnyTimes()

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(processingQueueStates, updateMaxReadLevel, nil, nil, nil)
	timerQueueProcessBase.processQueueCollections(map[int]struct{}{queueLevel: {}})

	s.Len(timerQueueProcessBase.processingQueueCollections, 1)
	s.Len(timerQueueProcessBase.processingQueueCollections[0].Queues(), 1)
	activeQueue := timerQueueProcessBase.processingQueueCollections[0].ActiveQueue()
	s.NotNil(activeQueue)
	s.Equal(ackLevel, activeQueue.State().AckLevel())
	s.Equal(newTimerTaskKey(response.Timers[1].VisibilityTimestamp, 0), activeQueue.State().ReadLevel())
	s.Equal(maxLevel, activeQueue.State().MaxLevel())
	s.Len(activeQueue.(*processingQueueImpl).outstandingTasks, 1)

	s.Len(timerQueueProcessBase.processingQueueReadProgress, 1)
	s.Equal(timeTaskReadProgress{
		currentQueue:  activeQueue,
		readLevel:     ackLevel,
		maxReadLevel:  shardMaxReadLevel,
		nextPageToken: response.NextPageToken,
	}, timerQueueProcessBase.processingQueueReadProgress[0])

	s.True(timerQueueProcessBase.nextPollTime[queueLevel].IsZero())
	s.Empty(timerQueueProcessBase.backoffTimer)
	time.Sleep(100 * time.Millisecond)
	select {
	case <-timerQueueProcessBase.timerGate.FireChan():
	default:
		s.Fail("timer gate should fire")
	}
}

func (s *timerQueueProcessorBaseSuite) TestProcessBatch_NoNextPage_HasLookAhead() {
	mockClusterMetadata := s.mockShard.Resource.ClusterMetadata
	mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(s.clusterName).AnyTimes()

	now := time.Now()
	queueLevel := 0
	ackLevel := newTimerTaskKey(now.Add(-5*time.Second), 0)
	shardMaxReadLevel := newTimerTaskKey(now.Add(1*time.Second), 0)
	maxLevel := newTimerTaskKey(now.Add(10*time.Second), 0)
	processingQueueStates := []ProcessingQueueState{
		NewProcessingQueueState(
			queueLevel,
			ackLevel,
			maxLevel,
			NewDomainFilter(map[string]struct{}{"excludedDomain": {}}, true),
		),
	}
	updateMaxReadLevel := func() task.Key {
		return shardMaxReadLevel
	}

	requestNextPageToken := []byte("some random input next page token")
	request := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  ackLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  shardMaxReadLevel.(timerTaskKey).visibilityTimestamp,
		BatchSize:     s.mockShard.GetConfig().TimerTaskBatchSize(),
		NextPageToken: requestNextPageToken,
	}

	lookAheadTaskTimestamp := now.Add(50 * time.Millisecond)
	response := &persistence.GetTimerIndexTasksResponse{
		Timers: []*persistence.TimerTaskInfo{
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: now.Add(-3 * time.Second),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
			{
				DomainID:            "excludedDomain",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: lookAheadTaskTimestamp,
				TaskID:              int64(60),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
		},
		NextPageToken: nil,
	}

	mockExecutionMgr := s.mockShard.Resource.ExecutionMgr
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, request).Return(response, nil).Once()

	s.mockTaskProcessor.EXPECT().TrySubmit(gomock.Any()).Return(true, nil).AnyTimes()

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(processingQueueStates, updateMaxReadLevel, nil, nil, nil)
	timerQueueProcessBase.processingQueueReadProgress[0] = timeTaskReadProgress{
		currentQueue:  timerQueueProcessBase.processingQueueCollections[0].ActiveQueue(),
		readLevel:     ackLevel,
		maxReadLevel:  shardMaxReadLevel,
		nextPageToken: requestNextPageToken,
	}
	timerQueueProcessBase.processQueueCollections(map[int]struct{}{queueLevel: {}})

	s.Len(timerQueueProcessBase.processingQueueCollections, 1)
	s.Len(timerQueueProcessBase.processingQueueCollections[0].Queues(), 1)
	activeQueue := timerQueueProcessBase.processingQueueCollections[0].ActiveQueue()
	s.NotNil(activeQueue)
	s.Equal(ackLevel, activeQueue.State().AckLevel())
	s.Equal(newTimerTaskKey(lookAheadTaskTimestamp, 0), activeQueue.State().ReadLevel())
	s.Equal(maxLevel, activeQueue.State().MaxLevel())
	s.Len(activeQueue.(*processingQueueImpl).outstandingTasks, 1)

	s.Empty(timerQueueProcessBase.processingQueueReadProgress)

	s.Empty(timerQueueProcessBase.backoffTimer)
	s.Equal(lookAheadTaskTimestamp, timerQueueProcessBase.nextPollTime[queueLevel])
	time.Sleep(100 * time.Millisecond)
	select {
	case <-timerQueueProcessBase.timerGate.FireChan():
	default:
		s.Fail("timer gate should fire")
	}
}

func (s *timerQueueProcessorBaseSuite) TestProcessBatch_NoNextPage_NoLookAhead() {
	mockClusterMetadata := s.mockShard.Resource.ClusterMetadata
	mockClusterMetadata.EXPECT().GetCurrentClusterName().Return(s.clusterName).AnyTimes()

	now := time.Now()
	queueLevel := 0
	ackLevel := newTimerTaskKey(now.Add(-5*time.Second), 0)
	shardMaxReadLevel := newTimerTaskKey(now.Add(1*time.Second), 0)
	maxLevel := newTimerTaskKey(now.Add(10*time.Second), 0)
	processingQueueStates := []ProcessingQueueState{
		NewProcessingQueueState(
			queueLevel,
			ackLevel,
			maxLevel,
			NewDomainFilter(map[string]struct{}{"excludedDomain": {}}, true),
		),
	}
	updateMaxReadLevel := func() task.Key {
		return shardMaxReadLevel
	}

	requestNextPageToken := []byte("some random input next page token")
	request := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  ackLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  shardMaxReadLevel.(timerTaskKey).visibilityTimestamp,
		BatchSize:     s.mockShard.GetConfig().TimerTaskBatchSize(),
		NextPageToken: requestNextPageToken,
	}

	lookAheadRequest := &persistence.GetTimerIndexTasksRequest{
		MinTimestamp:  shardMaxReadLevel.(timerTaskKey).visibilityTimestamp,
		MaxTimestamp:  maximumTimerTaskKey.(timerTaskKey).visibilityTimestamp,
		BatchSize:     1,
		NextPageToken: nil,
	}

	response := &persistence.GetTimerIndexTasksResponse{
		Timers: []*persistence.TimerTaskInfo{
			{
				DomainID:            "some random domain ID",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: now.Add(-3 * time.Second),
				TaskID:              int64(59),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
			{
				DomainID:            "excludedDomain",
				WorkflowID:          "some random workflow ID",
				RunID:               uuid.New(),
				VisibilityTimestamp: now.Add(-2 * time.Second),
				TaskID:              int64(60),
				TaskType:            1,
				TimeoutType:         2,
				EventID:             int64(28),
				ScheduleAttempt:     0,
			},
		},
		NextPageToken: nil,
	}

	mockExecutionMgr := s.mockShard.Resource.ExecutionMgr
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, request).Return(response, nil).Once()
	mockExecutionMgr.On("GetTimerIndexTasks", mock.Anything, lookAheadRequest).Return(&persistence.GetTimerIndexTasksResponse{}, nil).Once()

	s.mockTaskProcessor.EXPECT().TrySubmit(gomock.Any()).Return(true, nil).AnyTimes()

	timerQueueProcessBase := s.newTestTimerQueueProcessorBase(processingQueueStates, updateMaxReadLevel, nil, nil, nil)
	timerQueueProcessBase.processingQueueReadProgress[0] = timeTaskReadProgress{
		currentQueue:  timerQueueProcessBase.processingQueueCollections[0].ActiveQueue(),
		readLevel:     ackLevel,
		maxReadLevel:  shardMaxReadLevel,
		nextPageToken: requestNextPageToken,
	}
	timerQueueProcessBase.processQueueCollections(map[int]struct{}{queueLevel: {}})

	s.Len(timerQueueProcessBase.processingQueueCollections, 1)
	s.Len(timerQueueProcessBase.processingQueueCollections[0].Queues(), 1)
	activeQueue := timerQueueProcessBase.processingQueueCollections[0].ActiveQueue()
	s.NotNil(activeQueue)
	s.Equal(ackLevel, activeQueue.State().AckLevel())
	s.Equal(shardMaxReadLevel, activeQueue.State().ReadLevel())
	s.Equal(maxLevel, activeQueue.State().MaxLevel())
	s.Len(activeQueue.(*processingQueueImpl).outstandingTasks, 1)

	s.Empty(timerQueueProcessBase.processingQueueReadProgress)

	_, ok := timerQueueProcessBase.nextPollTime[queueLevel]
	s.True(ok) // this is the poll time for max poll interval
	time.Sleep(100 * time.Millisecond)
	select {
	case <-timerQueueProcessBase.timerGate.FireChan():
		s.Fail("timer gate should not fire")
	default:
	}
}

func (s *timerQueueProcessorBaseSuite) newTestTimerQueueProcessorBase(
	processingQueueStates []ProcessingQueueState,
	updateMaxReadLevel updateMaxReadLevelFn,
	updateClusterAckLevel updateClusterAckLevelFn,
	updateProcessingQueueStates updateProcessingQueueStatesFn,
	queueShutdown queueShutdownFn,
) *timerQueueProcessorBase {
	return newTimerQueueProcessorBase(
		s.clusterName,
		s.mockShard,
		processingQueueStates,
		s.mockTaskProcessor,
		NewLocalTimerGate(s.mockShard.GetTimeSource()),
		newTimerQueueProcessorOptions(s.mockShard.GetConfig(), true, false),
		updateMaxReadLevel,
		updateClusterAckLevel,
		updateProcessingQueueStates,
		queueShutdown,
		nil,
		nil,
		s.logger,
		s.metricsClient,
	)
}
