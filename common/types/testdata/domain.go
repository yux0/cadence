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
	"github.com/uber/cadence/common/types"
)

const (
	DomainID          = "DomainID"
	DomainName        = "DomainName"
	DomainDescription = "DomainDescription"
	DomainOwnerEmail  = "DomainOwnerEmail"
	DomainDataKey     = "DomainDataKey"
	DomainDataValue   = "DomainDataValue"
	DomainRetention   = 3
	DomainEmitMetric  = true

	ClusterName1 = "ClusterName1"
	ClusterName2 = "ClusterName2"

	BadBinaryReason   = "BadBinaryReason"
	BadBinaryOperator = "BadBinaryOperator"

	HistoryArchivalURI    = "HistoryArchivalURI"
	VisibilityArchivalURI = "VisibilityArchivalURI"

	DeleteBadBinary = "DeleteBadBinary"

	FailoverVersion1 = 301
	FailoverVersion2 = 302
)

var (
	BadBinaryInfo = types.BadBinaryInfo{
		Reason:          BadBinaryReason,
		Operator:        BadBinaryOperator,
		CreatedTimeNano: &Timestamp1,
	}
	BadBinaryInfoMap = map[string]*types.BadBinaryInfo{
		"BadBinary1": &BadBinaryInfo,
	}
	BadBinaries = types.BadBinaries{
		Binaries: BadBinaryInfoMap,
	}
	DomainData = map[string]string{DomainDataKey: DomainDataValue}
	DomainInfo = types.DomainInfo{
		Name:        DomainName,
		Status:      &DomainStatus,
		Description: DomainDescription,
		OwnerEmail:  DomainOwnerEmail,
		Data:        DomainData,
		UUID:        DomainID,
	}
	DomainConfiguration = types.DomainConfiguration{
		WorkflowExecutionRetentionPeriodInDays: DomainRetention,
		EmitMetric:                             DomainEmitMetric,
		BadBinaries:                            &BadBinaries,
		HistoryArchivalStatus:                  &ArchivalStatus,
		HistoryArchivalURI:                     HistoryArchivalURI,
		VisibilityArchivalStatus:               &ArchivalStatus,
		VisibilityArchivalURI:                  VisibilityArchivalURI,
	}
	DomainReplicationConfiguration = types.DomainReplicationConfiguration{
		ActiveClusterName: ClusterName1,
		Clusters:          ClusterReplicationConfigurationArray,
	}
	ClusterReplicationConfiguration = types.ClusterReplicationConfiguration{
		ClusterName: ClusterName1,
	}
	ClusterReplicationConfigurationArray = []*types.ClusterReplicationConfiguration{
		&ClusterReplicationConfiguration,
	}
)
