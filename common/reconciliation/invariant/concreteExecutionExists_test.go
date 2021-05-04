// The MIT License (MIT)
//
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

package invariant

import (
	"errors"
	"fmt"
	"testing"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/uber/cadence/.gen/go/shared"
	c "github.com/uber/cadence/common"
	"github.com/uber/cadence/common/mocks"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/reconciliation/entity"
)

type ConcreteExecutionExistsSuite struct {
	*require.Assertions
	suite.Suite
}

func TestConcreteExecutionExistsSuite(t *testing.T) {
	suite.Run(t, new(ConcreteExecutionExistsSuite))
}

func (s *ConcreteExecutionExistsSuite) SetupTest() {
	s.Assertions = require.New(s.T())
}

func (s *ConcreteExecutionExistsSuite) TestCheck() {
	existsError := shared.EntityNotExistsError{}
	unknownError := shared.BadRequestError{}
	testCases := []struct {
		execution       *entity.CurrentExecution
		getConcreteResp *persistence.IsWorkflowExecutionExistsResponse
		getConcreteErr  error
		getCurrentResp  *persistence.GetCurrentExecutionResponse
		getCurrentErr   error
		expectedResult  CheckResult
	}{
		{
			execution:       getClosedCurrentExecution(),
			getConcreteResp: &persistence.IsWorkflowExecutionExistsResponse{Exists: true},
			getCurrentResp: &persistence.GetCurrentExecutionResponse{
				RunID: getClosedCurrentExecution().CurrentRunID,
			},
			expectedResult: CheckResult{
				CheckResultType: CheckResultTypeHealthy,
				InvariantName:   ConcreteExecutionExists,
			},
		},
		{
			execution:      getOpenCurrentExecution(),
			getConcreteErr: errors.New("error getting concrete execution"),
			getCurrentResp: &persistence.GetCurrentExecutionResponse{
				RunID: getOpenCurrentExecution().CurrentRunID,
			},
			expectedResult: CheckResult{
				CheckResultType: CheckResultTypeFailed,
				InvariantName:   ConcreteExecutionExists,
				Info:            "failed to check if concrete execution exists",
				InfoDetails:     "error getting concrete execution",
			},
		},
		{
			execution:       getOpenCurrentExecution(),
			getConcreteResp: &persistence.IsWorkflowExecutionExistsResponse{Exists: false},
			getCurrentResp: &persistence.GetCurrentExecutionResponse{
				RunID: getOpenCurrentExecution().CurrentRunID,
			},
			expectedResult: CheckResult{
				CheckResultType: CheckResultTypeCorrupted,
				InvariantName:   ConcreteExecutionExists,
				Info:            "execution is open without having concrete execution",
				InfoDetails: fmt.Sprintf("concrete execution not found. WorkflowId: %v, RunId: %v",
					workflowID, currentRunID),
			},
		},
		{
			execution:       getOpenCurrentExecution(),
			getConcreteErr:  nil,
			getConcreteResp: &persistence.IsWorkflowExecutionExistsResponse{Exists: true},
			getCurrentResp: &persistence.GetCurrentExecutionResponse{
				RunID: getOpenCurrentExecution().CurrentRunID,
			},
			expectedResult: CheckResult{
				CheckResultType: CheckResultTypeHealthy,
				InvariantName:   ConcreteExecutionExists,
			},
		},
		{
			execution:       getOpenCurrentExecution(),
			getConcreteErr:  nil,
			getConcreteResp: &persistence.IsWorkflowExecutionExistsResponse{Exists: true},
			getCurrentResp: &persistence.GetCurrentExecutionResponse{
				RunID: uuid.New(),
			},
			expectedResult: CheckResult{
				CheckResultType: CheckResultTypeHealthy,
				InvariantName:   ConcreteExecutionExists,
			},
		},
		{
			execution:       getOpenCurrentExecution(),
			getConcreteErr:  nil,
			getConcreteResp: &persistence.IsWorkflowExecutionExistsResponse{Exists: true},
			getCurrentResp:  nil,
			getCurrentErr:   &existsError,
			expectedResult: CheckResult{
				CheckResultType: CheckResultTypeHealthy,
				InvariantName:   ConcreteExecutionExists,
			},
		},
		{
			execution:       getOpenCurrentExecution(),
			getConcreteErr:  nil,
			getConcreteResp: &persistence.IsWorkflowExecutionExistsResponse{Exists: false},
			getCurrentResp:  nil,
			getCurrentErr:   &unknownError,
			expectedResult: CheckResult{
				CheckResultType: CheckResultTypeFailed,
				InvariantName:   ConcreteExecutionExists,
				Info:            "failed to get current execution.",
				InfoDetails:     unknownError.Error(),
			},
		},
	}

	for _, tc := range testCases {
		execManager := &mocks.ExecutionManager{}
		execManager.On("IsWorkflowExecutionExists", mock.Anything, mock.Anything).Return(tc.getConcreteResp, tc.getConcreteErr)
		execManager.On("GetCurrentExecution", mock.Anything, mock.Anything).Return(tc.getCurrentResp, tc.getCurrentErr)
		o := NewConcreteExecutionExists(persistence.NewPersistenceRetryer(execManager, nil, c.CreatePersistenceRetryPolicy()))
		s.Equal(tc.expectedResult, o.Check(tc.execution))
	}
}
