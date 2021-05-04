// Copyright (c) 2017-2020 Uber Technologies Inc.
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

package types

import (
	"fmt"
	"strconv"
	"strings"
)

// AddActivityTaskRequest is an internal type (TBD...)
type AddActivityTaskRequest struct {
	DomainUUID                    string             `json:"domainUUID,omitempty"`
	Execution                     *WorkflowExecution `json:"execution,omitempty"`
	SourceDomainUUID              string             `json:"sourceDomainUUID,omitempty"`
	TaskList                      *TaskList          `json:"taskList,omitempty"`
	ScheduleID                    int64              `json:"scheduleId,omitempty"`
	ScheduleToStartTimeoutSeconds *int32             `json:"scheduleToStartTimeoutSeconds,omitempty"`
	Source                        *TaskSource        `json:"source,omitempty"`
	ForwardedFrom                 string             `json:"forwardedFrom,omitempty"`
}

// GetDomainUUID is an internal getter (TBD...)
func (v *AddActivityTaskRequest) GetDomainUUID() (o string) {
	if v != nil {
		return v.DomainUUID
	}
	return
}

// GetExecution is an internal getter (TBD...)
func (v *AddActivityTaskRequest) GetExecution() (o *WorkflowExecution) {
	if v != nil && v.Execution != nil {
		return v.Execution
	}
	return
}

// GetSourceDomainUUID is an internal getter (TBD...)
func (v *AddActivityTaskRequest) GetSourceDomainUUID() (o string) {
	if v != nil {
		return v.SourceDomainUUID
	}
	return
}

// GetTaskList is an internal getter (TBD...)
func (v *AddActivityTaskRequest) GetTaskList() (o *TaskList) {
	if v != nil && v.TaskList != nil {
		return v.TaskList
	}
	return
}

// GetScheduleID is an internal getter (TBD...)
func (v *AddActivityTaskRequest) GetScheduleID() (o int64) {
	if v != nil {
		return v.ScheduleID
	}
	return
}

// GetScheduleToStartTimeoutSeconds is an internal getter (TBD...)
func (v *AddActivityTaskRequest) GetScheduleToStartTimeoutSeconds() (o int32) {
	if v != nil && v.ScheduleToStartTimeoutSeconds != nil {
		return *v.ScheduleToStartTimeoutSeconds
	}
	return
}

// GetSource is an internal getter (TBD...)
func (v *AddActivityTaskRequest) GetSource() (o TaskSource) {
	if v != nil && v.Source != nil {
		return *v.Source
	}
	return
}

// GetForwardedFrom is an internal getter (TBD...)
func (v *AddActivityTaskRequest) GetForwardedFrom() (o string) {
	if v != nil {
		return v.ForwardedFrom
	}
	return
}

// AddDecisionTaskRequest is an internal type (TBD...)
type AddDecisionTaskRequest struct {
	DomainUUID                    string             `json:"domainUUID,omitempty"`
	Execution                     *WorkflowExecution `json:"execution,omitempty"`
	TaskList                      *TaskList          `json:"taskList,omitempty"`
	ScheduleID                    int64              `json:"scheduleId,omitempty"`
	ScheduleToStartTimeoutSeconds *int32             `json:"scheduleToStartTimeoutSeconds,omitempty"`
	Source                        *TaskSource        `json:"source,omitempty"`
	ForwardedFrom                 string             `json:"forwardedFrom,omitempty"`
}

// GetDomainUUID is an internal getter (TBD...)
func (v *AddDecisionTaskRequest) GetDomainUUID() (o string) {
	if v != nil {
		return v.DomainUUID
	}
	return
}

// GetExecution is an internal getter (TBD...)
func (v *AddDecisionTaskRequest) GetExecution() (o *WorkflowExecution) {
	if v != nil && v.Execution != nil {
		return v.Execution
	}
	return
}

// GetTaskList is an internal getter (TBD...)
func (v *AddDecisionTaskRequest) GetTaskList() (o *TaskList) {
	if v != nil && v.TaskList != nil {
		return v.TaskList
	}
	return
}

// GetScheduleID is an internal getter (TBD...)
func (v *AddDecisionTaskRequest) GetScheduleID() (o int64) {
	if v != nil {
		return v.ScheduleID
	}
	return
}

// GetScheduleToStartTimeoutSeconds is an internal getter (TBD...)
func (v *AddDecisionTaskRequest) GetScheduleToStartTimeoutSeconds() (o int32) {
	if v != nil && v.ScheduleToStartTimeoutSeconds != nil {
		return *v.ScheduleToStartTimeoutSeconds
	}
	return
}

// GetSource is an internal getter (TBD...)
func (v *AddDecisionTaskRequest) GetSource() (o TaskSource) {
	if v != nil && v.Source != nil {
		return *v.Source
	}
	return
}

// GetForwardedFrom is an internal getter (TBD...)
func (v *AddDecisionTaskRequest) GetForwardedFrom() (o string) {
	if v != nil {
		return v.ForwardedFrom
	}
	return
}

// CancelOutstandingPollRequest is an internal type (TBD...)
type CancelOutstandingPollRequest struct {
	DomainUUID   string    `json:"domainUUID,omitempty"`
	TaskListType *int32    `json:"taskListType,omitempty"`
	TaskList     *TaskList `json:"taskList,omitempty"`
	PollerID     string    `json:"pollerID,omitempty"`
}

// GetDomainUUID is an internal getter (TBD...)
func (v *CancelOutstandingPollRequest) GetDomainUUID() (o string) {
	if v != nil {
		return v.DomainUUID
	}
	return
}

// GetTaskListType is an internal getter (TBD...)
func (v *CancelOutstandingPollRequest) GetTaskListType() (o int32) {
	if v != nil && v.TaskListType != nil {
		return *v.TaskListType
	}
	return
}

// GetTaskList is an internal getter (TBD...)
func (v *CancelOutstandingPollRequest) GetTaskList() (o *TaskList) {
	if v != nil && v.TaskList != nil {
		return v.TaskList
	}
	return
}

// GetPollerID is an internal getter (TBD...)
func (v *CancelOutstandingPollRequest) GetPollerID() (o string) {
	if v != nil {
		return v.PollerID
	}
	return
}

// MatchingDescribeTaskListRequest is an internal type (TBD...)
type MatchingDescribeTaskListRequest struct {
	DomainUUID  string                   `json:"domainUUID,omitempty"`
	DescRequest *DescribeTaskListRequest `json:"descRequest,omitempty"`
}

// GetDomainUUID is an internal getter (TBD...)
func (v *MatchingDescribeTaskListRequest) GetDomainUUID() (o string) {
	if v != nil {
		return v.DomainUUID
	}
	return
}

// GetDescRequest is an internal getter (TBD...)
func (v *MatchingDescribeTaskListRequest) GetDescRequest() (o *DescribeTaskListRequest) {
	if v != nil && v.DescRequest != nil {
		return v.DescRequest
	}
	return
}

// MatchingListTaskListPartitionsRequest is an internal type (TBD...)
type MatchingListTaskListPartitionsRequest struct {
	Domain   string    `json:"domain,omitempty"`
	TaskList *TaskList `json:"taskList,omitempty"`
}

// GetDomain is an internal getter (TBD...)
func (v *MatchingListTaskListPartitionsRequest) GetDomain() (o string) {
	if v != nil {
		return v.Domain
	}
	return
}

// GetTaskList is an internal getter (TBD...)
func (v *MatchingListTaskListPartitionsRequest) GetTaskList() (o *TaskList) {
	if v != nil && v.TaskList != nil {
		return v.TaskList
	}
	return
}

// MatchingPollForActivityTaskRequest is an internal type (TBD...)
type MatchingPollForActivityTaskRequest struct {
	DomainUUID    string                      `json:"domainUUID,omitempty"`
	PollerID      string                      `json:"pollerID,omitempty"`
	PollRequest   *PollForActivityTaskRequest `json:"pollRequest,omitempty"`
	ForwardedFrom string                      `json:"forwardedFrom,omitempty"`
}

// GetDomainUUID is an internal getter (TBD...)
func (v *MatchingPollForActivityTaskRequest) GetDomainUUID() (o string) {
	if v != nil {
		return v.DomainUUID
	}
	return
}

// GetPollerID is an internal getter (TBD...)
func (v *MatchingPollForActivityTaskRequest) GetPollerID() (o string) {
	if v != nil {
		return v.PollerID
	}
	return
}

// GetPollRequest is an internal getter (TBD...)
func (v *MatchingPollForActivityTaskRequest) GetPollRequest() (o *PollForActivityTaskRequest) {
	if v != nil && v.PollRequest != nil {
		return v.PollRequest
	}
	return
}

// GetForwardedFrom is an internal getter (TBD...)
func (v *MatchingPollForActivityTaskRequest) GetForwardedFrom() (o string) {
	if v != nil {
		return v.ForwardedFrom
	}
	return
}

// MatchingPollForDecisionTaskRequest is an internal type (TBD...)
type MatchingPollForDecisionTaskRequest struct {
	DomainUUID    string                      `json:"domainUUID,omitempty"`
	PollerID      string                      `json:"pollerID,omitempty"`
	PollRequest   *PollForDecisionTaskRequest `json:"pollRequest,omitempty"`
	ForwardedFrom string                      `json:"forwardedFrom,omitempty"`
}

// GetDomainUUID is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskRequest) GetDomainUUID() (o string) {
	if v != nil {
		return v.DomainUUID
	}
	return
}

// GetPollerID is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskRequest) GetPollerID() (o string) {
	if v != nil {
		return v.PollerID
	}
	return
}

// GetPollRequest is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskRequest) GetPollRequest() (o *PollForDecisionTaskRequest) {
	if v != nil && v.PollRequest != nil {
		return v.PollRequest
	}
	return
}

// GetForwardedFrom is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskRequest) GetForwardedFrom() (o string) {
	if v != nil {
		return v.ForwardedFrom
	}
	return
}

// MatchingPollForDecisionTaskResponse is an internal type (TBD...)
type MatchingPollForDecisionTaskResponse struct {
	TaskToken                 []byte                    `json:"taskToken,omitempty"`
	WorkflowExecution         *WorkflowExecution        `json:"workflowExecution,omitempty"`
	WorkflowType              *WorkflowType             `json:"workflowType,omitempty"`
	PreviousStartedEventID    *int64                    `json:"previousStartedEventId,omitempty"`
	StartedEventID            int64                     `json:"startedEventId,omitempty"`
	Attempt                   int64                     `json:"attempt,omitempty"`
	NextEventID               int64                     `json:"nextEventId,omitempty"`
	BacklogCountHint          *int64                    `json:"backlogCountHint,omitempty"`
	StickyExecutionEnabled    bool                      `json:"stickyExecutionEnabled,omitempty"`
	Query                     *WorkflowQuery            `json:"query,omitempty"`
	DecisionInfo              *TransientDecisionInfo    `json:"decisionInfo,omitempty"`
	WorkflowExecutionTaskList *TaskList                 `json:"WorkflowExecutionTaskList,omitempty"`
	EventStoreVersion         *int32                    `json:"eventStoreVersion,omitempty"`
	BranchToken               []byte                    `json:"branchToken,omitempty"`
	ScheduledTimestamp        *int64                    `json:"scheduledTimestamp,omitempty"`
	StartedTimestamp          *int64                    `json:"startedTimestamp,omitempty"`
	Queries                   map[string]*WorkflowQuery `json:"queries,omitempty"`
}

// GetTaskToken is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetTaskToken() (o []byte) {
	if v != nil && v.TaskToken != nil {
		return v.TaskToken
	}
	return
}

// GetWorkflowExecution is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetWorkflowExecution() (o *WorkflowExecution) {
	if v != nil && v.WorkflowExecution != nil {
		return v.WorkflowExecution
	}
	return
}

// GetWorkflowType is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetWorkflowType() (o *WorkflowType) {
	if v != nil && v.WorkflowType != nil {
		return v.WorkflowType
	}
	return
}

// GetPreviousStartedEventID is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetPreviousStartedEventID() (o int64) {
	if v != nil && v.PreviousStartedEventID != nil {
		return *v.PreviousStartedEventID
	}
	return
}

// GetStartedEventID is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetStartedEventID() (o int64) {
	if v != nil {
		return v.StartedEventID
	}
	return
}

// GetAttempt is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetAttempt() (o int64) {
	if v != nil {
		return v.Attempt
	}
	return
}

// GetNextEventID is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetNextEventID() (o int64) {
	if v != nil {
		return v.NextEventID
	}
	return
}

// GetBacklogCountHint is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetBacklogCountHint() (o int64) {
	if v != nil && v.BacklogCountHint != nil {
		return *v.BacklogCountHint
	}
	return
}

// GetStickyExecutionEnabled is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetStickyExecutionEnabled() (o bool) {
	if v != nil {
		return v.StickyExecutionEnabled
	}
	return
}

// GetQuery is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetQuery() (o *WorkflowQuery) {
	if v != nil && v.Query != nil {
		return v.Query
	}
	return
}

// GetDecisionInfo is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetDecisionInfo() (o *TransientDecisionInfo) {
	if v != nil && v.DecisionInfo != nil {
		return v.DecisionInfo
	}
	return
}

// GetWorkflowExecutionTaskList is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetWorkflowExecutionTaskList() (o *TaskList) {
	if v != nil && v.WorkflowExecutionTaskList != nil {
		return v.WorkflowExecutionTaskList
	}
	return
}

// GetEventStoreVersion is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetEventStoreVersion() (o int32) {
	if v != nil && v.EventStoreVersion != nil {
		return *v.EventStoreVersion
	}
	return
}

// GetBranchToken is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetBranchToken() (o []byte) {
	if v != nil && v.BranchToken != nil {
		return v.BranchToken
	}
	return
}

// GetScheduledTimestamp is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetScheduledTimestamp() (o int64) {
	if v != nil && v.ScheduledTimestamp != nil {
		return *v.ScheduledTimestamp
	}
	return
}

// GetStartedTimestamp is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetStartedTimestamp() (o int64) {
	if v != nil && v.StartedTimestamp != nil {
		return *v.StartedTimestamp
	}
	return
}

// GetQueries is an internal getter (TBD...)
func (v *MatchingPollForDecisionTaskResponse) GetQueries() (o map[string]*WorkflowQuery) {
	if v != nil && v.Queries != nil {
		return v.Queries
	}
	return
}

// MatchingQueryWorkflowRequest is an internal type (TBD...)
type MatchingQueryWorkflowRequest struct {
	DomainUUID    string                `json:"domainUUID,omitempty"`
	TaskList      *TaskList             `json:"taskList,omitempty"`
	QueryRequest  *QueryWorkflowRequest `json:"queryRequest,omitempty"`
	ForwardedFrom string                `json:"forwardedFrom,omitempty"`
}

// GetDomainUUID is an internal getter (TBD...)
func (v *MatchingQueryWorkflowRequest) GetDomainUUID() (o string) {
	if v != nil {
		return v.DomainUUID
	}
	return
}

// GetTaskList is an internal getter (TBD...)
func (v *MatchingQueryWorkflowRequest) GetTaskList() (o *TaskList) {
	if v != nil && v.TaskList != nil {
		return v.TaskList
	}
	return
}

// GetQueryRequest is an internal getter (TBD...)
func (v *MatchingQueryWorkflowRequest) GetQueryRequest() (o *QueryWorkflowRequest) {
	if v != nil && v.QueryRequest != nil {
		return v.QueryRequest
	}
	return
}

// GetForwardedFrom is an internal getter (TBD...)
func (v *MatchingQueryWorkflowRequest) GetForwardedFrom() (o string) {
	if v != nil {
		return v.ForwardedFrom
	}
	return
}

// MatchingRespondQueryTaskCompletedRequest is an internal type (TBD...)
type MatchingRespondQueryTaskCompletedRequest struct {
	DomainUUID       string                            `json:"domainUUID,omitempty"`
	TaskList         *TaskList                         `json:"taskList,omitempty"`
	TaskID           *string                           `json:"taskID,omitempty"`
	CompletedRequest *RespondQueryTaskCompletedRequest `json:"completedRequest,omitempty"`
}

// GetDomainUUID is an internal getter (TBD...)
func (v *MatchingRespondQueryTaskCompletedRequest) GetDomainUUID() (o string) {
	if v != nil {
		return v.DomainUUID
	}
	return
}

// GetTaskList is an internal getter (TBD...)
func (v *MatchingRespondQueryTaskCompletedRequest) GetTaskList() (o *TaskList) {
	if v != nil && v.TaskList != nil {
		return v.TaskList
	}
	return
}

// GetTaskID is an internal getter (TBD...)
func (v *MatchingRespondQueryTaskCompletedRequest) GetTaskID() (o string) {
	if v != nil && v.TaskID != nil {
		return *v.TaskID
	}
	return
}

// GetCompletedRequest is an internal getter (TBD...)
func (v *MatchingRespondQueryTaskCompletedRequest) GetCompletedRequest() (o *RespondQueryTaskCompletedRequest) {
	if v != nil && v.CompletedRequest != nil {
		return v.CompletedRequest
	}
	return
}

// TaskSource is an internal type (TBD...)
type TaskSource int32

// Ptr is a helper function for getting pointer value
func (e TaskSource) Ptr() *TaskSource {
	return &e
}

// String returns a readable string representation of TaskSource.
func (e TaskSource) String() string {
	w := int32(e)
	switch w {
	case 0:
		return "HISTORY"
	case 1:
		return "DB_BACKLOG"
	}
	return fmt.Sprintf("TaskSource(%d)", w)
}

// UnmarshalText parses enum value from string representation
func (e *TaskSource) UnmarshalText(value []byte) error {
	switch s := strings.ToUpper(string(value)); s {
	case "HISTORY":
		*e = TaskSourceHistory
		return nil
	case "DB_BACKLOG":
		*e = TaskSourceDbBacklog
		return nil
	default:
		val, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return fmt.Errorf("unknown enum value %q for %q: %v", s, "TaskSource", err)
		}
		*e = TaskSource(val)
		return nil
	}
}

// MarshalText encodes TaskSource to text.
func (e TaskSource) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

const (
	// TaskSourceHistory is an option for TaskSource
	TaskSourceHistory TaskSource = iota
	// TaskSourceDbBacklog is an option for TaskSource
	TaskSourceDbBacklog
)
