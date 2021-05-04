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

package replicator

import (
	"context"
	"fmt"

	h "github.com/uber/cadence/.gen/go/history"
	"github.com/uber/cadence/client"
	"github.com/uber/cadence/client/admin"
	"github.com/uber/cadence/client/history"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/cluster"
	"github.com/uber/cadence/common/domain"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/membership"
	"github.com/uber/cadence/common/messaging"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/service/config"
	"github.com/uber/cadence/common/service/dynamicconfig"
	"github.com/uber/cadence/common/task"
	"github.com/uber/cadence/common/xdc"
)

type (
	// Replicator is the processor for replication tasks
	Replicator struct {
		domainCache                   cache.DomainCache
		clusterMetadata               cluster.Metadata
		domainReplicationTaskExecutor domain.ReplicationTaskExecutor
		clientBean                    client.Bean
		historyClient                 history.Client
		config                        *Config
		client                        messaging.Client
		processors                    []*replicationTaskProcessor
		domainProcessors              []*domainReplicationMessageProcessor
		logger                        log.Logger
		metricsClient                 metrics.Client
		historySerializer             persistence.PayloadSerializer
		hostInfo                      *membership.HostInfo
		serviceResolver               membership.ServiceResolver
		domainReplicationQueue        persistence.DomainReplicationQueue
	}

	// Config contains all the replication config for worker
	Config struct {
		PersistenceMaxQPS                  dynamicconfig.IntPropertyFn
		ReplicatorMetaTaskConcurrency      dynamicconfig.IntPropertyFn
		ReplicatorTaskConcurrency          dynamicconfig.IntPropertyFn
		ReplicatorMessageConcurrency       dynamicconfig.IntPropertyFn
		ReplicatorActivityBufferRetryCount dynamicconfig.IntPropertyFn
		ReplicatorHistoryBufferRetryCount  dynamicconfig.IntPropertyFn
		ReplicationTaskMaxRetryCount       dynamicconfig.IntPropertyFn
		ReplicationTaskMaxRetryDuration    dynamicconfig.DurationPropertyFn
		ReplicationTaskContextTimeout      dynamicconfig.DurationPropertyFn
		ReReplicationContextTimeout        dynamicconfig.DurationPropertyFnWithDomainIDFilter
		EnableRPCReplication               dynamicconfig.BoolPropertyFn
	}
)

// NewReplicator creates a new replicator for processing replication tasks
func NewReplicator(
	clusterMetadata cluster.Metadata,
	metadataManagerV2 persistence.MetadataManager,
	domainCache cache.DomainCache,
	clientBean client.Bean,
	config *Config,
	client messaging.Client,
	logger log.Logger,
	metricsClient metrics.Client,
	hostInfo *membership.HostInfo,
	serviceResolver membership.ServiceResolver,
	domainReplicationQueue persistence.DomainReplicationQueue,
	domainReplicationTaskExecutor domain.ReplicationTaskExecutor,
) *Replicator {

	logger = logger.WithTags(tag.ComponentReplicator)
	return &Replicator{
		hostInfo:                      hostInfo,
		serviceResolver:               serviceResolver,
		domainCache:                   domainCache,
		clusterMetadata:               clusterMetadata,
		domainReplicationTaskExecutor: domainReplicationTaskExecutor,
		clientBean:                    clientBean,
		historyClient:                 clientBean.GetHistoryClient(),
		config:                        config,
		client:                        client,
		logger:                        logger,
		metricsClient:                 metricsClient,
		historySerializer:             persistence.NewPayloadSerializer(),
		domainReplicationQueue:        domainReplicationQueue,
	}
}

// Start is called to start replicator
func (r *Replicator) Start() error {
	currentClusterName := r.clusterMetadata.GetCurrentClusterName()
	replicationConsumerConfig := r.clusterMetadata.GetReplicationConsumerConfig()
	for clusterName, info := range r.clusterMetadata.GetAllClusterInfo() {
		if !info.Enabled {
			continue
		}

		if clusterName != currentClusterName {
			if replicationConsumerConfig.Type == config.ReplicationConsumerTypeRPC && r.config.EnableRPCReplication() {
				processor := newDomainReplicationMessageProcessor(
					clusterName,
					r.logger.WithTags(tag.ComponentReplicationTaskProcessor, tag.SourceCluster(clusterName)),
					r.clientBean.GetRemoteAdminClient(clusterName),
					r.metricsClient,
					r.domainReplicationTaskExecutor,
					r.hostInfo,
					r.serviceResolver,
					r.domainReplicationQueue,
				)
				r.domainProcessors = append(r.domainProcessors, processor)
			} else {
				r.createKafkaProcessors(currentClusterName, clusterName)
			}
		}
	}

	for _, processor := range r.processors {
		if err := processor.Start(); err != nil {
			return err
		}
	}

	for _, domainProcessor := range r.domainProcessors {
		domainProcessor.Start()
	}

	return nil
}

func (r *Replicator) createKafkaProcessors(currentClusterName string, clusterName string) {
	consumerName := getConsumerName(currentClusterName, clusterName)
	adminClient := admin.NewRetryableClient(
		r.clientBean.GetRemoteAdminClient(clusterName),
		common.CreateAdminServiceRetryPolicy(),
		common.IsServiceTransientError,
	)
	historyClient := history.NewRetryableClient(
		r.historyClient,
		common.CreateHistoryServiceRetryPolicy(),
		common.IsServiceTransientError,
	)
	logger := r.logger.WithTags(tag.ComponentReplicationTaskProcessor, tag.SourceCluster(clusterName), tag.KafkaConsumerName(consumerName))
	historyRereplicator := xdc.NewHistoryRereplicator(
		currentClusterName,
		r.domainCache,
		adminClient,
		func(ctx context.Context, request *h.ReplicateRawEventsRequest) error {
			return historyClient.ReplicateRawEvents(ctx, request)
		},
		r.historySerializer,
		r.config.ReplicationTaskContextTimeout(),
		r.config.ReReplicationContextTimeout,
		r.logger,
	)
	nDCHistoryReplicator := xdc.NewNDCHistoryResender(
		r.domainCache,
		adminClient,
		func(ctx context.Context, request *h.ReplicateEventsV2Request) error {
			return historyClient.ReplicateEventsV2(ctx, request)
		},
		r.historySerializer,
		r.config.ReReplicationContextTimeout,
		logger,
	)
	r.processors = append(r.processors, newReplicationTaskProcessor(
		currentClusterName,
		clusterName,
		consumerName,
		r.client,
		r.config,
		logger,
		r.metricsClient,
		r.domainReplicationTaskExecutor,
		historyRereplicator,
		nDCHistoryReplicator,
		r.historyClient,
		r.domainCache,
		task.NewSequentialTaskProcessor(
			r.config.ReplicatorTaskConcurrency(),
			replicationSequentialTaskQueueHashFn,
			newReplicationSequentialTaskQueue,
			r.metricsClient,
			logger,
		),
	))
}

// Stop is called to stop replicator
func (r *Replicator) Stop() {
	for _, processor := range r.processors {
		processor.Stop()
	}

	for _, domainProcessor := range r.domainProcessors {
		domainProcessor.Stop()
	}

	r.domainCache.Stop()
}

func getConsumerName(currentCluster, remoteCluster string) string {
	return fmt.Sprintf("%v_consumer_for_%v", currentCluster, remoteCluster)
}
