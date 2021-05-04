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
	ErrorMessage = "ErrorMessage"
)

var (
	AccessDeniedError = types.AccessDeniedError{
		Message: ErrorMessage,
	}
	BadRequestError = types.BadRequestError{
		Message: ErrorMessage,
	}
	CancellationAlreadyRequestedError = types.CancellationAlreadyRequestedError{
		Message: ErrorMessage,
	}
	ClientVersionNotSupportedError = types.ClientVersionNotSupportedError{
		FeatureVersion:    FeatureVersion,
		ClientImpl:        ClientImpl,
		SupportedVersions: SupportedVersions,
	}
	CurrentBranchChangedError = types.CurrentBranchChangedError{
		Message:            ErrorMessage,
		CurrentBranchToken: BranchToken,
	}
	DomainAlreadyExistsError = types.DomainAlreadyExistsError{
		Message: ErrorMessage,
	}
	DomainNotActiveError = types.DomainNotActiveError{
		Message:        ErrorMessage,
		DomainName:     DomainName,
		CurrentCluster: ClusterName1,
		ActiveCluster:  ClusterName2,
	}
	EntityNotExistsError = types.EntityNotExistsError{
		Message:        ErrorMessage,
		CurrentCluster: ClusterName1,
		ActiveCluster:  ClusterName2,
	}
	WorkflowExecutionAlreadyCompletedError = types.WorkflowExecutionAlreadyCompletedError{
		Message: ErrorMessage,
	}
	EventAlreadyStartedError = types.EventAlreadyStartedError{
		Message: ErrorMessage,
	}
	InternalDataInconsistencyError = types.InternalDataInconsistencyError{
		Message: ErrorMessage,
	}
	InternalServiceError = types.InternalServiceError{
		Message: ErrorMessage,
	}
	LimitExceededError = types.LimitExceededError{
		Message: ErrorMessage,
	}
	QueryFailedError = types.QueryFailedError{
		Message: ErrorMessage,
	}
	RemoteSyncMatchedError = types.RemoteSyncMatchedError{
		Message: ErrorMessage,
	}
	RetryTaskV2Error = types.RetryTaskV2Error{
		Message:           ErrorMessage,
		DomainID:          DomainID,
		WorkflowID:        WorkflowID,
		RunID:             RunID,
		StartEventID:      common.Int64Ptr(EventID1),
		StartEventVersion: common.Int64Ptr(Version1),
		EndEventID:        common.Int64Ptr(EventID2),
		EndEventVersion:   common.Int64Ptr(Version2),
	}
	ServiceBusyError = types.ServiceBusyError{
		Message: ErrorMessage,
	}
	ShardOwnershipLostError = types.ShardOwnershipLostError{
		Message: ErrorMessage,
		Owner:   HostName,
	}
	WorkflowExecutionAlreadyStartedError = types.WorkflowExecutionAlreadyStartedError{
		Message:        ErrorMessage,
		StartRequestID: RequestID,
		RunID:          RunID,
	}
)
