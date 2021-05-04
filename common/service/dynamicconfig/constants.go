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

package dynamicconfig

// Key represents a key/property stored in dynamic config
type Key int

func (k Key) String() string {
	keyName, ok := keys[k]
	if !ok {
		return keys[unknownKey]
	}
	return keyName
}

// Mapping from Key to keyName, where keyName are used dynamic config source.
var keys = map[Key]string{
	unknownKey: "unknownKey",

	// tests keys
	testGetPropertyKey:                               "testGetPropertyKey",
	testGetIntPropertyKey:                            "testGetIntPropertyKey",
	testGetFloat64PropertyKey:                        "testGetFloat64PropertyKey",
	testGetDurationPropertyKey:                       "testGetDurationPropertyKey",
	testGetBoolPropertyKey:                           "testGetBoolPropertyKey",
	testGetStringPropertyKey:                         "testGetStringPropertyKey",
	testGetMapPropertyKey:                            "testGetMapPropertyKey",
	testGetIntPropertyFilteredByDomainKey:            "testGetIntPropertyFilteredByDomainKey",
	testGetDurationPropertyFilteredByDomainKey:       "testGetDurationPropertyFilteredByDomainKey",
	testGetIntPropertyFilteredByTaskListInfoKey:      "testGetIntPropertyFilteredByTaskListInfoKey",
	testGetDurationPropertyFilteredByTaskListInfoKey: "testGetDurationPropertyFilteredByTaskListInfoKey",
	testGetBoolPropertyFilteredByDomainIDKey:         "testGetBoolPropertyFilteredByDomainIDKey",
	testGetBoolPropertyFilteredByTaskListInfoKey:     "testGetBoolPropertyFilteredByTaskListInfoKey",

	// system settings
	EnableGlobalDomain:                  "system.enableGlobalDomain",
	EnableNewKafkaClient:                "system.enableNewKafkaClient",
	EnableVisibilitySampling:            "system.enableVisibilitySampling",
	EnableReadFromClosedExecutionV2:     "system.enableReadFromClosedExecutionV2",
	AdvancedVisibilityWritingMode:       "system.advancedVisibilityWritingMode",
	EnableReadVisibilityFromES:          "system.enableReadVisibilityFromES",
	HistoryArchivalStatus:               "system.historyArchivalStatus",
	EnableReadFromHistoryArchival:       "system.enableReadFromHistoryArchival",
	VisibilityArchivalStatus:            "system.visibilityArchivalStatus",
	EnableReadFromVisibilityArchival:    "system.enableReadFromVisibilityArchival",
	EnableDomainNotActiveAutoForwarding: "system.enableDomainNotActiveAutoForwarding",
	EnableGracefulFailover:              "system.enableGracefulFailover",
	TransactionSizeLimit:                "system.transactionSizeLimit",
	PersistenceErrorInjectionRate:       "system.persistenceErrorInjectionRate",
	MinRetentionDays:                    "system.minRetentionDays",
	MaxDecisionStartToCloseSeconds:      "system.maxDecisionStartToCloseSeconds",
	DisallowQuery:                       "system.disallowQuery",
	EnableBatcher:                       "worker.enableBatcher",
	EnableParentClosePolicyWorker:       "system.enableParentClosePolicyWorker",
	EnableFailoverManager:               "system.enableFailoverManager",
	EnableStickyQuery:                   "system.enableStickyQuery",
	EnablePriorityTaskProcessor:         "system.enablePriorityTaskProcessor",
	EnableAuthorization:                 "system.enableAuthorization",

	// size limit
	BlobSizeLimitError:      "limit.blobSize.error",
	BlobSizeLimitWarn:       "limit.blobSize.warn",
	HistorySizeLimitError:   "limit.historySize.error",
	HistorySizeLimitWarn:    "limit.historySize.warn",
	HistoryCountLimitError:  "limit.historyCount.error",
	HistoryCountLimitWarn:   "limit.historyCount.warn",
	MaxIDLengthLimit:        "limit.maxIDLength",
	MaxIDLengthWarnLimit:    "limit.maxIDWarnLength",
	MaxRawTaskListNameLimit: "limit.maxRawTaskListNameLength",

	// frontend settings
	FrontendPersistenceMaxQPS:                   "frontend.persistenceMaxQPS",
	FrontendPersistenceGlobalMaxQPS:             "frontend.persistenceGlobalMaxQPS",
	FrontendVisibilityMaxPageSize:               "frontend.visibilityMaxPageSize",
	FrontendVisibilityListMaxQPS:                "frontend.visibilityListMaxQPS",
	FrontendESVisibilityListMaxQPS:              "frontend.esVisibilityListMaxQPS",
	FrontendMaxBadBinaries:                      "frontend.maxBadBinaries",
	FrontendFailoverCoolDown:                    "frontend.failoverCoolDown",
	FrontendESIndexMaxResultWindow:              "frontend.esIndexMaxResultWindow",
	FrontendHistoryMaxPageSize:                  "frontend.historyMaxPageSize",
	FrontendRPS:                                 "frontend.rps",
	FrontendMaxDomainRPSPerInstance:             "frontend.domainrps",
	FrontendGlobalDomainRPS:                     "frontend.globalDomainrps",
	FrontendHistoryMgrNumConns:                  "frontend.historyMgrNumConns",
	FrontendShutdownDrainDuration:               "frontend.shutdownDrainDuration",
	DisableListVisibilityByFilter:               "frontend.disableListVisibilityByFilter",
	FrontendThrottledLogRPS:                     "frontend.throttledLogRPS",
	EnableClientVersionCheck:                    "frontend.enableClientVersionCheck",
	ValidSearchAttributes:                       "frontend.validSearchAttributes",
	SendRawWorkflowHistory:                      "frontend.sendRawWorkflowHistory",
	SearchAttributesNumberOfKeysLimit:           "frontend.searchAttributesNumberOfKeysLimit",
	SearchAttributesSizeOfValueLimit:            "frontend.searchAttributesSizeOfValueLimit",
	SearchAttributesTotalSizeLimit:              "frontend.searchAttributesTotalSizeLimit",
	VisibilityArchivalQueryMaxPageSize:          "frontend.visibilityArchivalQueryMaxPageSize",
	VisibilityArchivalQueryMaxRangeInDays:       "frontend.visibilityArchivalQueryMaxRangeInDays",
	VisibilityArchivalQueryMaxQPS:               "frontend.visibilityArchivalQueryMaxQPS",
	DomainFailoverRefreshInterval:               "frontend.domainFailoverRefreshInterval",
	DomainFailoverRefreshTimerJitterCoefficient: "frontend.domainFailoverRefreshTimerJitterCoefficient",

	// matching settings
	MatchingRPS:                             "matching.rps",
	MatchingPersistenceMaxQPS:               "matching.persistenceMaxQPS",
	MatchingPersistenceGlobalMaxQPS:         "matching.persistenceGlobalMaxQPS",
	MatchingMinTaskThrottlingBurstSize:      "matching.minTaskThrottlingBurstSize",
	MatchingGetTasksBatchSize:               "matching.getTasksBatchSize",
	MatchingLongPollExpirationInterval:      "matching.longPollExpirationInterval",
	MatchingEnableSyncMatch:                 "matching.enableSyncMatch",
	MatchingUpdateAckInterval:               "matching.updateAckInterval",
	MatchingIdleTasklistCheckInterval:       "matching.idleTasklistCheckInterval",
	MaxTasklistIdleTime:                     "matching.maxTasklistIdleTime",
	MatchingOutstandingTaskAppendsThreshold: "matching.outstandingTaskAppendsThreshold",
	MatchingMaxTaskBatchSize:                "matching.maxTaskBatchSize",
	MatchingMaxTaskDeleteBatchSize:          "matching.maxTaskDeleteBatchSize",
	MatchingThrottledLogRPS:                 "matching.throttledLogRPS",
	MatchingNumTasklistWritePartitions:      "matching.numTasklistWritePartitions",
	MatchingNumTasklistReadPartitions:       "matching.numTasklistReadPartitions",
	MatchingForwarderMaxOutstandingPolls:    "matching.forwarderMaxOutstandingPolls",
	MatchingForwarderMaxOutstandingTasks:    "matching.forwarderMaxOutstandingTasks",
	MatchingForwarderMaxRatePerSecond:       "matching.forwarderMaxRatePerSecond",
	MatchingForwarderMaxChildrenPerNode:     "matching.forwarderMaxChildrenPerNode",
	MatchingShutdownDrainDuration:           "matching.shutdownDrainDuration",

	// history settings
	HistoryRPS:                                            "history.rps",
	HistoryPersistenceMaxQPS:                              "history.persistenceMaxQPS",
	HistoryPersistenceGlobalMaxQPS:                        "history.persistenceGlobalMaxQPS",
	HistoryVisibilityOpenMaxQPS:                           "history.historyVisibilityOpenMaxQPS",
	HistoryVisibilityClosedMaxQPS:                         "history.historyVisibilityClosedMaxQPS",
	HistoryLongPollExpirationInterval:                     "history.longPollExpirationInterval",
	HistoryCacheInitialSize:                               "history.cacheInitialSize",
	HistoryMaxAutoResetPoints:                             "history.historyMaxAutoResetPoints",
	HistoryCacheMaxSize:                                   "history.cacheMaxSize",
	HistoryCacheTTL:                                       "history.cacheTTL",
	HistoryShutdownDrainDuration:                          "history.shutdownDrainDuration",
	EventsCacheInitialCount:                               "history.eventsCacheInitialSize",
	EventsCacheMaxCount:                                   "history.eventsCacheMaxSize",
	EventsCacheMaxSize:                                    "history.eventsCacheMaxSizeInBytes",
	EventsCacheTTL:                                        "history.eventsCacheTTL",
	EventsCacheGlobalEnable:                               "history.eventsCacheGlobalEnable",
	EventsCacheGlobalInitialCount:                         "history.eventsCacheGlobalInitialSize",
	EventsCacheGlobalMaxCount:                             "history.eventsCacheGlobalMaxSize",
	AcquireShardInterval:                                  "history.acquireShardInterval",
	AcquireShardConcurrency:                               "history.acquireShardConcurrency",
	StandbyClusterDelay:                                   "history.standbyClusterDelay",
	StandbyTaskMissingEventsResendDelay:                   "history.standbyTaskMissingEventsResendDelay",
	StandbyTaskMissingEventsDiscardDelay:                  "history.standbyTaskMissingEventsDiscardDelay",
	TaskProcessRPS:                                        "history.taskProcessRPS",
	TaskSchedulerType:                                     "history.taskSchedulerType",
	TaskSchedulerWorkerCount:                              "history.taskSchedulerWorkerCount",
	TaskSchedulerShardWorkerCount:                         "history.taskSchedulerShardWorkerCount",
	TaskSchedulerQueueSize:                                "history.taskSchedulerQueueSize",
	TaskSchedulerShardQueueSize:                           "history.taskSchedulerShardQueueSize",
	TaskSchedulerDispatcherCount:                          "history.taskSchedulerDispatcherCount",
	TaskSchedulerRoundRobinWeights:                        "history.taskSchedulerRoundRobinWeight",
	ActiveTaskRedispatchInterval:                          "history.activeTaskRedispatchInterval",
	StandbyTaskRedispatchInterval:                         "history.standbyTaskRedispatchInterval",
	TaskRedispatchIntervalJitterCoefficient:               "history.taskRedispatchIntervalJitterCoefficient",
	StandbyTaskReReplicationContextTimeout:                "history.standbyTaskReReplicationContextTimeout",
	QueueProcessorEnableSplit:                             "history.queueProcessorEnableSplit",
	QueueProcessorSplitMaxLevel:                           "history.queueProcessorSplitMaxLevel",
	QueueProcessorEnableRandomSplitByDomainID:             "history.queueProcessorEnableRandomSplitByDomainID",
	QueueProcessorRandomSplitProbability:                  "history.queueProcessorRandomSplitProbability",
	QueueProcessorEnablePendingTaskSplitByDomainID:        "history.queueProcessorEnablePendingTaskSplitByDomainID",
	QueueProcessorPendingTaskSplitThreshold:               "history.queueProcessorPendingTaskSplitThreshold",
	QueueProcessorEnableStuckTaskSplitByDomainID:          "history.queueProcessorEnableStuckTaskSplitByDomainID",
	QueueProcessorStuckTaskSplitThreshold:                 "history.queueProcessorStuckTaskSplitThreshold",
	QueueProcessorSplitLookAheadDurationByDomainID:        "history.queueProcessorSplitLookAheadDurationByDomainID",
	QueueProcessorPollBackoffInterval:                     "history.queueProcessorPollBackoffInterval",
	QueueProcessorPollBackoffIntervalJitterCoefficient:    "history.queueProcessorPollBackoffIntervalJitterCoefficient",
	QueueProcessorEnablePersistQueueStates:                "history.queueProcessorEnablePersistQueueStates",
	QueueProcessorEnableLoadQueueStates:                   "history.queueProcessorEnableLoadQueueStates",
	TimerTaskBatchSize:                                    "history.timerTaskBatchSize",
	TimerTaskWorkerCount:                                  "history.timerTaskWorkerCount",
	TimerTaskMaxRetryCount:                                "history.timerTaskMaxRetryCount",
	TimerProcessorGetFailureRetryCount:                    "history.timerProcessorGetFailureRetryCount",
	TimerProcessorCompleteTimerFailureRetryCount:          "history.timerProcessorCompleteTimerFailureRetryCount",
	TimerProcessorUpdateShardTaskCount:                    "history.timerProcessorUpdateShardTaskCount",
	TimerProcessorUpdateAckInterval:                       "history.timerProcessorUpdateAckInterval",
	TimerProcessorUpdateAckIntervalJitterCoefficient:      "history.timerProcessorUpdateAckIntervalJitterCoefficient",
	TimerProcessorCompleteTimerInterval:                   "history.timerProcessorCompleteTimerInterval",
	TimerProcessorFailoverMaxPollRPS:                      "history.timerProcessorFailoverMaxPollRPS",
	TimerProcessorMaxPollRPS:                              "history.timerProcessorMaxPollRPS",
	TimerProcessorMaxPollInterval:                         "history.timerProcessorMaxPollInterval",
	TimerProcessorMaxPollIntervalJitterCoefficient:        "history.timerProcessorMaxPollIntervalJitterCoefficient",
	TimerProcessorSplitQueueInterval:                      "history.timerProcessorSplitQueueInterval",
	TimerProcessorSplitQueueIntervalJitterCoefficient:     "history.timerProcessorSplitQueueIntervalJitterCoefficient",
	TimerProcessorMaxRedispatchQueueSize:                  "history.timerProcessorMaxRedispatchQueueSize",
	TimerProcessorEnablePriorityTaskProcessor:             "history.timerProcessorEnablePriorityTaskProcessor",
	TimerProcessorEnableMultiCurosrProcessor:              "history.timerProcessorEnableMultiCursorProcessor",
	TimerProcessorMaxTimeShift:                            "history.timerProcessorMaxTimeShift",
	TimerProcessorHistoryArchivalSizeLimit:                "history.timerProcessorHistoryArchivalSizeLimit",
	TimerProcessorArchivalTimeLimit:                       "history.timerProcessorArchivalTimeLimit",
	TransferTaskBatchSize:                                 "history.transferTaskBatchSize",
	TransferProcessorFailoverMaxPollRPS:                   "history.transferProcessorFailoverMaxPollRPS",
	TransferProcessorMaxPollRPS:                           "history.transferProcessorMaxPollRPS",
	TransferTaskWorkerCount:                               "history.transferTaskWorkerCount",
	TransferTaskMaxRetryCount:                             "history.transferTaskMaxRetryCount",
	TransferProcessorCompleteTransferFailureRetryCount:    "history.transferProcessorCompleteTransferFailureRetryCount",
	TransferProcessorUpdateShardTaskCount:                 "history.transferProcessorUpdateShardTaskCount",
	TransferProcessorMaxPollInterval:                      "history.transferProcessorMaxPollInterval",
	TransferProcessorMaxPollIntervalJitterCoefficient:     "history.transferProcessorMaxPollIntervalJitterCoefficient",
	TransferProcessorSplitQueueInterval:                   "history.transferProcessorSplitQueueInterval",
	TransferProcessorSplitQueueIntervalJitterCoefficient:  "history.transferProcessorSplitQueueIntervalJitterCoefficient",
	TransferProcessorUpdateAckInterval:                    "history.transferProcessorUpdateAckInterval",
	TransferProcessorUpdateAckIntervalJitterCoefficient:   "history.transferProcessorUpdateAckIntervalJitterCoefficient",
	TransferProcessorCompleteTransferInterval:             "history.transferProcessorCompleteTransferInterval",
	TransferProcessorMaxRedispatchQueueSize:               "history.transferProcessorMaxRedispatchQueueSize",
	TransferProcessorEnablePriorityTaskProcessor:          "history.transferProcessorEnablePriorityTaskProcessor",
	TransferProcessorEnableMultiCurosrProcessor:           "history.transferProcessorEnableMultiCursorProcessor",
	TransferProcessorVisibilityArchivalTimeLimit:          "history.transferProcessorVisibilityArchivalTimeLimit",
	ReplicatorTaskBatchSize:                               "history.replicatorTaskBatchSize",
	ReplicatorTaskWorkerCount:                             "history.replicatorTaskWorkerCount",
	ReplicatorReadTaskMaxRetryCount:                       "history.replicatorReadTaskMaxRetryCount",
	ReplicatorTaskMaxRetryCount:                           "history.replicatorTaskMaxRetryCount",
	ReplicatorProcessorMaxPollRPS:                         "history.replicatorProcessorMaxPollRPS",
	ReplicatorProcessorUpdateShardTaskCount:               "history.replicatorProcessorUpdateShardTaskCount",
	ReplicatorProcessorMaxPollInterval:                    "history.replicatorProcessorMaxPollInterval",
	ReplicatorProcessorMaxPollIntervalJitterCoefficient:   "history.replicatorProcessorMaxPollIntervalJitterCoefficient",
	ReplicatorProcessorUpdateAckInterval:                  "history.replicatorProcessorUpdateAckInterval",
	ReplicatorProcessorUpdateAckIntervalJitterCoefficient: "history.replicatorProcessorUpdateAckIntervalJitterCoefficient",
	ReplicatorProcessorMaxRedispatchQueueSize:             "history.replicatorProcessorMaxRedispatchQueueSize",
	ReplicatorProcessorEnablePriorityTaskProcessor:        "history.replicatorProcessorEnablePriorityTaskProcessor",
	ExecutionMgrNumConns:                                  "history.executionMgrNumConns",
	HistoryMgrNumConns:                                    "history.historyMgrNumConns",
	MaximumBufferedEventsBatch:                            "history.maximumBufferedEventsBatch",
	MaximumSignalsPerExecution:                            "history.maximumSignalsPerExecution",
	ShardUpdateMinInterval:                                "history.shardUpdateMinInterval",
	ShardSyncMinInterval:                                  "history.shardSyncMinInterval",
	ShardSyncTimerJitterCoefficient:                       "history.shardSyncMinInterval",
	DefaultEventEncoding:                                  "history.defaultEventEncoding",
	EnableAdminProtection:                                 "history.enableAdminProtection",
	AdminOperationToken:                                   "history.adminOperationToken",
	EnableParentClosePolicy:                               "history.enableParentClosePolicy",
	NumArchiveSystemWorkflows:                             "history.numArchiveSystemWorkflows",
	ArchiveRequestRPS:                                     "history.archiveRequestRPS",
	EmitShardDiffLog:                                      "history.emitShardDiffLog",
	HistoryThrottledLogRPS:                                "history.throttledLogRPS",
	StickyTTL:                                             "history.stickyTTL",
	DecisionHeartbeatTimeout:                              "history.decisionHeartbeatTimeout",
	ParentClosePolicyThreshold:                            "history.parentClosePolicyThreshold",
	NumParentClosePolicySystemWorkflows:                   "history.numParentClosePolicySystemWorkflows",
	ReplicationTaskFetcherParallelism:                     "history.ReplicationTaskFetcherParallelism",
	ReplicationTaskFetcherAggregationInterval:             "history.ReplicationTaskFetcherAggregationInterval",
	ReplicationTaskFetcherTimerJitterCoefficient:          "history.ReplicationTaskFetcherTimerJitterCoefficient",
	ReplicationTaskFetcherErrorRetryWait:                  "history.ReplicationTaskFetcherErrorRetryWait",
	ReplicationTaskFetcherServiceBusyWait:                 "history.ReplicationTaskFetcherServiceBusyWait",
	ReplicationTaskProcessorErrorRetryWait:                "history.ReplicationTaskProcessorErrorRetryWait",
	ReplicationTaskProcessorErrorRetryMaxAttempts:         "history.ReplicationTaskProcessorErrorRetryMaxAttempts",
	ReplicationTaskProcessorErrorSecondRetryWait:          "history.ReplicationTaskProcessorErrorSecondRetryWait",
	ReplicationTaskProcessorErrorSecondRetryMaxWait:       "history.ReplicationTaskProcessorErrorSecondRetryMaxWait",
	ReplicationTaskProcessorErrorSecondRetryExpiration:    "history.ReplicationTaskProcessorErrorSecondRetryExpiration",
	ReplicationTaskProcessorNoTaskInitialWait:             "history.ReplicationTaskProcessorNoTaskInitialWait",
	ReplicationTaskProcessorCleanupInterval:               "history.ReplicationTaskProcessorCleanupInterval",
	ReplicationTaskProcessorCleanupJitterCoefficient:      "history.ReplicationTaskProcessorCleanupJitterCoefficient",
	ReplicationTaskProcessorReadHistoryBatchSize:          "history.ReplicationTaskProcessorReadHistoryBatchSize",
	ReplicationTaskProcessorStartWait:                     "history.ReplicationTaskProcessorStartWait",
	ReplicationTaskProcessorStartWaitJitterCoefficient:    "history.ReplicationTaskProcessorStartWaitJitterCoefficient",
	ReplicationTaskProcessorHostQPS:                       "history.ReplicationTaskProcessorHostQPS",
	ReplicationTaskProcessorShardQPS:                      "history.ReplicationTaskProcessorShardQPS",
	ReplicationTaskGenerationQPS:                          "history.ReplicationTaskGenerationQPS",
	EnableConsistentQuery:                                 "history.EnableConsistentQuery",
	EnableConsistentQueryByDomain:                         "history.EnableConsistentQueryByDomain",
	MaxBufferedQueryCount:                                 "history.MaxBufferedQueryCount",
	MutableStateChecksumGenProbability:                    "history.mutableStateChecksumGenProbability",
	MutableStateChecksumVerifyProbability:                 "history.mutableStateChecksumVerifyProbability",
	MutableStateChecksumInvalidateBefore:                  "history.mutableStateChecksumInvalidateBefore",
	ReplicationEventsFromCurrentCluster:                   "history.ReplicationEventsFromCurrentCluster",
	NotifyFailoverMarkerInterval:                          "history.NotifyFailoverMarkerInterval",
	NotifyFailoverMarkerTimerJitterCoefficient:            "history.NotifyFailoverMarkerTimerJitterCoefficient",
	EnableDropStuckTaskByDomainID:                         "history.DropStuckTaskByDomain",
	EnableActivityLocalDispatchByDomain:                   "history.enableActivityLocalDispatchByDomain",

	WorkerPersistenceMaxQPS:                                  "worker.persistenceMaxQPS",
	WorkerPersistenceGlobalMaxQPS:                            "worker.persistenceGlobalMaxQPS",
	WorkerReplicatorMetaTaskConcurrency:                      "worker.replicatorMetaTaskConcurrency",
	WorkerReplicatorTaskConcurrency:                          "worker.replicatorTaskConcurrency",
	WorkerReplicatorMessageConcurrency:                       "worker.replicatorMessageConcurrency",
	WorkerReplicatorActivityBufferRetryCount:                 "worker.replicatorActivityBufferRetryCount",
	WorkerReplicatorHistoryBufferRetryCount:                  "worker.replicatorHistoryBufferRetryCount",
	WorkerReplicationTaskMaxRetryCount:                       "worker.replicationTaskMaxRetryCount",
	WorkerReplicationTaskMaxRetryDuration:                    "worker.replicationTaskMaxRetryDuration",
	WorkerReplicationTaskContextDuration:                     "worker.replicationTaskContextDuration",
	WorkerReReplicationContextTimeout:                        "worker.workerReReplicationContextTimeout",
	WorkerEnableReplication:                                  "worker.enableReplication",
	WorkerIndexerConcurrency:                                 "worker.indexerConcurrency",
	WorkerESProcessorNumOfWorkers:                            "worker.ESProcessorNumOfWorkers",
	WorkerESProcessorBulkActions:                             "worker.ESProcessorBulkActions",
	WorkerESProcessorBulkSize:                                "worker.ESProcessorBulkSize",
	WorkerESProcessorFlushInterval:                           "worker.ESProcessorFlushInterval",
	EnableArchivalCompression:                                "worker.EnableArchivalCompression",
	WorkerHistoryPageSize:                                    "worker.WorkerHistoryPageSize",
	WorkerTargetArchivalBlobSize:                             "worker.WorkerTargetArchivalBlobSize",
	WorkerArchiverConcurrency:                                "worker.ArchiverConcurrency",
	WorkerArchivalsPerIteration:                              "worker.ArchivalsPerIteration",
	WorkerDeterministicConstructionCheckProbability:          "worker.DeterministicConstructionCheckProbability",
	WorkerBlobIntegrityCheckProbability:                      "worker.BlobIntegrityCheckProbability",
	WorkerTimeLimitPerArchivalIteration:                      "worker.TimeLimitPerArchivalIteration",
	WorkerThrottledLogRPS:                                    "worker.throttledLogRPS",
	ScannerPersistenceMaxQPS:                                 "worker.scannerPersistenceMaxQPS",
	TaskListScannerEnabled:                                   "worker.taskListScannerEnabled",
	HistoryScannerEnabled:                                    "worker.historyScannerEnabled",
	ConcreteExecutionsScannerEnabled:                         "worker.executionsScannerEnabled",
	ConcreteExecutionsScannerBlobstoreFlushThreshold:         "worker.executionsScannerBlobstoreFlushThreshold",
	ConcreteExecutionsScannerActivityBatchSize:               "worker.executionsScannerActivityBatchSize",
	ConcreteExecutionsScannerConcurrency:                     "worker.executionsScannerConcurrency",
	ConcreteExecutionsScannerPersistencePageSize:             "worker.executionsScannerPersistencePageSize",
	ConcreteExecutionsScannerInvariantCollectionHistory:      "worker.executionsScannerInvariantCollectionHistory",
	ConcreteExecutionsScannerInvariantCollectionMutableState: "worker.executionsScannerInvariantCollectionMutableState",
	CurrentExecutionsScannerEnabled:                          "worker.currentExecutionsScannerEnabled",
	CurrentExecutionsScannerBlobstoreFlushThreshold:          "worker.currentExecutionsBlobstoreFlushThreshold",
	CurrentExecutionsScannerActivityBatchSize:                "worker.currentExecutionsActivityBatchSize",
	CurrentExecutionsScannerConcurrency:                      "worker.currentExecutionsConcurrency",
	CurrentExecutionsScannerPersistencePageSize:              "worker.currentExecutionsPersistencePageSize",
	CurrentExecutionsScannerInvariantCollectionHistory:       "worker.currentExecutionsScannerInvariantCollectionHistory",
	CurrentExecutionsScannerInvariantCollectionMutableState:  "worker.currentExecutionsInvariantCollectionMutableState",
}

const (
	unknownKey Key = iota

	// key for tests
	testGetPropertyKey
	testGetIntPropertyKey
	testGetFloat64PropertyKey
	testGetDurationPropertyKey
	testGetBoolPropertyKey
	testGetStringPropertyKey
	testGetMapPropertyKey
	testGetIntPropertyFilteredByDomainKey
	testGetDurationPropertyFilteredByDomainKey
	testGetIntPropertyFilteredByTaskListInfoKey
	testGetDurationPropertyFilteredByTaskListInfoKey
	testGetBoolPropertyFilteredByDomainIDKey
	testGetBoolPropertyFilteredByTaskListInfoKey

	// EnableGlobalDomain is key for enable global domain
	EnableGlobalDomain
	// EnableNewKafkaClient is key for using New Kafka client
	EnableNewKafkaClient
	// EnableVisibilitySampling is key for enable visibility sampling
	EnableVisibilitySampling
	// EnableReadFromClosedExecutionV2 is key for enable read from cadence_visibility.closed_executions_v2
	EnableReadFromClosedExecutionV2
	// AdvancedVisibilityWritingMode is key for how to write to advanced visibility
	AdvancedVisibilityWritingMode
	// EmitShardDiffLog whether emit the shard diff log
	EmitShardDiffLog
	// EnableReadVisibilityFromES is key for enable read from elastic search
	EnableReadVisibilityFromES
	// DisableListVisibilityByFilter is config to disable list open/close workflow using filter
	DisableListVisibilityByFilter
	// HistoryArchivalStatus is key for the status of history archival
	HistoryArchivalStatus
	// EnableReadFromHistoryArchival is key for enabling reading history from archival store
	EnableReadFromHistoryArchival
	// VisibilityArchivalStatus is key for the status of visibility archival
	VisibilityArchivalStatus
	// EnableReadFromVisibilityArchival is key for enabling reading visibility from archival store
	EnableReadFromVisibilityArchival
	// EnableDomainNotActiveAutoForwarding whether enabling DC auto forwarding to active cluster
	// for signal / start / signal with start API if domain is not active
	EnableDomainNotActiveAutoForwarding
	// EnableGracefulFailover whether enabling graceful failover
	EnableGracefulFailover
	// TransactionSizeLimit is the largest allowed transaction size to persistence
	TransactionSizeLimit
	// PersistenceErrorInjectionRate is the rate for injecting random error in persistence
	PersistenceErrorInjectionRate
	// MinRetentionDays is the minimal allowed retention days for domain
	MinRetentionDays
	// MaxDecisionStartToCloseSeconds is the minimal allowed decision start to close timeout in seconds
	MaxDecisionStartToCloseSeconds
	// DisallowQuery is the key to disallow query for a domain
	DisallowQuery
	// EnablePriorityTaskProcessor is the key for enabling priority task processor
	EnablePriorityTaskProcessor
	// EnableAuthorization is the key to enable authorization for a domain
	EnableAuthorization

	// BlobSizeLimitError is the per event blob size limit
	BlobSizeLimitError
	// BlobSizeLimitWarn is the per event blob size limit for warning
	BlobSizeLimitWarn
	// HistorySizeLimitError is the per workflow execution history size limit
	HistorySizeLimitError
	// HistorySizeLimitWarn is the per workflow execution history size limit for warning
	HistorySizeLimitWarn
	// HistoryCountLimitError is the per workflow execution history event count limit
	HistoryCountLimitError
	// HistoryCountLimitWarn is the per workflow execution history event count limit for warning
	HistoryCountLimitWarn

	// MaxIDLengthLimit is the length limit for various IDs, including: Domain, TaskList, WorkflowID, ActivityID, TimerID,
	// WorkflowType, ActivityType, SignalName, MarkerName, ErrorReason/FailureReason/CancelCause, Identity, RequestID
	MaxIDLengthLimit
	// MaxIDLengthWarnLimit is the warn length limit for various IDs, including: Domain, TaskList, WorkflowID, ActivityID, TimerID,
	// WorkflowType, ActivityType, SignalName, MarkerName, ErrorReason/FailureReason/CancelCause, Identity, RequestID
	MaxIDLengthWarnLimit
	// MaxRawTaskListNameLimit is the max length of user provided task list name (non-sticky and non-scalable)
	MaxRawTaskListNameLimit

	// key for frontend

	// FrontendPersistenceMaxQPS is the max qps frontend host can query DB
	FrontendPersistenceMaxQPS
	// FrontendPersistenceGlobalMaxQPS is the max qps frontend cluster can query DB
	FrontendPersistenceGlobalMaxQPS
	// FrontendVisibilityMaxPageSize is default max size for ListWorkflowExecutions in one page
	FrontendVisibilityMaxPageSize
	// FrontendVisibilityListMaxQPS is max qps frontend can list open/close workflows
	FrontendVisibilityListMaxQPS
	// FrontendESVisibilityListMaxQPS is max qps frontend can list open/close workflows from ElasticSearch
	FrontendESVisibilityListMaxQPS
	// FrontendESIndexMaxResultWindow is ElasticSearch index setting max_result_window
	FrontendESIndexMaxResultWindow
	// FrontendHistoryMaxPageSize is default max size for GetWorkflowExecutionHistory in one page
	FrontendHistoryMaxPageSize
	// FrontendRPS is workflow rate limit per second
	FrontendRPS
	// FrontendMaxDomainRPSPerInstance is workflow domain rate limit per second
	FrontendMaxDomainRPSPerInstance
	// FrontendGlobalDomainRPS is workflow domain rate limit per second for the whole Cadence cluster
	FrontendGlobalDomainRPS
	// FrontendHistoryMgrNumConns is for persistence cluster.NumConns
	FrontendHistoryMgrNumConns
	// FrontendThrottledLogRPS is the rate limit on number of log messages emitted per second for throttled logger
	FrontendThrottledLogRPS
	// FrontendShutdownDrainDuration is the duration of traffic drain during shutdown
	FrontendShutdownDrainDuration
	// EnableClientVersionCheck enables client version check for frontend
	EnableClientVersionCheck

	// FrontendMaxBadBinaries is the max number of bad binaries in domain config
	FrontendMaxBadBinaries
	// FrontendFailoverCoolDown is the duration between two domain failvoers
	FrontendFailoverCoolDown
	// ValidSearchAttributes is legal indexed keys that can be used in list APIs
	ValidSearchAttributes
	// SendRawWorkflowHistory is whether to enable raw history retrieving
	SendRawWorkflowHistory
	// SearchAttributesNumberOfKeysLimit is the limit of number of keys
	SearchAttributesNumberOfKeysLimit
	// SearchAttributesSizeOfValueLimit is the size limit of each value
	SearchAttributesSizeOfValueLimit
	// SearchAttributesTotalSizeLimit is the size limit of the whole map
	SearchAttributesTotalSizeLimit
	// VisibilityArchivalQueryMaxPageSize is the maximum page size for a visibility archival query
	VisibilityArchivalQueryMaxPageSize
	// VisibilityArchivalQueryMaxRangeInDays is the maximum number of days for a visibility archival query
	VisibilityArchivalQueryMaxRangeInDays
	// VisibilityArchivalQueryMaxQPS is the timeout for a visibility archival query
	VisibilityArchivalQueryMaxQPS

	// DomainFailoverRefreshInterval is the domain failover refresh timer
	DomainFailoverRefreshInterval
	// DomainFailoverRefreshTimerJitterCoefficient is the jitter for domain failover refresh timer jitter
	DomainFailoverRefreshTimerJitterCoefficient

	// key for matching

	// MatchingRPS is request rate per second for each matching host
	MatchingRPS
	// MatchingPersistenceMaxQPS is the max qps matching host can query DB
	MatchingPersistenceMaxQPS
	// MatchingPersistenceGlobalMaxQPS is the max qps matching cluster can query DB
	MatchingPersistenceGlobalMaxQPS
	// MatchingMinTaskThrottlingBurstSize is the minimum burst size for task list throttling
	MatchingMinTaskThrottlingBurstSize
	// MatchingGetTasksBatchSize is the maximum batch size to fetch from the task buffer
	MatchingGetTasksBatchSize
	// MatchingLongPollExpirationInterval is the long poll expiration interval in the matching service
	MatchingLongPollExpirationInterval
	// MatchingEnableSyncMatch is to enable sync match
	MatchingEnableSyncMatch
	// MatchingUpdateAckInterval is the interval for update ack
	MatchingUpdateAckInterval
	// MatchingIdleTasklistCheckInterval is the IdleTasklistCheckInterval
	MatchingIdleTasklistCheckInterval
	// MaxTasklistIdleTime is the max time tasklist being idle
	MaxTasklistIdleTime
	// MatchingOutstandingTaskAppendsThreshold is the threshold for outstanding task appends
	MatchingOutstandingTaskAppendsThreshold
	// MatchingMaxTaskBatchSize is max batch size for task writer
	MatchingMaxTaskBatchSize
	// MatchingMaxTaskDeleteBatchSize is the max batch size for range deletion of tasks
	MatchingMaxTaskDeleteBatchSize
	// MatchingThrottledLogRPS is the rate limit on number of log messages emitted per second for throttled logger
	MatchingThrottledLogRPS
	// MatchingNumTasklistWritePartitions is the number of write partitions for a task list
	MatchingNumTasklistWritePartitions
	// MatchingNumTasklistReadPartitions is the number of read partitions for a task list
	MatchingNumTasklistReadPartitions
	// MatchingForwarderMaxOutstandingPolls is the max number of inflight polls from the forwarder
	MatchingForwarderMaxOutstandingPolls
	// MatchingForwarderMaxOutstandingTasks is the max number of inflight addTask/queryTask from the forwarder
	MatchingForwarderMaxOutstandingTasks
	// MatchingForwarderMaxRatePerSecond is the max rate at which add/query can be forwarded
	MatchingForwarderMaxRatePerSecond
	// MatchingForwarderMaxChildrenPerNode is the max number of children per node in the task list partition tree
	MatchingForwarderMaxChildrenPerNode
	// MatchingShutdownDrainDuration is the duration of traffic drain during shutdown
	MatchingShutdownDrainDuration

	// key for history

	// HistoryRPS is request rate per second for each history host
	HistoryRPS
	// HistoryPersistenceMaxQPS is the max qps history host can query DB
	HistoryPersistenceMaxQPS
	// HistoryPersistenceGlobalMaxQPS is the max qps history cluster can query DB
	HistoryPersistenceGlobalMaxQPS
	// HistoryVisibilityOpenMaxQPS is max qps one history host can write visibility open_executions
	HistoryVisibilityOpenMaxQPS
	// HistoryVisibilityClosedMaxQPS is max qps one history host can write visibility closed_executions
	HistoryVisibilityClosedMaxQPS
	// HistoryLongPollExpirationInterval is the long poll expiration interval in the history service
	HistoryLongPollExpirationInterval
	// HistoryCacheInitialSize is initial size of history cache
	HistoryCacheInitialSize
	// HistoryCacheMaxSize is max size of history cache
	HistoryCacheMaxSize
	// HistoryCacheTTL is TTL of history cache
	HistoryCacheTTL
	// HistoryShutdownDrainDuration is the duration of traffic drain during shutdown
	HistoryShutdownDrainDuration
	// EventsCacheInitialCount is initial count of events cache
	EventsCacheInitialCount
	// EventsCacheMaxCount is max count of events cache
	EventsCacheMaxCount
	// EventsCacheMaxSize is max size of events cache in bytes
	EventsCacheMaxSize
	// EventsCacheTTL is TTL of events cache
	EventsCacheTTL
	// EventsCacheGlobalEnable enables global cache over all history shards
	EventsCacheGlobalEnable
	// EventsCacheGlobalInitialCount is initial count of global events cache
	EventsCacheGlobalInitialCount
	// EventsCacheGlobalMaxCount is max count of global events cache
	EventsCacheGlobalMaxCount
	// AcquireShardInterval is interval that timer used to acquire shard
	AcquireShardInterval
	// AcquireShardConcurrency is number of goroutines that can be used to acquire shards in the shard controller.
	AcquireShardConcurrency
	// StandbyClusterDelay is the artificial delay added to standby cluster's view of active cluster's time
	StandbyClusterDelay
	// StandbyTaskMissingEventsResendDelay is the amount of time standby cluster's will wait (if events are missing)
	// before calling remote for missing events
	StandbyTaskMissingEventsResendDelay
	// StandbyTaskMissingEventsDiscardDelay is the amount of time standby cluster's will wait (if events are missing)
	// before discarding the task
	StandbyTaskMissingEventsDiscardDelay
	// TaskProcessRPS is the task processing rate per second for each domain
	TaskProcessRPS
	// TaskSchedulerType is the task scheduler type for priority task processor
	TaskSchedulerType
	// TaskSchedulerWorkerCount is the number of workers per host in task scheduler
	TaskSchedulerWorkerCount
	// TaskSchedulerShardWorkerCount is the number of worker per shard in task scheduler
	TaskSchedulerShardWorkerCount
	// TaskSchedulerQueueSize is the size of task channel for host level task scheduler
	TaskSchedulerQueueSize
	// TaskSchedulerShardQueueSize is the size of task channel for shard level task scheduler
	TaskSchedulerShardQueueSize
	// TaskSchedulerDispatcherCount is the number of task dispatcher in task scheduler (only applies to host level task scheduler)
	TaskSchedulerDispatcherCount
	// TaskSchedulerRoundRobinWeights is the priority weight for weighted round robin task scheduler
	TaskSchedulerRoundRobinWeights
	// ActiveTaskRedispatchInterval is the active task redispatch interval
	ActiveTaskRedispatchInterval
	// StandbyTaskRedispatchInterval is the standby task redispatch interval
	StandbyTaskRedispatchInterval
	// TaskRedispatchIntervalJitterCoefficient is the task redispatch interval jitter coefficient
	TaskRedispatchIntervalJitterCoefficient
	// StandbyTaskReReplicationContextTimeout is the context timeout for standby task re-replication
	StandbyTaskReReplicationContextTimeout
	// QueueProcessorEnableSplit indicates whether processing queue split policy should be enabled
	QueueProcessorEnableSplit
	// QueueProcessorSplitMaxLevel is the max processing queue level
	QueueProcessorSplitMaxLevel
	// QueueProcessorEnableRandomSplitByDomainID indicates whether random queue split policy should be enabled for a domain
	QueueProcessorEnableRandomSplitByDomainID
	// QueueProcessorRandomSplitProbability is the probability for a domain to be split to a new processing queue
	QueueProcessorRandomSplitProbability
	// QueueProcessorEnablePendingTaskSplitByDomainID indicates whether pending task split policy should be enabled
	QueueProcessorEnablePendingTaskSplitByDomainID
	// QueueProcessorPendingTaskSplitThreshold is the threshold for the number of pending tasks per domain
	QueueProcessorPendingTaskSplitThreshold
	// QueueProcessorEnableStuckTaskSplitByDomainID indicates whether stuck task split policy should be enabled
	QueueProcessorEnableStuckTaskSplitByDomainID
	// QueueProcessorStuckTaskSplitThreshold is the threshold for the number of attempts of a task
	QueueProcessorStuckTaskSplitThreshold
	// QueueProcessorSplitLookAheadDurationByDomainID is the look ahead duration when spliting a domain to a new processing queue
	QueueProcessorSplitLookAheadDurationByDomainID
	// QueueProcessorPollBackoffInterval is the backoff duration when queue processor is throttled
	QueueProcessorPollBackoffInterval
	// QueueProcessorPollBackoffIntervalJitterCoefficient backoff interval jitter coefficient
	QueueProcessorPollBackoffIntervalJitterCoefficient
	// QueueProcessorEnablePersistQueueStates indicates whether processing queue states should be persisted
	QueueProcessorEnablePersistQueueStates
	// QueueProcessorEnableLoadQueueStates indicates whether processing queue states should be loaded
	QueueProcessorEnableLoadQueueStates
	// TimerTaskBatchSize is batch size for timer processor to process tasks
	TimerTaskBatchSize
	// TimerTaskWorkerCount is number of task workers for timer processor
	TimerTaskWorkerCount
	// TimerTaskMaxRetryCount is max retry count for timer processor
	TimerTaskMaxRetryCount
	// TimerProcessorGetFailureRetryCount is retry count for timer processor get failure operation
	TimerProcessorGetFailureRetryCount
	// TimerProcessorCompleteTimerFailureRetryCount is retry count for timer processor complete timer operation
	TimerProcessorCompleteTimerFailureRetryCount
	// TimerProcessorUpdateShardTaskCount is update shard count for timer processor
	TimerProcessorUpdateShardTaskCount
	// TimerProcessorUpdateAckInterval is update interval for timer processor
	TimerProcessorUpdateAckInterval
	// TimerProcessorUpdateAckIntervalJitterCoefficient is the update interval jitter coefficient
	TimerProcessorUpdateAckIntervalJitterCoefficient
	// TimerProcessorCompleteTimerInterval is complete timer interval for timer processor
	TimerProcessorCompleteTimerInterval
	// TimerProcessorFailoverMaxPollRPS is max poll rate per second for timer processor
	TimerProcessorFailoverMaxPollRPS
	// TimerProcessorMaxPollRPS is max poll rate per second for timer processor
	TimerProcessorMaxPollRPS
	// TimerProcessorMaxPollInterval is max poll interval for timer processor
	TimerProcessorMaxPollInterval
	// TimerProcessorMaxPollIntervalJitterCoefficient is the max poll interval jitter coefficient
	TimerProcessorMaxPollIntervalJitterCoefficient
	// TimerProcessorSplitQueueInterval is the split processing queue interval for timer processor
	TimerProcessorSplitQueueInterval
	// TimerProcessorSplitQueueIntervalJitterCoefficient is the split processing queue interval jitter coefficient
	TimerProcessorSplitQueueIntervalJitterCoefficient
	// TimerProcessorMaxRedispatchQueueSize is the threshold of the number of tasks in the redispatch queue for timer processor
	TimerProcessorMaxRedispatchQueueSize
	// TimerProcessorEnablePriorityTaskProcessor indicates whether priority task processor should be used for timer processor
	TimerProcessorEnablePriorityTaskProcessor
	// TimerProcessorEnableMultiCurosrProcessor indicates whether multi-cursor queue processor should be used for timer processor
	TimerProcessorEnableMultiCurosrProcessor
	// TimerProcessorMaxTimeShift is the max shift timer processor can have
	TimerProcessorMaxTimeShift
	// TimerProcessorHistoryArchivalSizeLimit is the max history size for inline archival
	TimerProcessorHistoryArchivalSizeLimit
	// TimerProcessorArchivalTimeLimit is the upper time limit for inline history archival
	TimerProcessorArchivalTimeLimit
	// TransferTaskBatchSize is batch size for transferQueueProcessor
	TransferTaskBatchSize
	// TransferProcessorFailoverMaxPollRPS is max poll rate per second for transferQueueProcessor
	TransferProcessorFailoverMaxPollRPS
	// TransferProcessorMaxPollRPS is max poll rate per second for transferQueueProcessor
	TransferProcessorMaxPollRPS
	// TransferTaskWorkerCount is number of worker for transferQueueProcessor
	TransferTaskWorkerCount
	// TransferTaskMaxRetryCount is max times of retry for transferQueueProcessor
	TransferTaskMaxRetryCount
	// TransferProcessorCompleteTransferFailureRetryCount is times of retry for failure
	TransferProcessorCompleteTransferFailureRetryCount
	// TransferProcessorUpdateShardTaskCount is update shard count for transferQueueProcessor
	TransferProcessorUpdateShardTaskCount
	// TransferProcessorMaxPollInterval max poll interval for transferQueueProcessor
	TransferProcessorMaxPollInterval
	// TransferProcessorMaxPollIntervalJitterCoefficient is the max poll interval jitter coefficient
	TransferProcessorMaxPollIntervalJitterCoefficient
	// TransferProcessorSplitQueueInterval is the split processing queue interval for transferQueueProcessor
	TransferProcessorSplitQueueInterval
	// TransferProcessorSplitQueueIntervalJitterCoefficient is the split processing queue interval jitter coefficient
	TransferProcessorSplitQueueIntervalJitterCoefficient
	// TransferProcessorUpdateAckInterval is update interval for transferQueueProcessor
	TransferProcessorUpdateAckInterval
	// TransferProcessorUpdateAckIntervalJitterCoefficient is the update interval jitter coefficient
	TransferProcessorUpdateAckIntervalJitterCoefficient
	// TransferProcessorCompleteTransferInterval is complete timer interval for transferQueueProcessor
	TransferProcessorCompleteTransferInterval
	// TransferProcessorMaxRedispatchQueueSize is the threshold of the number of tasks in the redispatch queue for transferQueueProcessor
	TransferProcessorMaxRedispatchQueueSize
	// TransferProcessorEnablePriorityTaskProcessor indicates whether priority task processor should be used for transferQueueProcessor
	TransferProcessorEnablePriorityTaskProcessor
	// TransferProcessorEnableMultiCurosrProcessor indicates whether multi-cursor queue processor should be used for transferQueueProcessor
	TransferProcessorEnableMultiCurosrProcessor
	// TransferProcessorVisibilityArchivalTimeLimit is the upper time limit for archiving visibility records
	TransferProcessorVisibilityArchivalTimeLimit
	// ReplicatorTaskBatchSize is batch size for ReplicatorProcessor
	ReplicatorTaskBatchSize
	// ReplicatorTaskWorkerCount is number of worker for ReplicatorProcessor
	ReplicatorTaskWorkerCount
	// ReplicatorReadTaskMaxRetryCount is the number of read replication task retry time
	ReplicatorReadTaskMaxRetryCount
	// ReplicatorTaskMaxRetryCount is max times of retry for ReplicatorProcessor
	ReplicatorTaskMaxRetryCount
	// ReplicatorProcessorMaxPollRPS is max poll rate per second for ReplicatorProcessor
	ReplicatorProcessorMaxPollRPS
	// ReplicatorProcessorUpdateShardTaskCount is update shard count for ReplicatorProcessor
	ReplicatorProcessorUpdateShardTaskCount
	// ReplicatorProcessorMaxPollInterval is max poll interval for ReplicatorProcessor
	ReplicatorProcessorMaxPollInterval
	// ReplicatorProcessorMaxPollIntervalJitterCoefficient is the max poll interval jitter coefficient
	ReplicatorProcessorMaxPollIntervalJitterCoefficient
	// ReplicatorProcessorUpdateAckInterval is update interval for ReplicatorProcessor
	ReplicatorProcessorUpdateAckInterval
	// ReplicatorProcessorUpdateAckIntervalJitterCoefficient is the update interval jitter coefficient
	ReplicatorProcessorUpdateAckIntervalJitterCoefficient
	// ReplicatorProcessorMaxRedispatchQueueSize is the threshold of the number of tasks in the redispatch queue for ReplicatorProcessor
	ReplicatorProcessorMaxRedispatchQueueSize
	// ReplicatorProcessorEnablePriorityTaskProcessor indicates whether priority task processor should be used for ReplicatorProcessor
	ReplicatorProcessorEnablePriorityTaskProcessor
	// ExecutionMgrNumConns is persistence connections number for ExecutionManager
	ExecutionMgrNumConns
	// HistoryMgrNumConns is persistence connections number for HistoryManager
	HistoryMgrNumConns
	// MaximumBufferedEventsBatch is max number of buffer event in mutable state
	MaximumBufferedEventsBatch
	// MaximumSignalsPerExecution is max number of signals supported by single execution
	MaximumSignalsPerExecution
	// ShardUpdateMinInterval is the minimal time interval which the shard info can be updated
	ShardUpdateMinInterval
	// ShardSyncMinInterval is the minimal time interval which the shard info should be sync to remote
	ShardSyncMinInterval
	// ShardSyncTimerJitterCoefficient is the sync shard jitter coefficient
	ShardSyncTimerJitterCoefficient
	// DefaultEventEncoding is the encoding type for history events
	DefaultEventEncoding
	// NumArchiveSystemWorkflows is key for number of archive system workflows running in total
	NumArchiveSystemWorkflows
	// ArchiveRequestRPS is the rate limit on the number of archive request per second
	ArchiveRequestRPS

	// EnableAdminProtection is whether to enable admin checking
	EnableAdminProtection
	// AdminOperationToken is the token to pass admin checking
	AdminOperationToken
	// HistoryMaxAutoResetPoints is the key for max number of auto reset points stored in mutableState
	HistoryMaxAutoResetPoints

	// EnableParentClosePolicy whether to  ParentClosePolicy
	EnableParentClosePolicy
	// ParentClosePolicyThreshold decides that parent close policy will be processed by sys workers(if enabled) if
	// the number of children greater than or equal to this threshold
	ParentClosePolicyThreshold
	// NumParentClosePolicySystemWorkflows is key for number of parentClosePolicy system workflows running in total
	NumParentClosePolicySystemWorkflows

	// HistoryThrottledLogRPS is the rate limit on number of log messages emitted per second for throttled logger
	HistoryThrottledLogRPS
	// StickyTTL is to expire a sticky tasklist if no update more than this duration
	StickyTTL
	// DecisionHeartbeatTimeout for decision heartbeat
	DecisionHeartbeatTimeout

	// EnableDropStuckTaskByDomainID is whether stuck timer/transfer task should be dropped for a domain
	EnableDropStuckTaskByDomainID

	// key for worker

	// WorkerPersistenceMaxQPS is the max qps worker host can query DB
	WorkerPersistenceMaxQPS
	// WorkerPersistenceGlobalMaxQPS is the max qps worker cluster can query DB
	WorkerPersistenceGlobalMaxQPS
	// WorkerReplicatorMetaTaskConcurrency is the number of coroutine handling metadata related tasks
	WorkerReplicatorMetaTaskConcurrency
	// WorkerReplicatorTaskConcurrency is the number of coroutine handling non metadata related tasks
	WorkerReplicatorTaskConcurrency
	// WorkerReplicatorMessageConcurrency is the max concurrent tasks provided by messaging client
	WorkerReplicatorMessageConcurrency
	// WorkerReplicatorActivityBufferRetryCount is the retry attempt when encounter retry error on activity
	WorkerReplicatorActivityBufferRetryCount
	// WorkerReplicatorHistoryBufferRetryCount is the retry attempt when encounter retry error on history
	WorkerReplicatorHistoryBufferRetryCount
	// WorkerReplicationTaskMaxRetryCount is the max retry count for any task
	WorkerReplicationTaskMaxRetryCount
	// WorkerReplicationTaskMaxRetryDuration is the max retry duration for any task
	WorkerReplicationTaskMaxRetryDuration
	// WorkerReplicationTaskContextDuration is the context timeout for apply replication tasks
	WorkerReplicationTaskContextDuration
	// WorkerReReplicationContextTimeout is the context timeout for end to end  re-replication process
	WorkerReReplicationContextTimeout
	// WorkerEnableReplication is the feature flag for kafka replication
	WorkerEnableReplication
	// WorkerIndexerConcurrency is the max concurrent messages to be processed at any given time
	WorkerIndexerConcurrency
	// WorkerESProcessorNumOfWorkers is num of workers for esProcessor
	WorkerESProcessorNumOfWorkers
	// WorkerESProcessorBulkActions is max number of requests in bulk for esProcessor
	WorkerESProcessorBulkActions
	// WorkerESProcessorBulkSize is max total size of bulk in bytes for esProcessor
	WorkerESProcessorBulkSize
	// WorkerESProcessorFlushInterval is flush interval for esProcessor
	WorkerESProcessorFlushInterval
	// EnableArchivalCompression indicates whether blobs are compressed before they are archived
	EnableArchivalCompression
	// WorkerHistoryPageSize indicates the page size of history fetched from persistence for archival
	WorkerHistoryPageSize
	// WorkerTargetArchivalBlobSize indicates the target blob size in bytes for archival, actual blob size may vary
	WorkerTargetArchivalBlobSize
	// WorkerArchiverConcurrency controls the number of coroutines handling archival work per archival workflow
	WorkerArchiverConcurrency
	// WorkerArchivalsPerIteration controls the number of archivals handled in each iteration of archival workflow
	WorkerArchivalsPerIteration
	// WorkerDeterministicConstructionCheckProbability controls the probability of running a deterministic construction check for any given archival
	WorkerDeterministicConstructionCheckProbability
	// WorkerBlobIntegrityCheckProbability controls the probability of running an integrity check for any given archival
	WorkerBlobIntegrityCheckProbability
	// WorkerTimeLimitPerArchivalIteration controls the time limit of each iteration of archival workflow
	WorkerTimeLimitPerArchivalIteration
	// WorkerThrottledLogRPS is the rate limit on number of log messages emitted per second for throttled logger
	WorkerThrottledLogRPS
	// ScannerPersistenceMaxQPS is the maximum rate of persistence calls from worker.Scanner
	ScannerPersistenceMaxQPS
	// TaskListScannerEnabled indicates if task list scanner should be started as part of worker.Scanner
	TaskListScannerEnabled
	// HistoryScannerEnabled indicates if history scanner should be started as part of worker.Scanner
	HistoryScannerEnabled
	// ConcreteExecutionsScannerEnabled indicates if executions scanner should be started as part of worker.Scanner
	ConcreteExecutionsScannerEnabled
	// ConcreteExecutionsScannerConcurrency indicates the concurrency of concrete execution scanner
	ConcreteExecutionsScannerConcurrency
	// ConcreteExecutionsScannerBlobstoreFlushThreshold indicates the flush threshold of blobstore in concrete execution scanner
	ConcreteExecutionsScannerBlobstoreFlushThreshold
	// ConcreteExecutionsScannerActivityBatchSize indicates the batch size of scanner activities
	ConcreteExecutionsScannerActivityBatchSize
	// ConcreteExecutionsScannerPersistencePageSize indicates the page size of execution persistence fetches in concrete execution scanner
	ConcreteExecutionsScannerPersistencePageSize
	// ConcreteExecutionsScannerInvariantCollectionMutableState indicates if mutable state invariant checks should be run
	ConcreteExecutionsScannerInvariantCollectionMutableState
	// ConcreteExecutionsScannerInvariantCollectionHistory indicates if history invariant checks should be run
	ConcreteExecutionsScannerInvariantCollectionHistory
	// CurrentExecutionsScannerEnabled indicates if current executions scanner should be started as part of worker.Scanner
	CurrentExecutionsScannerEnabled
	// CurrentExecutionsScannerConcurrency indicates the concurrency of current executions scanner
	CurrentExecutionsScannerConcurrency
	// CurrentExecutionsScannerBlobstoreFlushThreshold indicates the flush threshold of blobstore in current executions scanner
	CurrentExecutionsScannerBlobstoreFlushThreshold
	// CurrentExecutionsScannerActivityBatchSize indicates the batch size of scanner activities
	CurrentExecutionsScannerActivityBatchSize
	// CurrentExecutionsScannerPersistencePageSize indicates the page size of execution persistence fetches in current executions scanner
	CurrentExecutionsScannerPersistencePageSize
	// CurrentExecutionsScannerInvariantCollectionHistory indicates if history invariant checks should be run
	CurrentExecutionsScannerInvariantCollectionHistory
	// CurrentExecutionsScannerInvariantCollectionMutableState indicates if mutable state invariant checks should be run
	CurrentExecutionsScannerInvariantCollectionMutableState
	// EnableBatcher decides whether start batcher in our worker
	EnableBatcher
	// EnableParentClosePolicyWorker decides whether or not enable system workers for processing parent close policy task
	EnableParentClosePolicyWorker
	// EnableStickyQuery indicates if sticky query should be enabled per domain
	EnableStickyQuery
	// EnableFailoverManager indicates if failover manager is enabled
	EnableFailoverManager

	//ReplicationTaskFetcherParallelism determines how many go routines we spin up for fetching tasks
	ReplicationTaskFetcherParallelism
	// ReplicationTaskFetcherAggregationInterval determines how frequently the fetch requests are sent
	ReplicationTaskFetcherAggregationInterval
	// ReplicationTaskFetcherTimerJitterCoefficient is the jitter for fetcher timer
	ReplicationTaskFetcherTimerJitterCoefficient
	// ReplicationTaskFetcherErrorRetryWait is the wait time when fetcher encounters error
	ReplicationTaskFetcherErrorRetryWait
	// ReplicationTaskFetcherServiceBusyWait is the wait time when fetcher encounters service busy error
	ReplicationTaskFetcherServiceBusyWait
	// ReplicationTaskProcessorErrorRetryWait is the initial retry wait when we see errors in applying replication tasks
	ReplicationTaskProcessorErrorRetryWait
	// ReplicationTaskProcessorErrorRetryMaxAttempts is the max retry attempts for applying replication tasks
	ReplicationTaskProcessorErrorRetryMaxAttempts
	// ReplicationTaskProcessorErrorSecondRetryWait is the initial retry wait for the second phase retry
	ReplicationTaskProcessorErrorSecondRetryWait
	// ReplicationTaskProcessorErrorSecondRetryMaxWait is the max wait time for the second phase retry
	ReplicationTaskProcessorErrorSecondRetryMaxWait
	// ReplicationTaskProcessorErrorSecondRetryExpiration is the expiration duration for the second phase retry
	ReplicationTaskProcessorErrorSecondRetryExpiration
	// ReplicationTaskProcessorNoTaskInitialWait is the wait time when not ask is returned
	ReplicationTaskProcessorNoTaskInitialWait
	// ReplicationTaskProcessorCleanupInterval determines how frequently the cleanup replication queue
	ReplicationTaskProcessorCleanupInterval
	// ReplicationTaskProcessorCleanupJitterCoefficient is the jitter for cleanup timer
	ReplicationTaskProcessorCleanupJitterCoefficient
	// ReplicationTaskProcessorReadHistoryBatchSize is the batch size to read history events
	ReplicationTaskProcessorReadHistoryBatchSize
	// ReplicationTaskProcessorStartWait is the wait time before each task processing batch
	ReplicationTaskProcessorStartWait
	// ReplicationTaskProcessorStartWaitJitterCoefficient is the jitter for batch start wait timer
	ReplicationTaskProcessorStartWaitJitterCoefficient
	// ReplicationTaskProcessorHostQPS is the qps of task processing rate limiter on host level
	ReplicationTaskProcessorHostQPS
	// ReplicationTaskProcessorShardQPS is the qps of task processing rate limiter on shard level
	ReplicationTaskProcessorShardQPS
	//ReplicationTaskGenerationQPS is the wait time between each replication task generation qps
	ReplicationTaskGenerationQPS
	// EnableConsistentQuery indicates if consistent query is enabled for the cluster
	EnableConsistentQuery
	// EnableConsistentQueryByDomain indicates if consistent query is enabled for a domain
	EnableConsistentQueryByDomain
	// MaxBufferedQueryCount indicates the maximum number of queries which can be buffered at a given time for a single workflow
	MaxBufferedQueryCount
	// MutableStateChecksumGenProbability is the probability [0-100] that checksum will be generated for mutable state
	MutableStateChecksumGenProbability
	// MutableStateChecksumVerifyProbability is the probability [0-100] that checksum will be verified for mutable state
	MutableStateChecksumVerifyProbability
	// MutableStateChecksumInvalidateBefore is the epoch timestamp before which all checksums are to be discarded
	MutableStateChecksumInvalidateBefore

	// ReplicationEventsFromCurrentCluster is a feature flag to allow cross DC replicate events that generated from the current cluster
	ReplicationEventsFromCurrentCluster

	// NotifyFailoverMarkerInterval determines the frequency to notify failover marker
	NotifyFailoverMarkerInterval
	// NotifyFailoverMarkerTimerJitterCoefficient is the jitter for failover marker notifier timer
	NotifyFailoverMarkerTimerJitterCoefficient

	// lastKeyForTest must be the last one in this const group for testing purpose
	lastKeyForTest

	// EnableActivityLocalDispatchByDomain allows worker to dispatch activity tasks through local tunnel after decisions are made. This is an performance optimization to skip activity scheduling efforts.
	EnableActivityLocalDispatchByDomain
)

// Filter represents a filter on the dynamic config key
type Filter int

func (f Filter) String() string {
	if f <= unknownFilter || f > ClusterName {
		return filters[unknownFilter]
	}
	return filters[f]
}

func parseFilter(filterName string) Filter {
	switch filterName {
	case "domainName":
		return DomainName
	case "domainID":
		return DomainID
	case "taskListName":
		return TaskListName
	case "taskType":
		return TaskType
	case "shardID":
		return ShardID
	case "clusterName":
		return ClusterName
	default:
		return unknownFilter
	}
}

var filters = []string{
	"unknownFilter",
	"domainName",
	"domainID",
	"taskListName",
	"taskType",
	"shardID",
	"clusterName",
}

const (
	unknownFilter Filter = iota
	// DomainName is the domain name
	DomainName
	// DomainID is the domain id
	DomainID
	// TaskListName is the tasklist name
	TaskListName
	// TaskType is the task type (0:Decision, 1:Activity)
	TaskType
	// ShardID is the shard id
	ShardID
	// ClusterName is the cluster name in a multi-region setup
	ClusterName

	// lastFilterTypeForTest must be the last one in this const group for testing purpose
	lastFilterTypeForTest
)

// FilterOption is used to provide filters for dynamic config keys
type FilterOption func(filterMap map[Filter]interface{})

// TaskListFilter filters by task list name
func TaskListFilter(name string) FilterOption {
	return func(filterMap map[Filter]interface{}) {
		filterMap[TaskListName] = name
	}
}

// DomainFilter filters by domain name
func DomainFilter(name string) FilterOption {
	return func(filterMap map[Filter]interface{}) {
		filterMap[DomainName] = name
	}
}

// DomainIDFilter filters by domain id
func DomainIDFilter(domainID string) FilterOption {
	return func(filterMap map[Filter]interface{}) {
		filterMap[DomainID] = domainID
	}
}

// TaskTypeFilter filters by task type
func TaskTypeFilter(taskType int) FilterOption {
	return func(filterMap map[Filter]interface{}) {
		filterMap[TaskType] = taskType
	}
}

// ShardIDFilter filters by shard id
func ShardIDFilter(shardID int) FilterOption {
	return func(filterMap map[Filter]interface{}) {
		filterMap[ShardID] = shardID
	}
}

// ClusterNameFilter filters by cluster name
func ClusterNameFilter(clusterName string) FilterOption {
	return func(filterMap map[Filter]interface{}) {
		filterMap[ClusterName] = clusterName
	}
}
