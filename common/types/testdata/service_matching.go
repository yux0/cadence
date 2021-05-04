// Copyright (c) 2021 Uber Technologies Inc.
//
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

package testdata

import (
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/types"
)

const (
	ForwardedFrom = "ForwardedFrom"
	PollerID      = "PollerID"
)

var (
	MatchingAddActivityTaskRequest = types.AddActivityTaskRequest{
		DomainUUID:                    DomainID,
		Execution:                     &WorkflowExecution,
		SourceDomainUUID:              DomainID,
		TaskList:                      &TaskList,
		ScheduleID:                    EventID1,
		ScheduleToStartTimeoutSeconds: &Duration1,
		Source:                        types.TaskSourceDbBacklog.Ptr(),
		ForwardedFrom:                 ForwardedFrom,
	}
	MatchingAddDecisionTaskRequest = types.AddDecisionTaskRequest{
		DomainUUID:                    DomainID,
		Execution:                     &WorkflowExecution,
		TaskList:                      &TaskList,
		ScheduleID:                    EventID1,
		ScheduleToStartTimeoutSeconds: &Duration1,
		Source:                        types.TaskSourceDbBacklog.Ptr(),
		ForwardedFrom:                 ForwardedFrom,
	}
	MatchingCancelOutstandingPollRequest = types.CancelOutstandingPollRequest{
		DomainUUID:   DomainID,
		TaskListType: common.Int32Ptr(int32(TaskListType)),
		TaskList:     &TaskList,
		PollerID:     PollerID,
	}
	MatchingDescribeTaskListRequest = types.MatchingDescribeTaskListRequest{
		DomainUUID:  DomainID,
		DescRequest: &DescribeTaskListRequest,
	}
	MatchingDescribeTaskListResponse = types.DescribeTaskListResponse{
		Pollers:        PollerInfoArray,
		TaskListStatus: &TaskListStatus,
	}
	MatchingListTaskListPartitionsRequest = types.MatchingListTaskListPartitionsRequest{
		Domain:   DomainName,
		TaskList: &TaskList,
	}
	MatchingListTaskListPartitionsResponse = types.ListTaskListPartitionsResponse{
		ActivityTaskListPartitions: TaskListPartitionMetadataArray,
		DecisionTaskListPartitions: TaskListPartitionMetadataArray,
	}
	MatchingPollForActivityTaskRequest = types.MatchingPollForActivityTaskRequest{
		DomainUUID:    DomainID,
		PollerID:      PollerID,
		PollRequest:   &PollForActivityTaskRequest,
		ForwardedFrom: ForwardedFrom,
	}
	MatchingPollForActivityTaskResponse = types.PollForActivityTaskResponse{
		TaskToken:                       TaskToken,
		WorkflowExecution:               &WorkflowExecution,
		ActivityID:                      ActivityID,
		ActivityType:                    &ActivityType,
		Input:                           Payload1,
		ScheduledTimestamp:              &Timestamp1,
		ScheduleToCloseTimeoutSeconds:   &Duration1,
		StartedTimestamp:                &Timestamp2,
		StartToCloseTimeoutSeconds:      &Duration2,
		HeartbeatTimeoutSeconds:         &Duration3,
		Attempt:                         Attempt,
		ScheduledTimestampOfThisAttempt: &Timestamp3,
		HeartbeatDetails:                Payload2,
		WorkflowType:                    &WorkflowType,
		WorkflowDomain:                  DomainName,
		Header:                          &Header,
	}
	MatchingPollForDecisionTaskRequest = types.MatchingPollForDecisionTaskRequest{
		DomainUUID:    DomainID,
		PollerID:      PollerID,
		PollRequest:   &PollForDecisionTaskRequest,
		ForwardedFrom: ForwardedFrom,
	}
	MatchingPollForDecisionTaskResponse = types.MatchingPollForDecisionTaskResponse{
		TaskToken:                 TaskToken,
		WorkflowExecution:         &WorkflowExecution,
		WorkflowType:              &WorkflowType,
		PreviousStartedEventID:    common.Int64Ptr(EventID1),
		StartedEventID:            EventID2,
		Attempt:                   Attempt,
		NextEventID:               EventID3,
		BacklogCountHint:          BacklogCountHint,
		StickyExecutionEnabled:    true,
		Query:                     &WorkflowQuery,
		DecisionInfo:              &TransientDecisionInfo,
		WorkflowExecutionTaskList: &TaskList,
		EventStoreVersion:         EventStoreVersion,
		BranchToken:               BranchToken,
		ScheduledTimestamp:        &Timestamp1,
		StartedTimestamp:          &Timestamp2,
		Queries:                   WorkflowQueryMap,
	}
	MatchingQueryWorkflowRequest = types.MatchingQueryWorkflowRequest{
		DomainUUID:    DomainID,
		TaskList:      &TaskList,
		QueryRequest:  &QueryWorkflowRequest,
		ForwardedFrom: ForwardedFrom,
	}
	MatchingQueryWorkflowResponse = types.QueryWorkflowResponse{
		QueryResult:   Payload1,
		QueryRejected: &QueryRejected,
	}
	MatchingRespondQueryTaskCompletedRequest = types.MatchingRespondQueryTaskCompletedRequest{
		DomainUUID:       DomainID,
		TaskList:         &TaskList,
		TaskID:           "TaskID",
		CompletedRequest: &RespondQueryTaskCompletedRequest,
	}
)
