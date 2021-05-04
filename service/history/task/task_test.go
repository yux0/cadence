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

package task

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/clock"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/loggerimpl"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/service/dynamicconfig"
	t "github.com/uber/cadence/common/task"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/service/history/config"
	"github.com/uber/cadence/service/history/constants"
	"github.com/uber/cadence/service/history/shard"
)

type (
	taskSuite struct {
		suite.Suite
		*require.Assertions

		controller           *gomock.Controller
		mockShard            *shard.TestContext
		mockTaskExecutor     *MockExecutor
		mockTaskProcessor    *MockProcessor
		mockTaskRedispatcher *MockRedispatcher
		mockTaskInfo         *MockInfo

		logger        log.Logger
		timeSource    clock.TimeSource
		maxRetryCount dynamicconfig.IntPropertyFn
	}
)

func TestTaskSuite(t *testing.T) {
	s := new(taskSuite)
	suite.Run(t, s)
}

func (s *taskSuite) SetupTest() {
	s.Assertions = require.New(s.T())

	s.controller = gomock.NewController(s.T())
	s.mockShard = shard.NewTestContext(
		s.controller,
		&persistence.ShardInfo{
			ShardID: 10,
			RangeID: 1,
		},
		config.NewForTest(),
	)
	s.mockTaskExecutor = NewMockExecutor(s.controller)
	s.mockTaskProcessor = NewMockProcessor(s.controller)
	s.mockTaskRedispatcher = NewMockRedispatcher(s.controller)
	s.mockTaskInfo = NewMockInfo(s.controller)
	s.mockTaskInfo.EXPECT().GetDomainID().Return(constants.TestDomainID).AnyTimes()
	s.mockShard.Resource.DomainCache.EXPECT().GetDomainName(constants.TestDomainID).Return(constants.TestDomainName, nil).AnyTimes()

	s.logger = loggerimpl.NewDevelopmentForTest(s.Suite)
	s.timeSource = clock.NewRealTimeSource()
	s.maxRetryCount = dynamicconfig.GetIntPropertyFn(10)
}

func (s *taskSuite) TearDownTest() {
	s.controller.Finish()
	s.mockShard.Finish(s.T())
}

func (s *taskSuite) TestExecute_TaskFilterErr() {
	taskFilterErr := errors.New("some random error")
	taskBase := s.newTestQueueTaskBase(func(task Info) (bool, error) {
		return false, taskFilterErr
	})
	err := taskBase.Execute()
	s.Equal(taskFilterErr, err)
}

func (s *taskSuite) TestExecute_ExecutionErr() {
	taskBase := s.newTestQueueTaskBase(func(task Info) (bool, error) {
		return true, nil
	})

	executionErr := errors.New("some random error")
	s.mockTaskExecutor.EXPECT().Execute(taskBase.Info, true).Return(executionErr).Times(1)

	err := taskBase.Execute()
	s.Equal(executionErr, err)
}

func (s *taskSuite) TestExecute_Success() {
	taskBase := s.newTestQueueTaskBase(func(task Info) (bool, error) {
		return true, nil
	})

	s.mockTaskExecutor.EXPECT().Execute(taskBase.Info, true).Return(nil).Times(1)

	err := taskBase.Execute()
	s.NoError(err)
}

func (s *taskSuite) TestHandleErr_ErrEntityNotExists() {
	taskBase := s.newTestQueueTaskBase(func(task Info) (bool, error) {
		return true, nil
	})

	err := &types.EntityNotExistsError{}
	s.NoError(taskBase.HandleErr(err))
}

func (s *taskSuite) TestHandleErr_ErrTaskRetry() {
	taskBase := s.newTestQueueTaskBase(func(task Info) (bool, error) {
		return true, nil
	})

	err := ErrTaskRedispatch
	s.Equal(ErrTaskRedispatch, taskBase.HandleErr(err))
}

func (s *taskSuite) TestHandleErr_ErrTaskDiscarded() {
	taskBase := s.newTestQueueTaskBase(func(task Info) (bool, error) {
		return true, nil
	})

	err := ErrTaskDiscarded
	s.NoError(taskBase.HandleErr(err))
}

func (s *taskSuite) TestHandleErr_ErrDomainNotActive() {
	taskBase := s.newTestQueueTaskBase(func(task Info) (bool, error) {
		return true, nil
	})

	err := &types.DomainNotActiveError{}

	taskBase.submitTime = time.Now().Add(-cache.DomainCacheRefreshInterval * time.Duration(2))
	s.NoError(taskBase.HandleErr(err))

	taskBase.submitTime = time.Now()
	s.Equal(err, taskBase.HandleErr(err))
}

func (s *taskSuite) TestHandleErr_ErrCurrentWorkflowConditionFailed() {
	taskBase := s.newTestQueueTaskBase(func(task Info) (bool, error) {
		return true, nil
	})

	err := &persistence.CurrentWorkflowConditionFailedError{}
	s.NoError(taskBase.HandleErr(err))
}

func (s *taskSuite) TestHandleErr_UnknownErr() {
	taskBase := s.newTestQueueTaskBase(func(task Info) (bool, error) {
		return true, nil
	})

	err := errors.New("some random error")
	s.Equal(err, taskBase.HandleErr(err))
}

func (s *taskSuite) TestTaskState() {
	taskBase := s.newTestQueueTaskBase(func(task Info) (bool, error) {
		return true, nil
	})

	s.Equal(t.TaskStatePending, taskBase.State())

	taskBase.Ack()
	s.Equal(t.TaskStateAcked, taskBase.State())

	taskBase.Nack()
	s.Equal(t.TaskStateNacked, taskBase.State())
}

func (s *taskSuite) TestTaskPriority() {
	taskBase := s.newTestQueueTaskBase(func(task Info) (bool, error) {
		return true, nil
	})

	priority := 10
	taskBase.SetPriority(priority)
	s.Equal(priority, taskBase.Priority())
}

func (s *taskSuite) TestTaskNack_ResubmitSucceeded() {
	task := &transferTask{
		taskBase: s.newTestQueueTaskBase(
			func(task Info) (bool, error) {
				return true, nil
			},
		),
		ackMgr: nil,
		redispatchFn: func(task Task) {
			s.mockTaskRedispatcher.AddTask(task)
		},
	}

	s.mockTaskProcessor.EXPECT().TrySubmit(task).Return(true, nil).Times(1)

	task.Nack()
	s.Equal(t.TaskStateNacked, task.State())
}

func (s *taskSuite) TestTaskNack_ResubmitFailed() {
	task := &transferTask{
		taskBase: s.newTestQueueTaskBase(
			func(task Info) (bool, error) {
				return true, nil
			},
		),
		ackMgr: nil,
		redispatchFn: func(task Task) {
			s.mockTaskRedispatcher.AddTask(task)
		},
	}

	s.mockTaskProcessor.EXPECT().TrySubmit(task).Return(false, errTaskProcessorNotRunning).Times(1)
	s.mockTaskRedispatcher.EXPECT().AddTask(task).Times(1)

	task.Nack()
	s.Equal(t.TaskStateNacked, task.State())
}

func (s *taskSuite) newTestQueueTaskBase(
	taskFilter Filter,
) *taskBase {
	taskBase := newQueueTaskBase(
		s.mockShard,
		s.mockTaskInfo,
		QueueTypeActiveTransfer,
		0,
		s.logger,
		taskFilter,
		s.mockTaskExecutor,
		s.mockTaskProcessor,
		s.timeSource,
		s.maxRetryCount,
	)
	taskBase.scope = s.mockShard.GetMetricsClient().Scope(0)
	return taskBase
}
