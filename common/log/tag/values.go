// Copyright (c) 2017 Uber Technologies, Inc.
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

package tag

// Pre-defined values for TagWorkflowAction
var (
	// workflow start / finish
	WorkflowActionWorkflowStarted       = workflowAction("add-workflow-started-event")
	WorkflowActionWorkflowCanceled      = workflowAction("add-workflow-canceled-event")
	WorkflowActionWorkflowCompleted     = workflowAction("add-workflow-completed--event")
	WorkflowActionWorkflowFailed        = workflowAction("add-workflow-failed-event")
	WorkflowActionWorkflowTimeout       = workflowAction("add-workflow-timeout-event")
	WorkflowActionWorkflowTerminated    = workflowAction("add-workflow-terminated-event")
	WorkflowActionWorkflowContinueAsNew = workflowAction("add-workflow-continue-as-new-event")

	// workflow cancellation / sign
	WorkflowActionWorkflowCancelRequested        = workflowAction("add-workflow-cancel-requested-event")
	WorkflowActionWorkflowSignaled               = workflowAction("add-workflow-signaled-event")
	WorkflowActionWorkflowRecordMarker           = workflowAction("add-workflow-marker-record-event")
	WorkflowActionUpsertWorkflowSearchAttributes = workflowAction("add-workflow-upsert-search-attributes-event")

	// decision
	WorkflowActionDecisionTaskScheduled = workflowAction("add-decisiontask-scheduled-event")
	WorkflowActionDecisionTaskStarted   = workflowAction("add-decisiontask-started-event")
	WorkflowActionDecisionTaskCompleted = workflowAction("add-decisiontask-completed-event")
	WorkflowActionDecisionTaskTimedOut  = workflowAction("add-decisiontask-timedout-event")
	WorkflowActionDecisionTaskFailed    = workflowAction("add-decisiontask-failed-event")

	// in memory decision
	WorkflowActionInMemoryDecisionTaskScheduled = workflowAction("add-in-memory-decisiontask-scheduled")
	WorkflowActionInMemoryDecisionTaskStarted   = workflowAction("add-in-memory-decisiontask-started")

	// activity
	WorkflowActionActivityTaskScheduled       = workflowAction("add-activitytask-scheduled-event")
	WorkflowActionActivityTaskStarted         = workflowAction("add-activitytask-started-event")
	WorkflowActionActivityTaskCompleted       = workflowAction("add-activitytask-completed-event")
	WorkflowActionActivityTaskFailed          = workflowAction("add-activitytask-failed-event")
	WorkflowActionActivityTaskTimedOut        = workflowAction("add-activitytask-timed-event")
	WorkflowActionActivityTaskCanceled        = workflowAction("add-activitytask-canceled-event")
	WorkflowActionActivityTaskCancelRequested = workflowAction("add-activitytask-cancel-requested-event")
	WorkflowActionActivityTaskCancelFailed    = workflowAction("add-activitytask-cancel-failed-event")
	WorkflowActionActivityTaskRetry           = workflowAction("add-activitytask-retry-event")

	// timer
	WorkflowActionTimerStarted      = workflowAction("add-timer-started-event")
	WorkflowActionTimerFired        = workflowAction("add-timer-fired-event")
	WorkflowActionTimerCanceled     = workflowAction("add-timer-canceled-event")
	WorkflowActionTimerCancelFailed = workflowAction("add-timer-cancel-failed-event")

	// child workflow start / finish
	WorkflowActionChildWorkflowInitiated        = workflowAction("add-childworkflow-initiated-event")
	WorkflowActionChildWorkflowStarted          = workflowAction("add-childworkflow-started-event")
	WorkflowActionChildWorkflowInitiationFailed = workflowAction("add-childworkflow-initiation-failed-event")
	WorkflowActionChildWorkflowCanceled         = workflowAction("add-childworkflow-canceled-event")
	WorkflowActionChildWorkflowCompleted        = workflowAction("add-childworkflow-completed-event")
	WorkflowActionChildWorkflowFailed           = workflowAction("add-childworkflow-failed-event")
	WorkflowActionChildWorkflowTerminated       = workflowAction("add-childworkflow-terminated-event")
	WorkflowActionChildWorkflowTimedOut         = workflowAction("add-childworkflow-timedout-event")

	// external workflow cancellation
	WorkflowActionExternalWorkflowCancelInitiated = workflowAction("add-externalworkflow-cancel-initiated-event")
	WorkflowActionExternalWorkflowCancelRequested = workflowAction("add-externalworkflow-cancel-requested-event")
	WorkflowActionExternalWorkflowCancelFailed    = workflowAction("add-externalworkflow-cancel-failed-event")

	// external workflow signal
	WorkflowActionExternalWorkflowSignalInitiated = workflowAction("add-externalworkflow-signal-initiated-event")
	WorkflowActionExternalWorkflowSignalRequested = workflowAction("add-externalworkflow-signal-requested-event")
	WorkflowActionExternalWorkflowSignalFailed    = workflowAction("add-externalworkflow-signal-failed-event")

	WorkflowActionUnknown = workflowAction("add-unknown-event")
)

// Pre-defined values for TagWorkflowListFilterType
var (
	WorkflowListWorkflowFilterByID     = workflowListFilterType("WID")
	WorkflowListWorkflowFilterByType   = workflowListFilterType("WType")
	WorkflowListWorkflowFilterByStatus = workflowListFilterType("status")
)

// Pre-defined values for TagSysComponent
var (
	ComponentTaskList                 = component("tasklist")
	ComponentHistoryEngine            = component("history-engine")
	ComponentHistoryCache             = component("history-cache")
	ComponentDecisionHandler          = component("decision-handler")
	ComponentEventsCache              = component("events-cache")
	ComponentTransferQueue            = component("transfer-queue-processor")
	ComponentTimerQueue               = component("timer-queue-processor")
	ComponentTimerBuilder             = component("timer-builder")
	ComponentReplicatorQueue          = component("replicator-queue-processor")
	ComponentShardController          = component("shard-controller")
	ComponentShard                    = component("shard")
	ComponentShardItem                = component("shard-item")
	ComponentShardEngine              = component("shard-engine")
	ComponentMatchingEngine           = component("matching-engine")
	ComponentReplicator               = component("replicator")
	ComponentReplicationTaskProcessor = component("replication-task-processor")
	ComponentReplicationAckManager    = component("replication-ack-manager")
	ComponentHistoryReplicator        = component("history-replicator")
	ComponentHistoryResender          = component("history-resender")
	ComponentIndexer                  = component("indexer")
	ComponentIndexerProcessor         = component("indexer-processor")
	ComponentIndexerESProcessor       = component("indexer-es-processor")
	ComponentESVisibilityManager      = component("es-visibility-manager")
	ComponentArchiver                 = component("archiver")
	ComponentBatcher                  = component("batcher")
	ComponentWorker                   = component("worker")
	ComponentServiceResolver          = component("service-resolver")
	ComponentFailoverCoordinator      = component("failover-coordinator")
	ComponentFailoverMarkerNotifier   = component("failover-marker-notifier")
)

// Pre-defined values for TagSysLifecycle
var (
	LifeCycleStarting         = lifecycle("Starting")
	LifeCycleStarted          = lifecycle("Started")
	LifeCycleStopping         = lifecycle("Stopping")
	LifeCycleStopped          = lifecycle("Stopped")
	LifeCycleStopTimedout     = lifecycle("StopTimedout")
	LifeCycleStartFailed      = lifecycle("StartFailed")
	LifeCycleStopFailed       = lifecycle("StopFailed")
	LifeCycleProcessingFailed = lifecycle("ProcessingFailed")
)

// Pre-defined values for SysErrorType
var (
	ErrorTypeInvalidHistoryAction         = errorType("InvalidHistoryAction")
	ErrorTypeInvalidQueryTask             = errorType("InvalidQueryTask")
	ErrorTypeQueryTaskFailed              = errorType("QueryTaskFailed")
	ErrorTypePersistentStoreError         = errorType("PersistentStoreError")
	ErrorTypeHistorySerializationError    = errorType("HistorySerializationError")
	ErrorTypeHistoryDeserializationError  = errorType("HistoryDeserializationError")
	ErrorTypeDuplicateTask                = errorType("DuplicateTask")
	ErrorTypeMultipleCompletionDecisions  = errorType("MultipleCompletionDecisions")
	ErrorTypeDuplicateTransferTask        = errorType("DuplicateTransferTask")
	ErrorTypeDecisionFailed               = errorType("DecisionFailed")
	ErrorTypeInvalidMutableStateAction    = errorType("InvalidMutableStateAction")
	ErrorTypeInvalidMemDecisionTaskAction = errorType("InvalidMemDecisionTaskAction")
)

// Pre-defined values for SysShardUpdate
var (
	// Shard context events
	ValueShardRangeUpdated            = shardupdate("ShardRangeUpdated")
	ValueShardAllocateTimerBeforeRead = shardupdate("ShardAllocateTimerBeforeRead")
	ValueRingMembershipChangedEvent   = shardupdate("RingMembershipChangedEvent")
)

// Pre-defined values for OperationResult
var (
	OperationFailed   = operationResult("OperationFailed")
	OperationStuck    = operationResult("OperationStuck")
	OperationCritical = operationResult("OperationCritical")
)

// Pre-defined values for TagSysStoreOperation
var (
	StoreOperationCreateShard = storeOperation("create-shard")
	StoreOperationGetShard    = storeOperation("get-shard")
	StoreOperationUpdateShard = storeOperation("update-shard")

	StoreOperationCreateWorkflowExecution           = storeOperation("create-wf-execution")
	StoreOperationGetWorkflowExecution              = storeOperation("get-wf-execution")
	StoreOperationUpdateWorkflowExecution           = storeOperation("update-wf-execution")
	StoreOperationConflictResolveWorkflowExecution  = storeOperation("conflict-resolve-wf-execution")
	StoreOperationResetWorkflowExecution            = storeOperation("reset-wf-execution")
	StoreOperationDeleteWorkflowExecution           = storeOperation("delete-wf-execution")
	StoreOperationDeleteCurrentWorkflowExecution    = storeOperation("delete-current-wf-execution")
	StoreOperationGetCurrentExecution               = storeOperation("get-current-execution")
	StoreOperationListCurrentExecution              = storeOperation("list-current-execution")
	StoreOperationIsWorkflowExecutionExists         = storeOperation("is-wf-execution-exists")
	StoreOperationListConcreteExecution             = storeOperation("list-concrete-execution")
	StoreOperationGetTransferTasks                  = storeOperation("get-transfer-tasks")
	StoreOperationGetReplicationTasks               = storeOperation("get-replication-tasks")
	StoreOperationCompleteTransferTask              = storeOperation("complete-transfer-task")
	StoreOperationRangeCompleteTransferTask         = storeOperation("range-complete-transfer-task")
	StoreOperationCompleteReplicationTask           = storeOperation("complete-replication-task")
	StoreOperationRangeCompleteReplicationTask      = storeOperation("range-complete-replication-task")
	StoreOperationPutReplicationTaskToDLQ           = storeOperation("put-replication-task-to-dlq")
	StoreOperationGetReplicationTasksFromDLQ        = storeOperation("get-replication-tasks-from-dlq")
	StoreOperationGetReplicationDLQSize             = storeOperation("get-replication-dlq-size")
	StoreOperationDeleteReplicationTaskFromDLQ      = storeOperation("delete-replication-task-from-dlq")
	StoreOperationRangeDeleteReplicationTaskFromDLQ = storeOperation("range-delete-replication-task-from-dlq")
	StoreOperationCreateFailoverMarkerTasks         = storeOperation("createFailoverMarkerTasks")
	StoreOperationGetTimerIndexTasks                = storeOperation("get-timer-index-tasks")
	StoreOperationCompleteTimerTask                 = storeOperation("complete-timer-task")
	StoreOperationRangeCompleteTimerTask            = storeOperation("range-complete-timer-task")

	StoreOperationCreateTasks           = storeOperation("create-tasks")
	StoreOperationGetTasks              = storeOperation("get-tasks")
	StoreOperationCompleteTask          = storeOperation("complete-task")
	StoreOperationCompleteTasksLessThan = storeOperation("complete-tasks-less-than")
	StoreOperationLeaseTaskList         = storeOperation("lease-task-list")
	StoreOperationUpdateTaskList        = storeOperation("update-task-list")
	StoreOperationListTaskList          = storeOperation("list-task-list")
	StoreOperationDeleteTaskList        = storeOperation("delete-task-list")
	StoreOperationStopTaskList          = storeOperation("stop-task-list")

	StoreOperationCreateDomain       = storeOperation("create-domain")
	StoreOperationGetDomain          = storeOperation("get-domain")
	StoreOperationUpdateDomain       = storeOperation("update-domain")
	StoreOperationDeleteDomain       = storeOperation("delete-domain")
	StoreOperationDeleteDomainByName = storeOperation("delete-domain-by-name")
	StoreOperationListDomains        = storeOperation("list-domains")
	StoreOperationGetMetadata        = storeOperation("get-metadata")

	StoreOperationRecordWorkflowExecutionStarted           = storeOperation("record-wf-execution-started")
	StoreOperationRecordWorkflowExecutionClosed            = storeOperation("record-wf-execution-closed")
	StoreOperationUpsertWorkflowExecution                  = storeOperation("upsert-wf-execution")
	StoreOperationListOpenWorkflowExecutions               = storeOperation("list-open-wf-executions")
	StoreOperationListClosedWorkflowExecutions             = storeOperation("list-closed-wf-executions")
	StoreOperationListOpenWorkflowExecutionsByType         = storeOperation("list-open-wf-executions-by-type")
	StoreOperationListClosedWorkflowExecutionsByType       = storeOperation("list-closed-wf-executions-by-type")
	StoreOperationListOpenWorkflowExecutionsByWorkflowID   = storeOperation("list-open-wf-executions-by-wfID")
	StoreOperationListClosedWorkflowExecutionsByWorkflowID = storeOperation("list-closed-wf-executions-by-wfID")
	StoreOperationListClosedWorkflowExecutionsByStatus     = storeOperation("list-closed-wf-executions-by-status")
	StoreOperationGetClosedWorkflowExecution               = storeOperation("get-closed-wf-execution")
	StoreOperationVisibilityDeleteWorkflowExecution        = storeOperation("vis-delete-wf-execution")
	StoreOperationListWorkflowExecutions                   = storeOperation("list-wf-executions")
	StoreOperationScanWorkflowExecutions                   = storeOperation("scan-wf-executions")
	StoreOperationCountWorkflowExecutions                  = storeOperation("count-wf-executions")

	StoreOperationAppendHistoryNodes        = storeOperation("append-history-nodes")
	StoreOperationReadHistoryBranch         = storeOperation("read-history-branch")
	StoreOperationReadHistoryBranchByBatch  = storeOperation("read-history-branch-by-batch")
	StoreOperationReadRawHistoryBranch      = storeOperation("read-raw-history-branch")
	StoreOperationForkHistoryBranch         = storeOperation("fork-history-branch")
	StoreOperationDeleteHistoryBranch       = storeOperation("delete-history-branch")
	StoreOperationGetHistoryTree            = storeOperation("get-history-tree")
	StoreOperationGetAllHistoryTreeBranches = storeOperation("get-all-history-tree-branches")

	StoreOperationEnqueueMessage             = storeOperation("enqueue-message")
	StoreOperationReadMessages               = storeOperation("read-messages")
	StoreOperationUpdateAckLevel             = storeOperation("update-ack-level")
	StoreOperationGetAckLevels               = storeOperation("get-ack-levels")
	StoreOperationDeleteMessagesBefore       = storeOperation("delete-messages-before")
	StoreOperationEnqueueMessageToDLQ        = storeOperation("enqueue-message-to-dlq")
	StoreOperationReadMessagesFromDLQ        = storeOperation("read-messages-from-dlq")
	StoreOperationRangeDeleteMessagesFromDLQ = storeOperation("range-delete-messages-from-dlq")
	StoreOperationUpdateDLQAckLevel          = storeOperation("update-dlq-ack-level")
	StoreOperationGetDLQAckLevels            = storeOperation("get-dlq-ack-levels")
	StoreOperationGetDLQSize                 = storeOperation("get-dlq-size")
	StoreOperationDeleteMessageFromDLQ       = storeOperation("delete-message-from-dlq")
)

// Pre-defined values for TagSysClientOperation
var (
	AdminClientOperationAddSearchAttribute               = clientOperation("admin-add-search-attribute")
	AdminClientOperationDescribeHistoryHost              = clientOperation("admin-describe-history-host")
	AdminClientOperationRemoveTask                       = clientOperation("admin-remove-task")
	AdminClientOperationCloseShard                       = clientOperation("admin-close-shard")
	AdminClientOperationResetQueue                       = clientOperation("admin-reset-queue")
	AdminClientOperationDescribeQueue                    = clientOperation("admin-describe-queue")
	AdminClientOperationDescribeWorkflowExecution        = clientOperation("admin-describe-wf-execution")
	AdminClientOperationGetWorkflowExecutionRawHistoryV2 = clientOperation("admin-get-wf-execution-raw-history-v2")
	AdminClientOperationDescribeCluster                  = clientOperation("admin-describe-cluster")
	AdminClientOperationGetReplicationMessages           = clientOperation("admin-get-replication-messsages")
	AdminClientOperationGetDomainReplicationMessages     = clientOperation("admin-get-domain-replication-messsages")
	AdminClientOperationGetDLQReplicationMessages        = clientOperation("admin-get-dlq-replication-messsages")
	AdminClientOperationReapplyEvents                    = clientOperation("admin-reapply-events")
	AdminClientOperationReadDLQMessages                  = clientOperation("admin-read-dlq-messsages")
	AdminClientOperationPurgeDLQMessages                 = clientOperation("admin-purge-dlq-messsages")
	AdminClientOperationMergeDLQMessages                 = clientOperation("admin-merge-dlq-messsages")
	AdminClientOperationRefreshWorkflowTasks             = clientOperation("admin-refresh-wf-tasks")
	AdminClientOperationResendReplicationTasks           = clientOperation("admin-resend-replication-tasks")

	FrontendClientOperationDeprecateDomain                  = clientOperation("frontend-deprecate-domain")
	FrontendClientOperationDescribeDomain                   = clientOperation("frontend-describe-domain")
	FrontendClientOperationDescribeTaskList                 = clientOperation("frontend-describe-task-list")
	FrontendClientOperationDescribeWorkflowExecution        = clientOperation("frontend-describe-wf-execution")
	FrontendClientOperationGetWorkflowExecutionHistory      = clientOperation("frontend-get-wf-execution-history")
	FrontendClientOperationListArchivedWorkflowExecutions   = clientOperation("frontend-list-archived-wf-executions")
	FrontendClientOperationListClosedWorkflowExecutions     = clientOperation("frontend-list-closed-wf-executions")
	FrontendClientOperationListDomains                      = clientOperation("frontend-list-domains")
	FrontendClientOperationListOpenWorkflowExecutions       = clientOperation("frontend-list-open-wf-executions")
	FrontendClientOperationListWorkflowExecutions           = clientOperation("frontend-list-wf-executions")
	FrontendClientOperationScanWorkflowExecutions           = clientOperation("frontend-scan-wf-executions")
	FrontendClientOperationCountWorkflowExecutions          = clientOperation("frontend-count-wf-executions")
	FrontendClientOperationGetSearchAttributes              = clientOperation("frontend-get-search-attributes")
	FrontendClientOperationPollForActivityTask              = clientOperation("frontend-poll-for-activity-task")
	FrontendClientOperationPollForDecisionTask              = clientOperation("frontend-poll-for-decision-task")
	FrontendClientOperationQueryWorkflow                    = clientOperation("frontend-query-workflow")
	FrontendClientOperationRecordActivityTaskHeartbeat      = clientOperation("frontend-record-activity-heartbeat")
	FrontendClientOperationRecordActivityTaskHeartbeatByID  = clientOperation("frontend-record-activity-heartbeat-by-id")
	FrontendClientOperationRegisterDomain                   = clientOperation("frontend-register-domain")
	FrontendClientOperationRequestCancelWorkflowExecution   = clientOperation("frontend-request-cancel-wf-execution")
	FrontendClientOperationResetStickyTaskList              = clientOperation("frontend-reset-sticky-task-list")
	FrontendClientOperationResetWorkflowExecution           = clientOperation("frontend-reset-wf-execution")
	FrontendClientOperationRespondActivityTaskCanceled      = clientOperation("frontend-respond-activity-task-canceled")
	FrontendClientOperationRespondActivityTaskCanceledByID  = clientOperation("frontend-respond-activity-task-canceled-by-id")
	FrontendClientOperationRespondActivityTaskCompleted     = clientOperation("frontend-respond-activity-task-completed")
	FrontendClientOperationRespondActivityTaskCompletedByID = clientOperation("frontend-respond-activity-task-completed-by-id")
	FrontendClientOperationRespondActivityTaskFailed        = clientOperation("frontend-respond-activity-task-failed")
	FrontendClientOperationRespondActivityTaskFailedByID    = clientOperation("frontend-respond-activity-task-failed-by-id")
	FrontendClientOperationRespondDecisionTaskCompleted     = clientOperation("frontend-respond-decision-task-completed")
	FrontendClientOperationRespondDecisionTaskFailed        = clientOperation("frontend-respond-decision-task-failed")
	FrontendClientOperationRespondQueryTaskCompleted        = clientOperation("frontend-respond-query-task-completed")
	FrontendClientOperationSignalWithStartWorkflowExecution = clientOperation("frontend-signal-with-start-wf-execution")
	FrontendClientOperationSignalWorkflowExecution          = clientOperation("frontend-signal-wf-execution")
	FrontendClientOperationStartWorkflowExecution           = clientOperation("frontend-start-wf-execution")
	FrontendClientOperationTerminateWorkflowExecution       = clientOperation("frontend-terminate-wf-execution")
	FrontendClientOperationUpdateDomain                     = clientOperation("frontend-update-domain")
	FrontendClientOperationGetClusterInfo                   = clientOperation("frontend-get-cluster-info")
	FrontendClientOperationListTaskListPartitions           = clientOperation("frontend-list-task-list-partitions")

	HistoryClientOperationStartWorkflowExecution           = clientOperation("history-start-wf-execution")
	HistoryClientOperationDescribeHistoryHost              = clientOperation("history-describe-history-host")
	HistoryClientOperationCloseShard                       = clientOperation("history-close-shard")
	HistoryClientOperationResetQueue                       = clientOperation("history-reset-queue")
	HistoryClientOperationDescribeQueue                    = clientOperation("history-describe-queue")
	HistoryClientOperationRemoveTask                       = clientOperation("history-remove-task")
	HistoryClientOperationDescribeMutableState             = clientOperation("history-describe-mutable-state")
	HistoryClientOperationGetMutableState                  = clientOperation("history-get-mutable-state")
	HistoryClientOperationPollMutableState                 = clientOperation("history-poll-mutable-state")
	HistoryClientOperationResetStickyTaskList              = clientOperation("history-reset-task-list")
	HistoryClientOperationDescribeWorkflowExecution        = clientOperation("history-describe-wf-execution")
	HistoryClientOperationRecordDecisionTaskStarted        = clientOperation("history-record-decision-task-started")
	HistoryClientOperationRecordActivityTaskStarted        = clientOperation("history-record-activity-task-started")
	HistoryClientOperationRecordDecisionTaskCompleted      = clientOperation("history-record-decision-task-completed")
	HistoryClientOperationRecordDecisionTaskFailed         = clientOperation("history-record-decision-task-failed")
	HistoryClientOperationRecordActivityTaskCompleted      = clientOperation("history-record-activity-task-completed")
	HistoryClientOperationRecordActivityTaskFailed         = clientOperation("history-record-activity-task-failed")
	HistoryClientOperationRecordActivityTaskCanceled       = clientOperation("history-record-activity-task-canceled")
	HistoryClientOperationRecordActivityTaskHeartbeat      = clientOperation("history-record-activity-task-heartbeat")
	HistoryClientOperationRequestCancelWorkflowExecution   = clientOperation("history-request-cancel-wf-execution")
	HistoryClientOperationSignalWorkflowExecution          = clientOperation("history-signal-wf-execution")
	HistoryClientOperationSignalWithStartWorkflowExecution = clientOperation("history-signal-with-start-wf-execution")
	HistoryClientOperationRemoveSignalMutableState         = clientOperation("history-remove-signal-mutable-state")
	HistoryClientOperationTerminateWorkflowExecution       = clientOperation("history-terminate-wf-execution")
	HistoryClientOperationResetWorkflowExecution           = clientOperation("history-reset-wf-execution")
	HistoryClientOperationScheduleDecisionTask             = clientOperation("history-schedule-decision-task")
	HistoryClientOperationRecordChildExecutionCompleted    = clientOperation("history-record-child-execution-completed")
	HistoryClientOperationReplicateEventsV2                = clientOperation("history-replicate-events-v2")
	HistoryClientOperationSyncShardStatus                  = clientOperation("history-sync-shard-status")
	HistoryClientOperationSyncActivity                     = clientOperation("history-sync-activity")
	HistoryClientOperationGetReplicationMessages           = clientOperation("history-get-replication-messages")
	HistoryClientOperationGetDLQReplicationMessages        = clientOperation("history-get-dlq-replication-messages")
	HistoryClientOperationQueryWorkflow                    = clientOperation("history-query-wf")
	HistoryClientOperationReapplyEvents                    = clientOperation("history-reapply-events")
	HistoryClientOperationReadDLQMessages                  = clientOperation("history-read-dlq-messages")
	HistoryClientOperationPurgeDLQMessages                 = clientOperation("history-purge-dlq-messages")
	HistoryClientOperationMergeDLQMessages                 = clientOperation("history-merge-dlq-messages")
	HistoryClientOperationRefreshWorkflowTasks             = clientOperation("history-refresh-wf-tasks")
	HistoryClientOperationNotifyFailoverMarkers            = clientOperation("history-notify-failover-markers")

	MatchingClientOperationAddActivityTask        = clientOperation("matching-add-activity-task")
	MatchingClientOperationAddDecisionTask        = clientOperation("matching-add-decision-task")
	MatchingClientOperationPollForActivityTask    = clientOperation("matching-poll-for-activity-task")
	MatchingClientOperationPollForDecisionTask    = clientOperation("matching-poll-for-decision-task")
	MatchingClientOperationQueryWorkflow          = clientOperation("matching-query-wf")
	MatchingClientOperationQueryTaskCompleted     = clientOperation("matching-query-task-completed")
	MatchingClientOperationCancelOutstandingPoll  = clientOperation("matching-cancel-outstanding-poll")
	MatchingClientOperationDescribeTaskList       = clientOperation("matching-describe-task-list")
	MatchingClientOperationListTaskListPartitions = clientOperation("matching-list-task-list-partitions")
)
