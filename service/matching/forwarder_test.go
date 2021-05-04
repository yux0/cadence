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

package matching

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"

	gen "github.com/uber/cadence/.gen/go/matching"
	"github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/client/matching"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/common/types/mapper/thrift"
)

type ForwarderTestSuite struct {
	suite.Suite
	controller *gomock.Controller
	client     *matching.MockClient
	fwdr       *Forwarder
	cfg        *forwarderConfig
	taskList   *taskListID
}

func TestForwarderSuite(t *testing.T) {
	suite.Run(t, new(ForwarderTestSuite))
}

func (t *ForwarderTestSuite) SetupTest() {
	t.controller = gomock.NewController(t.T())
	t.client = matching.NewMockClient(t.controller)
	t.cfg = &forwarderConfig{
		ForwarderMaxOutstandingPolls: func() int { return 1 },
		ForwarderMaxRatePerSecond:    func() int { return 2 },
		ForwarderMaxChildrenPerNode:  func() int { return 20 },
		ForwarderMaxOutstandingTasks: func() int { return 1 },
	}
	t.taskList = newTestTaskListID("fwdr", "tl0", persistence.TaskListTypeDecision)
	t.fwdr = newForwarder(t.cfg, t.taskList, shared.TaskListKindNormal, t.client)
}

func (t *ForwarderTestSuite) TearDownTest() {
	t.controller.Finish()
}

func (t *ForwarderTestSuite) TestForwardTaskError() {
	task := newInternalTask(&persistence.TaskInfo{}, nil, gen.TaskSourceHistory, "", false)
	t.Equal(errNoParent, t.fwdr.ForwardTask(context.Background(), task))

	t.usingTasklistPartition(persistence.TaskListTypeActivity)
	t.fwdr.taskListKind = shared.TaskListKindSticky
	t.Equal(errTaskListKind, t.fwdr.ForwardTask(context.Background(), task))
}

func (t *ForwarderTestSuite) TestForwardDecisionTask() {
	t.usingTasklistPartition(persistence.TaskListTypeDecision)

	var request *gen.AddDecisionTaskRequest
	t.client.EXPECT().AddDecisionTask(gomock.Any(), gomock.Any()).Do(
		func(arg0 context.Context, arg1 *types.AddDecisionTaskRequest) {
			request = thrift.FromAddDecisionTaskRequest(arg1)
		},
	).Return(nil).Times(1)

	taskInfo := t.newTaskInfo()
	task := newInternalTask(taskInfo, nil, gen.TaskSourceHistory, "", false)
	t.NoError(t.fwdr.ForwardTask(context.Background(), task))
	t.NotNil(request)
	t.Equal(t.taskList.Parent(20), request.TaskList.GetName())
	t.Equal(t.fwdr.taskListKind, request.TaskList.GetKind())
	t.Equal(taskInfo.DomainID, request.GetDomainUUID())
	t.Equal(taskInfo.WorkflowID, request.GetExecution().GetWorkflowId())
	t.Equal(taskInfo.RunID, request.GetExecution().GetRunId())
	t.Equal(taskInfo.ScheduleID, request.GetScheduleId())
	t.Equal(taskInfo.ScheduleToStartTimeout, request.GetScheduleToStartTimeoutSeconds())
	t.Equal(t.taskList.name, request.GetForwardedFrom())
}

func (t *ForwarderTestSuite) TestForwardActivityTask() {
	t.usingTasklistPartition(persistence.TaskListTypeActivity)

	var request *gen.AddActivityTaskRequest
	t.client.EXPECT().AddActivityTask(gomock.Any(), gomock.Any()).Do(
		func(arg0 context.Context, arg1 *types.AddActivityTaskRequest) {
			request = thrift.FromAddActivityTaskRequest(arg1)
		},
	).Return(nil).Times(1)

	taskInfo := t.newTaskInfo()
	task := newInternalTask(taskInfo, nil, gen.TaskSourceHistory, "", false)
	t.NoError(t.fwdr.ForwardTask(context.Background(), task))
	t.NotNil(request)
	t.Equal(t.taskList.Parent(20), request.TaskList.GetName())
	t.Equal(t.fwdr.taskListKind, request.TaskList.GetKind())
	t.Equal(t.taskList.domainID, request.GetDomainUUID())
	t.Equal(taskInfo.DomainID, request.GetSourceDomainUUID())
	t.Equal(taskInfo.WorkflowID, request.GetExecution().GetWorkflowId())
	t.Equal(taskInfo.RunID, request.GetExecution().GetRunId())
	t.Equal(taskInfo.ScheduleID, request.GetScheduleId())
	t.Equal(taskInfo.ScheduleToStartTimeout, request.GetScheduleToStartTimeoutSeconds())
	t.Equal(t.taskList.name, request.GetForwardedFrom())
}

func (t *ForwarderTestSuite) TestForwardTaskRateExceeded() {
	t.usingTasklistPartition(persistence.TaskListTypeActivity)

	rps := 2
	t.client.EXPECT().AddActivityTask(gomock.Any(), gomock.Any()).Return(nil).Times(rps)
	taskInfo := t.newTaskInfo()
	task := newInternalTask(taskInfo, nil, gen.TaskSourceHistory, "", false)
	for i := 0; i < rps; i++ {
		t.NoError(t.fwdr.ForwardTask(context.Background(), task))
	}
	t.Equal(errForwarderSlowDown, t.fwdr.ForwardTask(context.Background(), task))
}

func (t *ForwarderTestSuite) TestForwardQueryTaskError() {
	task := newInternalQueryTask("id1", &gen.QueryWorkflowRequest{})
	_, err := t.fwdr.ForwardQueryTask(context.Background(), task)
	t.Equal(errNoParent, err)

	t.usingTasklistPartition(persistence.TaskListTypeDecision)
	t.fwdr.taskListKind = shared.TaskListKindSticky
	_, err = t.fwdr.ForwardQueryTask(context.Background(), task)
	t.Equal(errTaskListKind, err)
}

func (t *ForwarderTestSuite) TestForwardQueryTask() {
	t.usingTasklistPartition(persistence.TaskListTypeDecision)
	task := newInternalQueryTask("id1", &gen.QueryWorkflowRequest{})
	resp := &shared.QueryWorkflowResponse{}
	var request *gen.QueryWorkflowRequest
	t.client.EXPECT().QueryWorkflow(gomock.Any(), gomock.Any()).Do(
		func(arg0 context.Context, arg1 *types.MatchingQueryWorkflowRequest) {
			request = thrift.FromMatchingQueryWorkflowRequest(arg1)
		},
	).Return(thrift.ToQueryWorkflowResponse(resp), nil).Times(1)

	gotResp, err := t.fwdr.ForwardQueryTask(context.Background(), task)
	t.NoError(err)
	t.Equal(t.taskList.Parent(20), request.TaskList.GetName())
	t.Equal(t.fwdr.taskListKind, request.TaskList.GetKind())
	t.True(task.query.request.QueryRequest == request.QueryRequest)
	t.Equal(resp, gotResp)
}

func (t *ForwarderTestSuite) TestForwardQueryTaskRateNotEnforced() {
	t.usingTasklistPartition(persistence.TaskListTypeActivity)
	task := newInternalQueryTask("id1", &gen.QueryWorkflowRequest{})
	resp := &shared.QueryWorkflowResponse{}
	rps := 2
	t.client.EXPECT().QueryWorkflow(gomock.Any(), gomock.Any()).Return(thrift.ToQueryWorkflowResponse(resp), nil).Times(rps + 1)
	for i := 0; i < rps; i++ {
		_, err := t.fwdr.ForwardQueryTask(context.Background(), task)
		t.NoError(err)
	}
	_, err := t.fwdr.ForwardQueryTask(context.Background(), task)
	t.NoError(err) // no rateliming should be enforced for query task
}

func (t *ForwarderTestSuite) TestForwardPollError() {
	_, err := t.fwdr.ForwardPoll(context.Background())
	t.Equal(errNoParent, err)

	t.usingTasklistPartition(persistence.TaskListTypeActivity)
	t.fwdr.taskListKind = shared.TaskListKindSticky
	_, err = t.fwdr.ForwardPoll(context.Background())
	t.Equal(errTaskListKind, err)

}

func (t *ForwarderTestSuite) TestForwardPollForDecision() {
	t.usingTasklistPartition(persistence.TaskListTypeDecision)

	pollerID := uuid.New()
	ctx := context.WithValue(context.Background(), pollerIDKey, pollerID)
	ctx = context.WithValue(ctx, identityKey, "id1")
	resp := &gen.PollForDecisionTaskResponse{}

	var request *gen.PollForDecisionTaskRequest
	t.client.EXPECT().PollForDecisionTask(gomock.Any(), gomock.Any()).Do(
		func(arg0 context.Context, arg1 *types.MatchingPollForDecisionTaskRequest) {
			request = thrift.FromMatchingPollForDecisionTaskRequest(arg1)
		},
	).Return(thrift.ToMatchingPollForDecisionTaskResponse(resp), nil).Times(1)

	task, err := t.fwdr.ForwardPoll(ctx)
	t.NoError(err)
	t.NotNil(task)
	t.NotNil(request)
	t.Equal(pollerID, request.GetPollerID())
	t.Equal(t.taskList.domainID, request.GetDomainUUID())
	t.Equal("id1", request.GetPollRequest().GetIdentity())
	t.Equal(t.taskList.Parent(20), request.GetPollRequest().GetTaskList().GetName())
	t.Equal(t.fwdr.taskListKind, request.GetPollRequest().GetTaskList().GetKind())
	t.Equal(resp, task.pollForDecisionResponse())
	t.Nil(task.pollForActivityResponse())
}

func (t *ForwarderTestSuite) TestForwardPollForActivity() {
	t.usingTasklistPartition(persistence.TaskListTypeActivity)

	pollerID := uuid.New()
	ctx := context.WithValue(context.Background(), pollerIDKey, pollerID)
	ctx = context.WithValue(ctx, identityKey, "id1")
	resp := &shared.PollForActivityTaskResponse{}

	var request *gen.PollForActivityTaskRequest
	t.client.EXPECT().PollForActivityTask(gomock.Any(), gomock.Any()).Do(
		func(arg0 context.Context, arg1 *types.MatchingPollForActivityTaskRequest) {
			request = thrift.FromMatchingPollForActivityTaskRequest(arg1)
		},
	).Return(thrift.ToPollForActivityTaskResponse(resp), nil).Times(1)

	task, err := t.fwdr.ForwardPoll(ctx)
	t.NoError(err)
	t.NotNil(task)
	t.NotNil(request)
	t.Equal(pollerID, request.GetPollerID())
	t.Equal(t.taskList.domainID, request.GetDomainUUID())
	t.Equal("id1", request.GetPollRequest().GetIdentity())
	t.Equal(t.taskList.Parent(20), request.GetPollRequest().GetTaskList().GetName())
	t.Equal(t.fwdr.taskListKind, request.GetPollRequest().GetTaskList().GetKind())
	t.Equal(resp, task.pollForActivityResponse())
	t.Nil(task.pollForDecisionResponse())
}

func (t *ForwarderTestSuite) TestMaxOutstandingConcurrency() {
	concurrency := 50
	testCases := []struct {
		name          string
		mustLeakToken bool
		output        int32
	}{
		{"contention", false, int32(concurrency)},
		{"token_leak", true, 1},
	}

	var adds int32
	var polls int32
	var wg sync.WaitGroup

	for _, tc := range testCases {
		adds = 0
		polls = 0
		t.Run(tc.name, func() {
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					select {
					case token := <-t.fwdr.AddReqTokenC():
						if !tc.mustLeakToken {
							token.release()
						}
						atomic.AddInt32(&adds, 1)
					case <-time.After(time.Millisecond * 200):
						break
					}

					select {
					case token := <-t.fwdr.PollReqTokenC():
						if !tc.mustLeakToken {
							token.release()
						}
						atomic.AddInt32(&polls, 1)
					case <-time.After(time.Millisecond * 200):
						break
					}
					wg.Done()
				}()
			}
			t.True(common.AwaitWaitGroup(&wg, time.Second))
			t.Equal(tc.output, adds)
			t.Equal(tc.output, polls)
		})
	}
}

func (t *ForwarderTestSuite) TestMaxOutstandingConfigUpdate() {
	maxOutstandingTasks := int32(1)
	maxOutstandingPolls := int32(1)
	t.fwdr.cfg.ForwarderMaxOutstandingTasks = func() int { return int(atomic.LoadInt32(&maxOutstandingTasks)) }
	t.fwdr.cfg.ForwarderMaxOutstandingPolls = func() int { return int(atomic.LoadInt32(&maxOutstandingPolls)) }

	startC := make(chan struct{})
	doneWG := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		doneWG.Add(1)
		go func() {
			<-startC
			token1 := <-t.fwdr.AddReqTokenC()
			token1.release()
			token2 := <-t.fwdr.PollReqTokenC()
			token2.release()
			doneWG.Done()
		}()
	}

	maxOutstandingTasks = 10
	maxOutstandingPolls = 10
	close(startC)
	t.True(common.AwaitWaitGroup(&doneWG, time.Second))

	t.Equal(10, cap(t.fwdr.addReqToken.Load().(*ForwarderReqToken).ch))
	t.Equal(10, cap(t.fwdr.pollReqToken.Load().(*ForwarderReqToken).ch))
}

func (t *ForwarderTestSuite) usingTasklistPartition(taskType int) {
	t.taskList = newTestTaskListID("fwdr", taskListPartitionPrefix+"tl0/1", taskType)
	t.fwdr.taskListID = t.taskList
}

func (t *ForwarderTestSuite) newTaskInfo() *persistence.TaskInfo {
	return &persistence.TaskInfo{
		DomainID:               uuid.New(),
		WorkflowID:             uuid.New(),
		RunID:                  uuid.New(),
		TaskID:                 rand.Int63(),
		ScheduleID:             rand.Int63(),
		ScheduleToStartTimeout: rand.Int31(),
	}
}
