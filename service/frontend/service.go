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

package frontend

import (
	"sync/atomic"
	"time"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/client"
	"github.com/uber/cadence/common/definition"
	"github.com/uber/cadence/common/domain"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/messaging"
	"github.com/uber/cadence/common/persistence"
	persistenceClient "github.com/uber/cadence/common/persistence/client"
	espersistence "github.com/uber/cadence/common/persistence/elasticsearch"
	"github.com/uber/cadence/common/resource"
	"github.com/uber/cadence/common/service"
	"github.com/uber/cadence/common/service/config"
	"github.com/uber/cadence/common/service/dynamicconfig"
)

// Config represents configuration for cadence-frontend service
type Config struct {
	NumHistoryShards                int
	domainConfig                    domain.Config
	PersistenceMaxQPS               dynamicconfig.IntPropertyFn
	PersistenceGlobalMaxQPS         dynamicconfig.IntPropertyFn
	VisibilityMaxPageSize           dynamicconfig.IntPropertyFnWithDomainFilter
	EnableVisibilitySampling        dynamicconfig.BoolPropertyFn
	EnableReadFromClosedExecutionV2 dynamicconfig.BoolPropertyFn
	VisibilityListMaxQPS            dynamicconfig.IntPropertyFnWithDomainFilter
	EnableReadVisibilityFromES      dynamicconfig.BoolPropertyFnWithDomainFilter
	ESVisibilityListMaxQPS          dynamicconfig.IntPropertyFnWithDomainFilter
	ESIndexMaxResultWindow          dynamicconfig.IntPropertyFn
	HistoryMaxPageSize              dynamicconfig.IntPropertyFnWithDomainFilter
	RPS                             dynamicconfig.IntPropertyFn
	MaxDomainRPSPerInstance         dynamicconfig.IntPropertyFnWithDomainFilter
	GlobalDomainRPS                 dynamicconfig.IntPropertyFnWithDomainFilter
	MaxIDLengthLimit                dynamicconfig.IntPropertyFnWithDomainFilter
	MaxIDLengthWarnLimit            dynamicconfig.IntPropertyFn
	MaxRawTaskListNameLimit         dynamicconfig.IntPropertyFnWithDomainFilter
	EnableClientVersionCheck        dynamicconfig.BoolPropertyFn
	DisallowQuery                   dynamicconfig.BoolPropertyFnWithDomainFilter
	ShutdownDrainDuration           dynamicconfig.DurationPropertyFn

	// Persistence settings
	HistoryMgrNumConns dynamicconfig.IntPropertyFn

	// security protection settings
	EnableAdminProtection         dynamicconfig.BoolPropertyFn
	AdminOperationToken           dynamicconfig.StringPropertyFn
	DisableListVisibilityByFilter dynamicconfig.BoolPropertyFnWithDomainFilter

	// size limit system protection
	BlobSizeLimitError dynamicconfig.IntPropertyFnWithDomainFilter
	BlobSizeLimitWarn  dynamicconfig.IntPropertyFnWithDomainFilter

	ThrottledLogRPS dynamicconfig.IntPropertyFn

	// Domain specific config
	EnableDomainNotActiveAutoForwarding         dynamicconfig.BoolPropertyFnWithDomainFilter
	EnableGracefulFailover                      dynamicconfig.BoolPropertyFn
	DomainFailoverRefreshInterval               dynamicconfig.DurationPropertyFn
	DomainFailoverRefreshTimerJitterCoefficient dynamicconfig.FloatPropertyFn

	// ValidSearchAttributes is legal indexed keys that can be used in list APIs
	ValidSearchAttributes             dynamicconfig.MapPropertyFn
	SearchAttributesNumberOfKeysLimit dynamicconfig.IntPropertyFnWithDomainFilter
	SearchAttributesSizeOfValueLimit  dynamicconfig.IntPropertyFnWithDomainFilter
	SearchAttributesTotalSizeLimit    dynamicconfig.IntPropertyFnWithDomainFilter

	// VisibilityArchival system protection
	VisibilityArchivalQueryMaxPageSize dynamicconfig.IntPropertyFn

	SendRawWorkflowHistory dynamicconfig.BoolPropertyFnWithDomainFilter
}

// NewConfig returns new service config with default values
func NewConfig(dc *dynamicconfig.Collection, numHistoryShards int, enableReadFromES bool, sendRawWorkflowHistory bool) *Config {
	return &Config{
		NumHistoryShards:                            numHistoryShards,
		PersistenceMaxQPS:                           dc.GetIntProperty(dynamicconfig.FrontendPersistenceMaxQPS, 2000),
		PersistenceGlobalMaxQPS:                     dc.GetIntProperty(dynamicconfig.FrontendPersistenceGlobalMaxQPS, 0),
		VisibilityMaxPageSize:                       dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendVisibilityMaxPageSize, 1000),
		EnableVisibilitySampling:                    dc.GetBoolProperty(dynamicconfig.EnableVisibilitySampling, true),
		EnableReadFromClosedExecutionV2:             dc.GetBoolProperty(dynamicconfig.EnableReadFromClosedExecutionV2, false),
		VisibilityListMaxQPS:                        dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendVisibilityListMaxQPS, 1),
		EnableReadVisibilityFromES:                  dc.GetBoolPropertyFilteredByDomain(dynamicconfig.EnableReadVisibilityFromES, enableReadFromES),
		ESVisibilityListMaxQPS:                      dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendESVisibilityListMaxQPS, 3),
		ESIndexMaxResultWindow:                      dc.GetIntProperty(dynamicconfig.FrontendESIndexMaxResultWindow, 10000),
		HistoryMaxPageSize:                          dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendHistoryMaxPageSize, common.GetHistoryMaxPageSize),
		RPS:                                         dc.GetIntProperty(dynamicconfig.FrontendRPS, 1200),
		MaxDomainRPSPerInstance:                     dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendMaxDomainRPSPerInstance, 1200),
		GlobalDomainRPS:                             dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendGlobalDomainRPS, 0),
		MaxIDLengthLimit:                            dc.GetIntPropertyFilteredByDomain(dynamicconfig.MaxIDLengthLimit, 1000),
		MaxIDLengthWarnLimit:                        dc.GetIntProperty(dynamicconfig.MaxIDLengthWarnLimit, 128),
		MaxRawTaskListNameLimit:                     dc.GetIntPropertyFilteredByDomain(dynamicconfig.MaxRawTaskListNameLimit, 1000),
		HistoryMgrNumConns:                          dc.GetIntProperty(dynamicconfig.FrontendHistoryMgrNumConns, 10),
		EnableAdminProtection:                       dc.GetBoolProperty(dynamicconfig.EnableAdminProtection, false),
		AdminOperationToken:                         dc.GetStringProperty(dynamicconfig.AdminOperationToken, common.DefaultAdminOperationToken),
		DisableListVisibilityByFilter:               dc.GetBoolPropertyFilteredByDomain(dynamicconfig.DisableListVisibilityByFilter, false),
		BlobSizeLimitError:                          dc.GetIntPropertyFilteredByDomain(dynamicconfig.BlobSizeLimitError, 2*1024*1024),
		BlobSizeLimitWarn:                           dc.GetIntPropertyFilteredByDomain(dynamicconfig.BlobSizeLimitWarn, 256*1024),
		ThrottledLogRPS:                             dc.GetIntProperty(dynamicconfig.FrontendThrottledLogRPS, 20),
		ShutdownDrainDuration:                       dc.GetDurationProperty(dynamicconfig.FrontendShutdownDrainDuration, 0),
		EnableDomainNotActiveAutoForwarding:         dc.GetBoolPropertyFilteredByDomain(dynamicconfig.EnableDomainNotActiveAutoForwarding, true),
		EnableGracefulFailover:                      dc.GetBoolProperty(dynamicconfig.EnableGracefulFailover, false),
		DomainFailoverRefreshInterval:               dc.GetDurationProperty(dynamicconfig.DomainFailoverRefreshInterval, 10*time.Second),
		DomainFailoverRefreshTimerJitterCoefficient: dc.GetFloat64Property(dynamicconfig.DomainFailoverRefreshTimerJitterCoefficient, 0.1),
		EnableClientVersionCheck:                    dc.GetBoolProperty(dynamicconfig.EnableClientVersionCheck, false),
		ValidSearchAttributes:                       dc.GetMapProperty(dynamicconfig.ValidSearchAttributes, definition.GetDefaultIndexedKeys()),
		SearchAttributesNumberOfKeysLimit:           dc.GetIntPropertyFilteredByDomain(dynamicconfig.SearchAttributesNumberOfKeysLimit, 100),
		SearchAttributesSizeOfValueLimit:            dc.GetIntPropertyFilteredByDomain(dynamicconfig.SearchAttributesSizeOfValueLimit, 2*1024),
		SearchAttributesTotalSizeLimit:              dc.GetIntPropertyFilteredByDomain(dynamicconfig.SearchAttributesTotalSizeLimit, 40*1024),
		VisibilityArchivalQueryMaxPageSize:          dc.GetIntProperty(dynamicconfig.VisibilityArchivalQueryMaxPageSize, 10000),
		DisallowQuery:                               dc.GetBoolPropertyFilteredByDomain(dynamicconfig.DisallowQuery, false),
		SendRawWorkflowHistory:                      dc.GetBoolPropertyFilteredByDomain(dynamicconfig.SendRawWorkflowHistory, sendRawWorkflowHistory),
		domainConfig: domain.Config{
			MaxBadBinaryCount: dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendMaxBadBinaries, domain.MaxBadBinaries),
			MinRetentionDays:  dc.GetIntProperty(dynamicconfig.MinRetentionDays, domain.MinRetentionDays),
			FailoverCoolDown:  dc.GetDurationPropertyFilteredByDomain(dynamicconfig.FrontendFailoverCoolDown, domain.FailoverCoolDown),
		},
	}
}

// Service represents the cadence-frontend service
type Service struct {
	resource.Resource

	status       int32
	handler      *WorkflowHandler
	adminHandler *adminHandlerImpl
	stopC        chan struct{}
	config       *Config
	params       *service.BootstrapParams
}

// NewService builds a new cadence-frontend service
func NewService(
	params *service.BootstrapParams,
) (resource.Resource, error) {

	isAdvancedVisExistInConfig := len(params.PersistenceConfig.AdvancedVisibilityStore) != 0
	serviceConfig := NewConfig(
		dynamicconfig.NewCollection(
			params.DynamicConfig,
			params.Logger,
			dynamicconfig.ClusterNameFilter(params.ClusterMetadata.GetCurrentClusterName()),
		),
		params.PersistenceConfig.NumHistoryShards,
		isAdvancedVisExistInConfig,
		false,
	)

	params.PersistenceConfig.HistoryMaxConns = serviceConfig.HistoryMgrNumConns()
	params.PersistenceConfig.VisibilityConfig = &config.VisibilityConfig{
		VisibilityListMaxQPS:            serviceConfig.VisibilityListMaxQPS,
		EnableSampling:                  serviceConfig.EnableVisibilitySampling,
		EnableReadFromClosedExecutionV2: serviceConfig.EnableReadFromClosedExecutionV2,
	}

	visibilityManagerInitializer := func(
		persistenceBean persistenceClient.Bean,
		logger log.Logger,
	) (persistence.VisibilityManager, error) {
		visibilityFromDB := persistenceBean.GetVisibilityManager()

		var visibilityFromES persistence.VisibilityManager
		if params.ESConfig != nil {
			visibilityIndexName := params.ESConfig.Indices[common.VisibilityAppName]
			visibilityConfigForES := &config.VisibilityConfig{
				MaxQPS:                 serviceConfig.PersistenceMaxQPS,
				VisibilityListMaxQPS:   serviceConfig.ESVisibilityListMaxQPS,
				ESIndexMaxResultWindow: serviceConfig.ESIndexMaxResultWindow,
				ValidSearchAttributes:  serviceConfig.ValidSearchAttributes,
			}
			visibilityFromES = espersistence.NewESVisibilityManager(visibilityIndexName, params.ESClient, visibilityConfigForES,
				nil, params.MetricsClient, logger)
		}
		return persistence.NewVisibilityManagerWrapper(
			visibilityFromDB,
			visibilityFromES,
			serviceConfig.EnableReadVisibilityFromES,
			dynamicconfig.GetStringPropertyFn(common.AdvancedVisibilityWritingModeOff), // frontend visibility never write
		), nil
	}

	serviceResource, err := resource.New(
		params,
		common.FrontendServiceName,
		serviceConfig.PersistenceMaxQPS,
		serviceConfig.PersistenceGlobalMaxQPS,
		serviceConfig.ThrottledLogRPS,
		visibilityManagerInitializer,
	)
	if err != nil {
		return nil, err
	}

	return &Service{
		Resource: serviceResource,
		status:   common.DaemonStatusInitialized,
		config:   serviceConfig,
		stopC:    make(chan struct{}),
		params:   params,
	}, nil
}

// Start starts the service
func (s *Service) Start() {
	if !atomic.CompareAndSwapInt32(&s.status, common.DaemonStatusInitialized, common.DaemonStatusStarted) {
		return
	}

	logger := s.GetLogger()
	logger.Info("frontend starting")

	var replicationMessageSink messaging.Producer
	clusterMetadata := s.GetClusterMetadata()
	if clusterMetadata.IsGlobalDomainEnabled() {
		replicationMessageSink = s.GetDomainReplicationQueue()
	} else {
		replicationMessageSink = messaging.NewNoopProducer()
	}

	// Base handler
	s.handler = NewWorkflowHandler(s, s.config, replicationMessageSink, client.NewVersionChecker())

	// Additional decorations
	var handler Handler = s.handler
	handler = NewDCRedirectionHandler(handler, s, s.config, s.params.DCRedirectionPolicy)
	if s.params.Authorizer != nil {
		handler = NewAccessControlledHandlerImpl(handler, s, s.params.Authorizer)
	}

	// Register the latest (most decorated) handler
	thriftHandler := NewThriftHandler(handler)
	thriftHandler.register(s.GetDispatcher())

	s.adminHandler = NewAdminHandler(s, s.params, s.config)

	adminThriftHandler := NewAdminThriftHandler(s.adminHandler)
	adminThriftHandler.register(s.GetDispatcher())

	// must start resource first
	s.Resource.Start()
	s.handler.Start()
	s.adminHandler.Start()

	// base (service is not started in frontend or admin handler) in case of race condition in yarpc registration function

	logger.Info("frontend started")

	<-s.stopC
}

// Stop stops the service
func (s *Service) Stop() {
	if !atomic.CompareAndSwapInt32(&s.status, common.DaemonStatusStarted, common.DaemonStatusStopped) {
		return
	}

	// initiate graceful shutdown:
	// 1. Fail rpc health check, this will cause client side load balancer to stop forwarding requests to this node
	// 2. wait for failure detection time
	// 3. stop taking new requests by returning InternalServiceError
	// 4. Wait for a second
	// 5. Stop everything forcefully and return

	requestDrainTime := common.MinDuration(time.Second, s.config.ShutdownDrainDuration())
	failureDetectionTime := common.MaxDuration(0, s.config.ShutdownDrainDuration()-requestDrainTime)

	s.GetLogger().Info("ShutdownHandler: Updating rpc health status to ShuttingDown")
	s.handler.UpdateHealthStatus(HealthStatusShuttingDown)

	s.GetLogger().Info("ShutdownHandler: Waiting for others to discover I am unhealthy")
	time.Sleep(failureDetectionTime)

	s.handler.Stop()
	s.adminHandler.Stop()

	s.GetLogger().Info("ShutdownHandler: Draining traffic")
	time.Sleep(requestDrainTime)

	close(s.stopC)
	s.Resource.Stop()
	s.params.Logger.Info("frontend stopped")
}
