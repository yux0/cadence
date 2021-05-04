// The MIT License (MIT)

// Copyright (c) 2017-2020 Uber Technologies Inc.

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

package serialization

import (
	"time"

	"github.com/uber/cadence/common/types"
)

// GetStolenSinceRenew internal sql blob getter
func (s *ShardInfo) GetStolenSinceRenew() (o int32) {
	if s != nil && s.StolenSinceRenew != nil {
		return *s.StolenSinceRenew
	}
	return
}

// GetUpdatedAt internal sql blob getter
func (s *ShardInfo) GetUpdatedAt() time.Time {
	if s != nil && s.UpdatedAt != nil {
		return *s.UpdatedAt
	}
	return time.Unix(0, 0)
}

// GetReplicationAckLevel internal sql blob getter
func (s *ShardInfo) GetReplicationAckLevel() (o int64) {
	if s != nil && s.ReplicationAckLevel != nil {
		return *s.ReplicationAckLevel
	}
	return
}

// GetTransferAckLevel internal sql blob getter
func (s *ShardInfo) GetTransferAckLevel() (o int64) {
	if s != nil && s.TransferAckLevel != nil {
		return *s.TransferAckLevel
	}
	return
}

// GetTimerAckLevel internal sql blob getter
func (s *ShardInfo) GetTimerAckLevel() time.Time {
	if s != nil && s.TimerAckLevel != nil {
		return *s.TimerAckLevel
	}
	return time.Unix(0, 0)
}

// GetDomainNotificationVersion internal sql blob getter
func (s *ShardInfo) GetDomainNotificationVersion() (o int64) {
	if s != nil && s.DomainNotificationVersion != nil {
		return *s.DomainNotificationVersion
	}
	return
}

// GetClusterTransferAckLevel internal sql blob getter
func (s *ShardInfo) GetClusterTransferAckLevel() (o map[string]int64) {
	if s != nil {
		return s.ClusterTransferAckLevel
	}
	return
}

// GetClusterTimerAckLevel internal sql blob getter
func (s *ShardInfo) GetClusterTimerAckLevel() (o map[string]time.Time) {
	if s != nil {
		return s.ClusterTimerAckLevel
	}
	return
}

// GetOwner internal sql blob getter
func (s *ShardInfo) GetOwner() (o string) {
	if s != nil && s.Owner != nil {
		return *s.Owner
	}
	return
}

// GetClusterReplicationLevel internal sql blob getter
func (s *ShardInfo) GetClusterReplicationLevel() (o map[string]int64) {
	if s != nil {
		return s.ClusterReplicationLevel
	}
	return
}

// GetPendingFailoverMarkers internal sql blob getter
func (s *ShardInfo) GetPendingFailoverMarkers() (o []byte) {
	if s != nil {
		return s.PendingFailoverMarkers
	}
	return
}

// GetPendingFailoverMarkersEncoding internal sql blob getter
func (s *ShardInfo) GetPendingFailoverMarkersEncoding() (o string) {
	if s != nil && s.PendingFailoverMarkersEncoding != nil {
		return *s.PendingFailoverMarkersEncoding
	}
	return
}

// GetReplicationDlqAckLevel internal sql blob getter
func (s *ShardInfo) GetReplicationDlqAckLevel() (o map[string]int64) {
	if s != nil {
		return s.ReplicationDlqAckLevel
	}
	return
}

// GetTransferProcessingQueueStates internal sql blob getter
func (s *ShardInfo) GetTransferProcessingQueueStates() (o []byte) {
	if s != nil {
		return s.TransferProcessingQueueStates
	}
	return
}

// GetTransferProcessingQueueStatesEncoding internal sql blob getter
func (s *ShardInfo) GetTransferProcessingQueueStatesEncoding() (o string) {
	if s != nil && s.TransferProcessingQueueStatesEncoding != nil {
		return *s.TransferProcessingQueueStatesEncoding
	}
	return
}

// GetTimerProcessingQueueStates internal sql blob getter
func (s *ShardInfo) GetTimerProcessingQueueStates() (o []byte) {
	if s != nil {
		return s.TimerProcessingQueueStates
	}
	return
}

// GetTimerProcessingQueueStatesEncoding internal sql blob getter
func (s *ShardInfo) GetTimerProcessingQueueStatesEncoding() (o string) {
	if s != nil && s.TimerProcessingQueueStatesEncoding != nil {
		return *s.TimerProcessingQueueStatesEncoding
	}
	return
}

// GetName internal sql blob getter
func (d *DomainInfo) GetName() (o string) {
	if d != nil && d.Name != nil {
		return *d.Name
	}
	return
}

// GetDescription internal sql blob getter
func (d *DomainInfo) GetDescription() (o string) {
	if d != nil && d.Description != nil {
		return *d.Description
	}
	return
}

// GetOwner internal sql blob getter
func (d *DomainInfo) GetOwner() (o string) {
	if d != nil && d.Owner != nil {
		return *d.Owner
	}
	return
}

// GetStatus internal sql blob getter
func (d *DomainInfo) GetStatus() (o int32) {
	if d != nil && d.Status != nil {
		return *d.Status
	}
	return
}

// GetRetention internal sql blob getter
func (d *DomainInfo) GetRetention() time.Duration {
	if d != nil && d.Retention != nil {
		return *d.Retention
	}
	return time.Duration(0)
}

// GetEmitMetric internal sql blob getter
func (d *DomainInfo) GetEmitMetric() (o bool) {
	if d != nil && d.EmitMetric != nil {
		return *d.EmitMetric
	}
	return
}

// GetArchivalBucket internal sql blob getter
func (d *DomainInfo) GetArchivalBucket() (o string) {
	if d != nil && d.ArchivalBucket != nil {
		return *d.ArchivalBucket
	}
	return
}

// GetArchivalStatus internal sql blob getter
func (d *DomainInfo) GetArchivalStatus() (o int16) {
	if d != nil && d.ArchivalStatus != nil {
		return *d.ArchivalStatus
	}
	return
}

// GetConfigVersion internal sql blob getter
func (d *DomainInfo) GetConfigVersion() (o int64) {
	if d != nil && d.ConfigVersion != nil {
		return *d.ConfigVersion
	}
	return
}

// GetNotificationVersion internal sql blob getter
func (d *DomainInfo) GetNotificationVersion() (o int64) {
	if d != nil && d.NotificationVersion != nil {
		return *d.NotificationVersion
	}
	return
}

// GetFailoverNotificationVersion internal sql blob getter
func (d *DomainInfo) GetFailoverNotificationVersion() (o int64) {
	if d != nil && d.FailoverNotificationVersion != nil {
		return *d.FailoverNotificationVersion
	}
	return
}

// GetFailoverVersion internal sql blob getter
func (d *DomainInfo) GetFailoverVersion() (o int64) {
	if d != nil && d.FailoverVersion != nil {
		return *d.FailoverVersion
	}
	return
}

// GetActiveClusterName internal sql blob getter
func (d *DomainInfo) GetActiveClusterName() (o string) {
	if d != nil && d.ActiveClusterName != nil {
		return *d.ActiveClusterName
	}
	return
}

// GetClusters internal sql blob getter
func (d *DomainInfo) GetClusters() (o []string) {
	if d != nil {
		return d.Clusters
	}
	return
}

// GetData internal sql blob getter
func (d *DomainInfo) GetData() (o map[string]string) {
	if d != nil {
		return d.Data
	}
	return
}

// GetBadBinaries internal sql blob getter
func (d *DomainInfo) GetBadBinaries() (o []byte) {
	if d != nil {
		return d.BadBinaries
	}
	return
}

// GetBadBinariesEncoding internal sql blob getter
func (d *DomainInfo) GetBadBinariesEncoding() (o string) {
	if d != nil && d.BadBinariesEncoding != nil {
		return *d.BadBinariesEncoding
	}
	return
}

// GetHistoryArchivalStatus internal sql blob getter
func (d *DomainInfo) GetHistoryArchivalStatus() (o int16) {
	if d != nil && d.HistoryArchivalStatus != nil {
		return *d.HistoryArchivalStatus
	}
	return
}

// GetHistoryArchivalURI internal sql blob getter
func (d *DomainInfo) GetHistoryArchivalURI() (o string) {
	if d != nil && d.HistoryArchivalURI != nil {
		return *d.HistoryArchivalURI
	}
	return
}

// GetVisibilityArchivalStatus internal sql blob getter
func (d *DomainInfo) GetVisibilityArchivalStatus() (o int16) {
	if d != nil && d.VisibilityArchivalStatus != nil {
		return *d.VisibilityArchivalStatus
	}
	return
}

// GetVisibilityArchivalURI internal sql blob getter
func (d *DomainInfo) GetVisibilityArchivalURI() (o string) {
	if d != nil && d.VisibilityArchivalURI != nil {
		return *d.VisibilityArchivalURI
	}
	return
}

// GetFailoverEndTimestamp internal sql blob getter
func (d *DomainInfo) GetFailoverEndTimestamp() time.Time {
	if d != nil && d.FailoverEndTimestamp != nil {
		return *d.FailoverEndTimestamp
	}
	return time.Unix(0, 0)
}

// GetPreviousFailoverVersion internal sql blob getter
func (d *DomainInfo) GetPreviousFailoverVersion() (o int64) {
	if d != nil && d.PreviousFailoverVersion != nil {
		return *d.PreviousFailoverVersion
	}
	return
}

// GetLastUpdatedTimestamp internal sql blob getter
func (d *DomainInfo) GetLastUpdatedTimestamp() time.Time {
	if d != nil && d.LastUpdatedTimestamp != nil {
		return *d.LastUpdatedTimestamp
	}
	return time.Unix(0, 0)
}

// GetCreatedTimestamp internal sql blob getter
func (h *HistoryTreeInfo) GetCreatedTimestamp() time.Time {
	if h != nil && h.CreatedTimestamp != nil {
		return *h.CreatedTimestamp
	}
	return time.Unix(0, 0)
}

// GetAncestors internal sql blob getter
func (h *HistoryTreeInfo) GetAncestors() (o []*types.HistoryBranchRange) {
	if h != nil {
		return h.Ancestors
	}
	return
}

// GetInfo internal sql blob getter
func (h *HistoryTreeInfo) GetInfo() (o string) {
	if h != nil && h.Info != nil {
		return *h.Info
	}
	return
}

// GetParentDomainID internal sql blob getter
func (w *WorkflowExecutionInfo) GetParentDomainID() (o []byte) {
	if w != nil {
		return w.ParentDomainID
	}
	return
}

// GetRetryBackoffCoefficient internal sql blob getter
func (w *WorkflowExecutionInfo) GetRetryBackoffCoefficient() (o float64) {
	if w != nil && w.RetryBackoffCoefficient != nil {
		return *w.RetryBackoffCoefficient
	}
	return
}

// GetParentWorkflowID internal sql blob getter
func (w *WorkflowExecutionInfo) GetParentWorkflowID() (o string) {
	if w != nil && w.ParentWorkflowID != nil {
		return *w.ParentWorkflowID
	}
	return
}

// GetParentRunID internal sql blob getter
func (w *WorkflowExecutionInfo) GetParentRunID() (o []byte) {
	if w != nil {
		return w.ParentRunID
	}
	return
}

// GetCompletionEventEncoding internal sql blob getter
func (w *WorkflowExecutionInfo) GetCompletionEventEncoding() (o string) {
	if w != nil && w.CompletionEventEncoding != nil {
		return *w.CompletionEventEncoding
	}
	return
}

// GetTaskList internal sql blob getter
func (w *WorkflowExecutionInfo) GetTaskList() (o string) {
	if w != nil && w.TaskList != nil {
		return *w.TaskList
	}
	return
}

// GetWorkflowTypeName internal sql blob getter
func (w *WorkflowExecutionInfo) GetWorkflowTypeName() (o string) {
	if w != nil && w.WorkflowTypeName != nil {
		return *w.WorkflowTypeName
	}
	return
}

// GetCreateRequestID internal sql blob getter
func (w *WorkflowExecutionInfo) GetCreateRequestID() (o string) {
	if w != nil && w.CreateRequestID != nil {
		return *w.CreateRequestID
	}
	return
}

// GetDecisionRequestID internal sql blob getter
func (w *WorkflowExecutionInfo) GetDecisionRequestID() (o string) {
	if w != nil && w.DecisionRequestID != nil {
		return *w.DecisionRequestID
	}
	return
}

// GetCancelRequestID internal sql blob getter
func (w *WorkflowExecutionInfo) GetCancelRequestID() (o string) {
	if w != nil && w.CancelRequestID != nil {
		return *w.CancelRequestID
	}
	return
}

// GetStickyTaskList internal sql blob getter
func (w *WorkflowExecutionInfo) GetStickyTaskList() (o string) {
	if w != nil && w.StickyTaskList != nil {
		return *w.StickyTaskList
	}
	return
}

// GetCronSchedule internal sql blob getter
func (w *WorkflowExecutionInfo) GetCronSchedule() (o string) {
	if w != nil && w.CronSchedule != nil {
		return *w.CronSchedule
	}
	return
}

// GetClientLibraryVersion internal sql blob getter
func (w *WorkflowExecutionInfo) GetClientLibraryVersion() (o string) {
	if w != nil && w.ClientLibraryVersion != nil {
		return *w.ClientLibraryVersion
	}
	return
}

// GetClientFeatureVersion internal sql blob getter
func (w *WorkflowExecutionInfo) GetClientFeatureVersion() (o string) {
	if w != nil && w.ClientFeatureVersion != nil {
		return *w.ClientFeatureVersion
	}
	return
}

// GetClientImpl internal sql blob getter
func (w *WorkflowExecutionInfo) GetClientImpl() (o string) {
	if w != nil && w.ClientImpl != nil {
		return *w.ClientImpl
	}
	return
}

// GetAutoResetPointsEncoding internal sql blob getter
func (w *WorkflowExecutionInfo) GetAutoResetPointsEncoding() (o string) {
	if w != nil && w.AutoResetPointsEncoding != nil {
		return *w.AutoResetPointsEncoding
	}
	return
}

// GetVersionHistoriesEncoding internal sql blob getter
func (w *WorkflowExecutionInfo) GetVersionHistoriesEncoding() (o string) {
	if w != nil && w.VersionHistoriesEncoding != nil {
		return *w.VersionHistoriesEncoding
	}
	return
}

// GetInitiatedID internal sql blob getter
func (w *WorkflowExecutionInfo) GetInitiatedID() (o int64) {
	if w != nil && w.InitiatedID != nil {
		return *w.InitiatedID
	}
	return
}

// GetCompletionEventBatchID internal sql blob getter
func (w *WorkflowExecutionInfo) GetCompletionEventBatchID() (o int64) {
	if w != nil && w.CompletionEventBatchID != nil {
		return *w.CompletionEventBatchID
	}
	return
}

// GetStartVersion internal sql blob getter
func (w *WorkflowExecutionInfo) GetStartVersion() (o int64) {
	if w != nil && w.StartVersion != nil {
		return *w.StartVersion
	}
	return
}

// GetLastWriteEventID internal sql blob getter
func (w *WorkflowExecutionInfo) GetLastWriteEventID() (o int64) {
	if w != nil && w.LastWriteEventID != nil {
		return *w.LastWriteEventID
	}
	return
}

// GetLastEventTaskID internal sql blob getter
func (w *WorkflowExecutionInfo) GetLastEventTaskID() (o int64) {
	if w != nil && w.LastEventTaskID != nil {
		return *w.LastEventTaskID
	}
	return
}

// GetLastFirstEventID internal sql blob getter
func (w *WorkflowExecutionInfo) GetLastFirstEventID() (o int64) {
	if w != nil && w.LastFirstEventID != nil {
		return *w.LastFirstEventID
	}
	return
}

// GetLastProcessedEvent internal sql blob getter
func (w *WorkflowExecutionInfo) GetLastProcessedEvent() (o int64) {
	if w != nil && w.LastProcessedEvent != nil {
		return *w.LastProcessedEvent
	}
	return
}

// GetDecisionVersion internal sql blob getter
func (w *WorkflowExecutionInfo) GetDecisionVersion() (o int64) {
	if w != nil && w.DecisionVersion != nil {
		return *w.DecisionVersion
	}
	return
}

// GetDecisionScheduleID internal sql blob getter
func (w *WorkflowExecutionInfo) GetDecisionScheduleID() (o int64) {
	if w != nil && w.DecisionScheduleID != nil {
		return *w.DecisionScheduleID
	}
	return
}

// GetDecisionStartedID internal sql blob getter
func (w *WorkflowExecutionInfo) GetDecisionStartedID() (o int64) {
	if w != nil && w.DecisionStartedID != nil {
		return *w.DecisionStartedID
	}
	return
}

// GetDecisionAttempt internal sql blob getter
func (w *WorkflowExecutionInfo) GetDecisionAttempt() (o int64) {
	if w != nil && w.DecisionAttempt != nil {
		return *w.DecisionAttempt
	}
	return
}

// GetRetryAttempt internal sql blob getter
func (w *WorkflowExecutionInfo) GetRetryAttempt() (o int64) {
	if w != nil && w.RetryAttempt != nil {
		return *w.RetryAttempt
	}
	return
}

// GetSignalCount internal sql blob getter
func (w *WorkflowExecutionInfo) GetSignalCount() (o int64) {
	if w != nil && w.SignalCount != nil {
		return *w.SignalCount
	}
	return
}

// GetHistorySize internal sql blob getter
func (w *WorkflowExecutionInfo) GetHistorySize() (o int64) {
	if w != nil && w.HistorySize != nil {
		return *w.HistorySize
	}
	return
}

// GetState internal sql blob getter
func (w *WorkflowExecutionInfo) GetState() (o int32) {
	if w != nil && w.State != nil {
		return *w.State
	}
	return
}

// GetCloseStatus internal sql blob getter
func (w *WorkflowExecutionInfo) GetCloseStatus() (o int32) {
	if w != nil && w.CloseStatus != nil {
		return *w.CloseStatus
	}
	return
}

// GetRetryMaximumAttempts internal sql blob getter
func (w *WorkflowExecutionInfo) GetRetryMaximumAttempts() (o int32) {
	if w != nil && w.RetryMaximumAttempts != nil {
		return *w.RetryMaximumAttempts
	}
	return
}

// GetEventStoreVersion internal sql blob getter
func (w *WorkflowExecutionInfo) GetEventStoreVersion() (o int32) {
	if w != nil && w.EventStoreVersion != nil {
		return *w.EventStoreVersion
	}
	return
}

// GetWorkflowTimeout internal sql blob getter
func (w *WorkflowExecutionInfo) GetWorkflowTimeout() time.Duration {
	if w != nil && w.WorkflowTimeout != nil {
		return *w.WorkflowTimeout
	}
	return time.Duration(0)
}

// GetDecisionTaskTimeout internal sql blob getter
func (w *WorkflowExecutionInfo) GetDecisionTaskTimeout() time.Duration {
	if w != nil && w.DecisionTaskTimeout != nil {
		return *w.DecisionTaskTimeout
	}
	return time.Duration(0)
}

// GetDecisionTimeout internal sql blob getter
func (w *WorkflowExecutionInfo) GetDecisionTimeout() time.Duration {
	if w != nil && w.DecisionTimeout != nil {
		return *w.DecisionTimeout
	}
	return time.Duration(0)
}

// GetStickyScheduleToStartTimeout internal sql blob getter
func (w *WorkflowExecutionInfo) GetStickyScheduleToStartTimeout() time.Duration {
	if w != nil && w.StickyScheduleToStartTimeout != nil {
		return *w.StickyScheduleToStartTimeout
	}
	return time.Duration(0)
}

// GetRetryInitialInterval internal sql blob getter
func (w *WorkflowExecutionInfo) GetRetryInitialInterval() time.Duration {
	if w != nil && w.RetryInitialInterval != nil {
		return *w.RetryInitialInterval
	}
	return time.Duration(0)
}

// GetRetryMaximumInterval internal sql blob getter
func (w *WorkflowExecutionInfo) GetRetryMaximumInterval() time.Duration {
	if w != nil && w.RetryMaximumInterval != nil {
		return *w.RetryMaximumInterval
	}
	return time.Duration(0)
}

// GetRetryExpiration internal sql blob getter
func (w *WorkflowExecutionInfo) GetRetryExpiration() time.Duration {
	if w != nil && w.RetryExpiration != nil {
		return *w.RetryExpiration
	}
	return time.Duration(0)
}

// GetStartTimestamp internal sql blob getter
func (w *WorkflowExecutionInfo) GetStartTimestamp() time.Time {
	if w != nil && w.StartTimestamp != nil {
		return *w.StartTimestamp
	}
	return time.Unix(0, 0)
}

// GetLastUpdatedTimestamp internal sql blob getter
func (w *WorkflowExecutionInfo) GetLastUpdatedTimestamp() time.Time {
	if w != nil && w.LastUpdatedTimestamp != nil {
		return *w.LastUpdatedTimestamp
	}
	return time.Unix(0, 0)
}

// GetDecisionStartedTimestamp internal sql blob getter
func (w *WorkflowExecutionInfo) GetDecisionStartedTimestamp() time.Time {
	if w != nil && w.DecisionStartedTimestamp != nil {
		return *w.DecisionStartedTimestamp
	}
	return time.Unix(0, 0)
}

// GetDecisionScheduledTimestamp internal sql blob getter
func (w *WorkflowExecutionInfo) GetDecisionScheduledTimestamp() time.Time {
	if w != nil && w.DecisionScheduledTimestamp != nil {
		return *w.DecisionScheduledTimestamp
	}
	return time.Unix(0, 0)
}

// GetDecisionOriginalScheduledTimestamp internal sql blob getter
func (w *WorkflowExecutionInfo) GetDecisionOriginalScheduledTimestamp() time.Time {
	if w != nil && w.DecisionOriginalScheduledTimestamp != nil {
		return *w.DecisionOriginalScheduledTimestamp
	}
	return time.Unix(0, 0)
}

// GetRetryExpirationTimestamp internal sql blob getter
func (w *WorkflowExecutionInfo) GetRetryExpirationTimestamp() time.Time {
	if w != nil && w.RetryExpirationTimestamp != nil {
		return *w.RetryExpirationTimestamp
	}
	return time.Unix(0, 0)
}

// GetCompletionEvent internal sql blob getter
func (w *WorkflowExecutionInfo) GetCompletionEvent() (o []byte) {
	if w != nil {
		return w.CompletionEvent
	}
	return
}

// GetExecutionContext internal sql blob getter
func (w *WorkflowExecutionInfo) GetExecutionContext() (o []byte) {
	if w != nil {
		return w.ExecutionContext
	}
	return
}

// GetEventBranchToken internal sql blob getter
func (w *WorkflowExecutionInfo) GetEventBranchToken() (o []byte) {
	if w != nil {
		return w.EventBranchToken
	}
	return
}

// GetAutoResetPoints internal sql blob getter
func (w *WorkflowExecutionInfo) GetAutoResetPoints() (o []byte) {
	if w != nil {
		return w.AutoResetPoints
	}
	return
}

// GetVersionHistories internal sql blob getter
func (w *WorkflowExecutionInfo) GetVersionHistories() (o []byte) {
	if w != nil {
		return w.VersionHistories
	}
	return
}

// GetMemo internal sql blob getter
func (w *WorkflowExecutionInfo) GetMemo() (o map[string][]byte) {
	if w != nil {
		return w.Memo
	}
	return
}

// GetSearchAttributes internal sql blob getter
func (w *WorkflowExecutionInfo) GetSearchAttributes() (o map[string][]byte) {
	if w != nil {
		return w.SearchAttributes
	}
	return
}

// GetRetryNonRetryableErrors internal sql blob getter
func (w *WorkflowExecutionInfo) GetRetryNonRetryableErrors() (o []string) {
	if w != nil {
		return w.RetryNonRetryableErrors
	}
	return
}

// GetCancelRequested internal sql blob getter
func (w *WorkflowExecutionInfo) GetCancelRequested() (o bool) {
	if w != nil && w.CancelRequested != nil {
		return *w.CancelRequested
	}
	return
}

// GetHasRetryPolicy internal sql blob getter
func (w *WorkflowExecutionInfo) GetHasRetryPolicy() (o bool) {
	if w != nil && w.HasRetryPolicy != nil {
		return *w.HasRetryPolicy
	}
	return
}

// GetVersion internal sql blob getter
func (a *ActivityInfo) GetVersion() (o int64) {
	if a != nil && a.Version != nil {
		return *a.Version
	}
	return
}

// GetScheduledEventBatchID internal sql blob getter
func (a *ActivityInfo) GetScheduledEventBatchID() (o int64) {
	if a != nil && a.ScheduledEventBatchID != nil {
		return *a.ScheduledEventBatchID
	}
	return
}

// GetStartedID internal sql blob getter
func (a *ActivityInfo) GetStartedID() (o int64) {
	if a != nil && a.StartedID != nil {
		return *a.StartedID
	}
	return
}

// GetCancelRequestID internal sql blob getter
func (a *ActivityInfo) GetCancelRequestID() (o int64) {
	if a != nil && a.CancelRequestID != nil {
		return *a.CancelRequestID
	}
	return
}

// GetTimerTaskStatus internal sql blob getter
func (a *ActivityInfo) GetTimerTaskStatus() (o int32) {
	if a != nil && a.TimerTaskStatus != nil {
		return *a.TimerTaskStatus
	}
	return
}

// GetScheduledEventEncoding internal sql blob getter
func (a *ActivityInfo) GetScheduledEventEncoding() (o string) {
	if a != nil && a.ScheduledEventEncoding != nil {
		return *a.ScheduledEventEncoding
	}
	return
}

// GetStartedIdentity internal sql blob getter
func (a *ActivityInfo) GetStartedIdentity() (o string) {
	if a != nil && a.StartedIdentity != nil {
		return *a.StartedIdentity
	}
	return
}

// GetRetryLastFailureReason internal sql blob getter
func (a *ActivityInfo) GetRetryLastFailureReason() (o string) {
	if a != nil && a.RetryLastFailureReason != nil {
		return *a.RetryLastFailureReason
	}
	return
}

// GetRetryLastWorkerIdentity internal sql blob getter
func (a *ActivityInfo) GetRetryLastWorkerIdentity() (o string) {
	if a != nil && a.RetryLastWorkerIdentity != nil {
		return *a.RetryLastWorkerIdentity
	}
	return
}

// GetTaskList internal sql blob getter
func (a *ActivityInfo) GetTaskList() (o string) {
	if a != nil && a.TaskList != nil {
		return *a.TaskList
	}
	return
}

// GetStartedEventEncoding internal sql blob getter
func (a *ActivityInfo) GetStartedEventEncoding() (o string) {
	if a != nil && a.StartedEventEncoding != nil {
		return *a.StartedEventEncoding
	}
	return
}

// GetActivityID internal sql blob getter
func (a *ActivityInfo) GetActivityID() (o string) {
	if a != nil && a.ActivityID != nil {
		return *a.ActivityID
	}
	return
}

// GetRequestID internal sql blob getter
func (a *ActivityInfo) GetRequestID() (o string) {
	if a != nil && a.RequestID != nil {
		return *a.RequestID
	}
	return
}

// GetAttempt internal sql blob getter
func (a *ActivityInfo) GetAttempt() (o int32) {
	if a != nil && a.Attempt != nil {
		return *a.Attempt
	}
	return
}

// GetRetryMaximumAttempts internal sql blob getter
func (a *ActivityInfo) GetRetryMaximumAttempts() (o int32) {
	if a != nil && a.RetryMaximumAttempts != nil {
		return *a.RetryMaximumAttempts
	}
	return
}

// GetScheduledTimestamp internal sql blob getter
func (a *ActivityInfo) GetScheduledTimestamp() time.Time {
	if a != nil && a.ScheduledTimestamp != nil {
		return *a.ScheduledTimestamp
	}
	return time.Unix(0, 0)
}

// GetStartedTimestamp internal sql blob getter
func (a *ActivityInfo) GetStartedTimestamp() time.Time {
	if a != nil && a.StartedTimestamp != nil {
		return *a.StartedTimestamp
	}
	return time.Unix(0, 0)
}

// GetRetryExpirationTimestamp internal sql blob getter
func (a *ActivityInfo) GetRetryExpirationTimestamp() time.Time {
	if a != nil && a.RetryExpirationTimestamp != nil {
		return *a.RetryExpirationTimestamp
	}
	return time.Unix(0, 0)
}

// GetScheduleToStartTimeout internal sql blob getter
func (a *ActivityInfo) GetScheduleToStartTimeout() time.Duration {
	if a != nil && a.ScheduleToStartTimeout != nil {
		return *a.ScheduleToStartTimeout
	}
	return time.Duration(0)
}

// GetScheduleToCloseTimeout internal sql blob getter
func (a *ActivityInfo) GetScheduleToCloseTimeout() time.Duration {
	if a != nil && a.ScheduleToCloseTimeout != nil {
		return *a.ScheduleToCloseTimeout
	}
	return time.Duration(0)
}

// GetStartToCloseTimeout internal sql blob getter
func (a *ActivityInfo) GetStartToCloseTimeout() time.Duration {
	if a != nil && a.StartToCloseTimeout != nil {
		return *a.StartToCloseTimeout
	}
	return time.Duration(0)
}

// GetHeartbeatTimeout internal sql blob getter
func (a *ActivityInfo) GetHeartbeatTimeout() time.Duration {
	if a != nil && a.HeartbeatTimeout != nil {
		return *a.HeartbeatTimeout
	}
	return time.Duration(0)
}

// GetRetryInitialInterval internal sql blob getter
func (a *ActivityInfo) GetRetryInitialInterval() time.Duration {
	if a != nil && a.RetryInitialInterval != nil {
		return *a.RetryInitialInterval
	}
	return time.Duration(0)
}

// GetRetryMaximumInterval internal sql blob getter
func (a *ActivityInfo) GetRetryMaximumInterval() time.Duration {
	if a != nil && a.RetryMaximumInterval != nil {
		return *a.RetryMaximumInterval
	}
	return time.Duration(0)
}

// GetScheduledEvent internal sql blob getter
func (a *ActivityInfo) GetScheduledEvent() (o []byte) {
	if a != nil {
		return a.ScheduledEvent
	}
	return
}

// GetStartedEvent internal sql blob getter
func (a *ActivityInfo) GetStartedEvent() (o []byte) {
	if a != nil {
		return a.StartedEvent
	}
	return
}

// GetRetryLastFailureDetails internal sql blob getter
func (a *ActivityInfo) GetRetryLastFailureDetails() (o []byte) {
	if a != nil {
		return a.RetryLastFailureDetails
	}
	return
}

// GetCancelRequested internal sql blob getter
func (a *ActivityInfo) GetCancelRequested() (o bool) {
	if a != nil && a.CancelRequested != nil {
		return *a.CancelRequested
	}
	return
}

// GetHasRetryPolicy internal sql blob getter
func (a *ActivityInfo) GetHasRetryPolicy() (o bool) {
	if a != nil && a.HasRetryPolicy != nil {
		return *a.HasRetryPolicy
	}
	return
}

// GetRetryBackoffCoefficient internal sql blob getter
func (a *ActivityInfo) GetRetryBackoffCoefficient() (o float64) {
	if a != nil && a.RetryBackoffCoefficient != nil {
		return *a.RetryBackoffCoefficient
	}
	return
}

// GetRetryNonRetryableErrors internal sql blob getter
func (a *ActivityInfo) GetRetryNonRetryableErrors() (o []string) {
	if a != nil {
		return a.RetryNonRetryableErrors
	}
	return
}

// GetVersion internal sql blob getter
func (c *ChildExecutionInfo) GetVersion() (o int64) {
	if c != nil && c.Version != nil {
		return *c.Version
	}
	return
}

// GetInitiatedEventBatchID internal sql blob getter
func (c *ChildExecutionInfo) GetInitiatedEventBatchID() (o int64) {
	if c != nil && c.InitiatedEventBatchID != nil {
		return *c.InitiatedEventBatchID
	}
	return
}

// GetStartedID internal sql blob getter
func (c *ChildExecutionInfo) GetStartedID() (o int64) {
	if c != nil && c.StartedID != nil {
		return *c.StartedID
	}
	return
}

// GetParentClosePolicy internal sql blob getter
func (c *ChildExecutionInfo) GetParentClosePolicy() (o int32) {
	if c != nil && c.ParentClosePolicy != nil {
		return *c.ParentClosePolicy
	}
	return
}

// GetInitiatedEventEncoding internal sql blob getter
func (c *ChildExecutionInfo) GetInitiatedEventEncoding() (o string) {
	if c != nil && c.InitiatedEventEncoding != nil {
		return *c.InitiatedEventEncoding
	}
	return
}

// GetStartedWorkflowID internal sql blob getter
func (c *ChildExecutionInfo) GetStartedWorkflowID() (o string) {
	if c != nil && c.StartedWorkflowID != nil {
		return *c.StartedWorkflowID
	}
	return
}

// GetStartedRunID internal sql blob getter
func (c *ChildExecutionInfo) GetStartedRunID() (o []byte) {
	if c != nil {
		return c.StartedRunID
	}
	return
}

// GetStartedEventEncoding internal sql blob getter
func (c *ChildExecutionInfo) GetStartedEventEncoding() (o string) {
	if c != nil && c.StartedEventEncoding != nil {
		return *c.StartedEventEncoding
	}
	return
}

// GetCreateRequestID internal sql blob getter
func (c *ChildExecutionInfo) GetCreateRequestID() (o string) {
	if c != nil && c.CreateRequestID != nil {
		return *c.CreateRequestID
	}
	return
}

// GetDomainName internal sql blob getter
func (c *ChildExecutionInfo) GetDomainName() (o string) {
	if c != nil && c.DomainName != nil {
		return *c.DomainName
	}
	return
}

// GetWorkflowTypeName internal sql blob getter
func (c *ChildExecutionInfo) GetWorkflowTypeName() (o string) {
	if c != nil && c.WorkflowTypeName != nil {
		return *c.WorkflowTypeName
	}
	return
}

// GetInitiatedEvent internal sql blob getter
func (c *ChildExecutionInfo) GetInitiatedEvent() (o []byte) {
	if c != nil {
		return c.InitiatedEvent
	}
	return
}

// GetStartedEvent internal sql blob getter
func (c *ChildExecutionInfo) GetStartedEvent() (o []byte) {
	if c != nil {
		return c.StartedEvent
	}
	return
}

// GetVersion internal sql blob getter
func (s *SignalInfo) GetVersion() (o int64) {
	if s != nil && s.Version != nil {
		return *s.Version
	}
	return
}

// GetInitiatedEventBatchID internal sql blob getter
func (s *SignalInfo) GetInitiatedEventBatchID() (o int64) {
	if s != nil && s.InitiatedEventBatchID != nil {
		return *s.InitiatedEventBatchID
	}
	return
}

// GetRequestID internal sql blob getter
func (s *SignalInfo) GetRequestID() (o string) {
	if s != nil && s.RequestID != nil {
		return *s.RequestID
	}
	return
}

// GetName internal sql blob getter
func (s *SignalInfo) GetName() (o string) {
	if s != nil && s.Name != nil {
		return *s.Name
	}
	return
}

// GetInput internal sql blob getter
func (s *SignalInfo) GetInput() (o []byte) {
	if s != nil {
		return s.Input
	}
	return
}

// GetControl internal sql blob getter
func (s *SignalInfo) GetControl() (o []byte) {
	if s != nil {
		return s.Control
	}
	return
}

// GetVersion internal sql blob getter
func (r *RequestCancelInfo) GetVersion() (o int64) {
	if r != nil && r.Version != nil {
		return *r.Version
	}
	return
}

// GetInitiatedEventBatchID internal sql blob getter
func (r *RequestCancelInfo) GetInitiatedEventBatchID() (o int64) {
	if r != nil && r.InitiatedEventBatchID != nil {
		return *r.InitiatedEventBatchID
	}
	return
}

// GetCancelRequestID internal sql blob getter
func (r *RequestCancelInfo) GetCancelRequestID() (o string) {
	if r != nil && r.CancelRequestID != nil {
		return *r.CancelRequestID
	}
	return
}

// GetVersion internal sql blob getter
func (t *TimerInfo) GetVersion() (o int64) {
	if t != nil && t.Version != nil {
		return *t.Version
	}
	return
}

// GetStartedID internal sql blob getter
func (t *TimerInfo) GetStartedID() (o int64) {
	if t != nil && t.StartedID != nil {
		return *t.StartedID
	}
	return
}

// GetTaskID internal sql blob getter
func (t *TimerInfo) GetTaskID() (o int64) {
	if t != nil && t.TaskID != nil {
		return *t.TaskID
	}
	return
}

// GetExpiryTimestamp internal sql blob getter
func (t *TimerInfo) GetExpiryTimestamp() (o time.Time) {
	if t != nil && t.ExpiryTimestamp != nil {
		return *t.ExpiryTimestamp
	}
	return time.Unix(0, 0)
}

// GetWorkflowID internal sql blob getter
func (t *TaskInfo) GetWorkflowID() (o string) {
	if t != nil && t.WorkflowID != nil {
		return *t.WorkflowID
	}
	return
}

// GetRunID internal sql blob getter
func (t *TaskInfo) GetRunID() (o []byte) {
	if t != nil {
		return t.RunID
	}
	return
}

// GetScheduleID internal sql blob getter
func (t *TaskInfo) GetScheduleID() (o int64) {
	if t != nil && t.ScheduleID != nil {
		return *t.ScheduleID
	}
	return
}

// GetExpiryTimestamp internal sql blob getter
func (t *TaskInfo) GetExpiryTimestamp() time.Time {
	if t != nil && t.ExpiryTimestamp != nil {
		return *t.ExpiryTimestamp
	}
	return time.Unix(0, 0)
}

// GetCreatedTimestamp internal sql blob getter
func (t *TaskInfo) GetCreatedTimestamp() time.Time {
	if t != nil && t.CreatedTimestamp != nil {
		return *t.CreatedTimestamp
	}
	return time.Unix(0, 0)
}

// GetKind internal sql blob getter
func (t *TaskListInfo) GetKind() (o int16) {
	if t != nil && t.Kind != nil {
		return *t.Kind
	}
	return
}

// GetAckLevel internal sql blob getter
func (t *TaskListInfo) GetAckLevel() (o int64) {
	if t != nil && t.AckLevel != nil {
		return *t.AckLevel
	}
	return
}

// GetExpiryTimestamp internal sql blob getter
func (t *TaskListInfo) GetExpiryTimestamp() time.Time {
	if t != nil && t.ExpiryTimestamp != nil {
		return *t.ExpiryTimestamp
	}
	return time.Unix(0, 0)
}

// GetLastUpdated internal sql blob getter
func (t *TaskListInfo) GetLastUpdated() time.Time {
	if t != nil && t.LastUpdated != nil {
		return *t.LastUpdated
	}
	return time.Unix(0, 0)
}

// GetDomainID internal sql blob getter
func (t *TransferTaskInfo) GetDomainID() (o []byte) {
	if t != nil {
		return t.DomainID
	}
	return
}

// GetWorkflowID internal sql blob getter
func (t *TransferTaskInfo) GetWorkflowID() (o string) {
	if t != nil && t.WorkflowID != nil {
		return *t.WorkflowID
	}
	return
}

// GetRunID internal sql blob getter
func (t *TransferTaskInfo) GetRunID() (o []byte) {
	if t != nil {
		return t.RunID
	}
	return
}

// GetTaskType internal sql blob getter
func (t *TransferTaskInfo) GetTaskType() (o int16) {
	if t != nil && t.TaskType != nil {
		return *t.TaskType
	}
	return
}

// GetTargetDomainID internal sql blob getter
func (t *TransferTaskInfo) GetTargetDomainID() (o []byte) {
	if t != nil {
		return t.TargetDomainID
	}
	return
}

// GetTargetWorkflowID internal sql blob getter
func (t *TransferTaskInfo) GetTargetWorkflowID() (o string) {
	if t != nil && t.TargetWorkflowID != nil {
		return *t.TargetWorkflowID
	}
	return
}

// GetTargetRunID internal sql blob getter
func (t *TransferTaskInfo) GetTargetRunID() (o []byte) {
	if t != nil {
		return t.TargetRunID
	}
	return
}

// GetTaskList internal sql blob getter
func (t *TransferTaskInfo) GetTaskList() (o string) {
	if t != nil && t.TaskList != nil {
		return *t.TaskList
	}
	return
}

// GetTargetChildWorkflowOnly internal sql blob getter
func (t *TransferTaskInfo) GetTargetChildWorkflowOnly() (o bool) {
	if t != nil && t.TargetChildWorkflowOnly != nil {
		return *t.TargetChildWorkflowOnly
	}
	return
}

// GetScheduleID internal sql blob getter
func (t *TransferTaskInfo) GetScheduleID() (o int64) {
	if t != nil && t.ScheduleID != nil {
		return *t.ScheduleID
	}
	return
}

// GetVersion internal sql blob getter
func (t *TransferTaskInfo) GetVersion() (o int64) {
	if t != nil && t.Version != nil {
		return *t.Version
	}
	return
}

// GetVisibilityTimestamp internal sql blob getter
func (t *TransferTaskInfo) GetVisibilityTimestamp() time.Time {
	if t != nil && t.VisibilityTimestamp != nil {
		return *t.VisibilityTimestamp
	}
	return time.Unix(0, 0)
}

// GetDomainID internal sql blob getter
func (t *TimerTaskInfo) GetDomainID() (o []byte) {
	if t != nil && t.DomainID != nil {
		return t.DomainID
	}
	return
}

// GetWorkflowID internal sql blob getter
func (t *TimerTaskInfo) GetWorkflowID() (o string) {
	if t != nil && t.WorkflowID != nil {
		return *t.WorkflowID
	}
	return
}

// GetRunID internal sql blob getter
func (t *TimerTaskInfo) GetRunID() (o []byte) {
	if t != nil && t.RunID != nil {
		return t.RunID
	}
	return
}

// GetTaskType internal sql blob getter
func (t *TimerTaskInfo) GetTaskType() (o int16) {
	if t != nil && t.TaskType != nil {
		return *t.TaskType
	}
	return
}

// GetTimeoutType internal sql blob getter
func (t *TimerTaskInfo) GetTimeoutType() (o int16) {
	if t != nil && t.TimeoutType != nil {
		return *t.TimeoutType
	}
	return
}

// GetVersion internal sql blob getter
func (t *TimerTaskInfo) GetVersion() (o int64) {
	if t != nil && t.Version != nil {
		return *t.Version
	}
	return
}

// GetScheduleAttempt internal sql blob getter
func (t *TimerTaskInfo) GetScheduleAttempt() (o int64) {
	if t != nil && t.ScheduleAttempt != nil {
		return *t.ScheduleAttempt
	}
	return
}

// GetEventID internal sql blob getter
func (t *TimerTaskInfo) GetEventID() (o int64) {
	if t != nil && t.EventID != nil {
		return *t.EventID
	}
	return
}

// GetDomainID internal sql blob getter
func (t *ReplicationTaskInfo) GetDomainID() (o []byte) {
	if t != nil {
		return t.DomainID
	}
	return
}

// GetWorkflowID internal sql blob getter
func (t *ReplicationTaskInfo) GetWorkflowID() (o string) {
	if t != nil && t.WorkflowID != nil {
		return *t.WorkflowID
	}
	return
}

// GetRunID internal sql blob getter
func (t *ReplicationTaskInfo) GetRunID() (o []byte) {
	if t != nil {
		return t.RunID
	}
	return
}

// GetTaskType internal sql blob getter
func (t *ReplicationTaskInfo) GetTaskType() (o int16) {
	if t != nil && t.TaskType != nil {
		return *t.TaskType
	}
	return
}

// GetVersion internal sql blob getter
func (t *ReplicationTaskInfo) GetVersion() (o int64) {
	if t != nil && t.Version != nil {
		return *t.Version
	}
	return
}

// GetFirstEventID internal sql blob getter
func (t *ReplicationTaskInfo) GetFirstEventID() (o int64) {
	if t != nil && t.FirstEventID != nil {
		return *t.FirstEventID
	}
	return
}

// GetNextEventID internal sql blob getter
func (t *ReplicationTaskInfo) GetNextEventID() (o int64) {
	if t != nil && t.NextEventID != nil {
		return *t.NextEventID
	}
	return
}

// GetScheduledID internal sql blob getter
func (t *ReplicationTaskInfo) GetScheduledID() (o int64) {
	if t != nil && t.ScheduledID != nil {
		return *t.ScheduledID
	}
	return
}

// GetEventStoreVersion internal sql blob getter
func (t *ReplicationTaskInfo) GetEventStoreVersion() (o int32) {
	if t != nil && t.EventStoreVersion != nil {
		return *t.EventStoreVersion
	}
	return
}

// GetNewRunEventStoreVersion internal sql blob getter
func (t *ReplicationTaskInfo) GetNewRunEventStoreVersion() (o int32) {
	if t != nil && t.NewRunEventStoreVersion != nil {
		return *t.NewRunEventStoreVersion
	}
	return
}

// GetBranchToken internal sql blob getter
func (t *ReplicationTaskInfo) GetBranchToken() (o []byte) {
	if t != nil {
		return t.BranchToken
	}
	return
}

// GetNewRunBranchToken internal sql blob getter
func (t *ReplicationTaskInfo) GetNewRunBranchToken() (o []byte) {
	if t != nil {
		return t.NewRunBranchToken
	}
	return
}

// GetCreationTimestamp internal sql blob getter
func (t *ReplicationTaskInfo) GetCreationTimestamp() time.Time {
	if t != nil && t.CreationTimestamp != nil {
		return *t.CreationTimestamp
	}
	return time.Unix(0, 0)
}
