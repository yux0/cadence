// Copyright (c) 2016 Uber Technologies, Inc.
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

package host

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pborman/uuid"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/types"
	cadencehistory "github.com/uber/cadence/service/history"
	"github.com/uber/cadence/service/history/execution"
)

func (s *integrationSuite) TestSignalWorkflow() {
	id := "integration-signal-workflow-test"
	wt := "integration-signal-workflow-test-type"
	tl := "integration-signal-workflow-test-tasklist"
	identity := "worker1"
	activityName := "activity_type1"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	// Send a signal to non-exist workflow
	err0 := s.engine.SignalWorkflowExecution(createContext(), &types.SignalWorkflowExecutionRequest{
		Domain: common.StringPtr(s.domainName),
		WorkflowExecution: &types.WorkflowExecution{
			WorkflowID: common.StringPtr(id),
			RunID:      common.StringPtr(uuid.New()),
		},
		SignalName: common.StringPtr("failed signal."),
		Input:      nil,
		Identity:   common.StringPtr(identity),
	})
	s.NotNil(err0)
	s.IsType(&types.EntityNotExistsError{}, err0)

	// Start workflow execution
	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.domainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            common.StringPtr(identity),
	}

	we, err0 := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err0)

	s.Logger.Info("StartWorkflowExecution", tag.WorkflowRunID(*we.RunID))

	// decider logic
	workflowComplete := false
	activityScheduled := false
	activityData := int32(1)
	var signalEvent *types.HistoryEvent
	dtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {

		if !activityScheduled {
			activityScheduled = true
			buf := new(bytes.Buffer)
			s.Nil(binary.Write(buf, binary.LittleEndian, activityData))

			return nil, []*types.Decision{{
				DecisionType: types.DecisionTypeScheduleActivityTask.Ptr(),
				ScheduleActivityTaskDecisionAttributes: &types.ScheduleActivityTaskDecisionAttributes{
					ActivityID:                    common.StringPtr(strconv.Itoa(int(1))),
					ActivityType:                  &types.ActivityType{Name: common.StringPtr(activityName)},
					TaskList:                      &types.TaskList{Name: &tl},
					Input:                         buf.Bytes(),
					ScheduleToCloseTimeoutSeconds: common.Int32Ptr(100),
					ScheduleToStartTimeoutSeconds: common.Int32Ptr(2),
					StartToCloseTimeoutSeconds:    common.Int32Ptr(50),
					HeartbeatTimeoutSeconds:       common.Int32Ptr(5),
				},
			}}, nil
		} else if previousStartedEventID > 0 {
			for _, event := range history.Events[previousStartedEventID:] {
				if *event.EventType == types.EventTypeWorkflowExecutionSignaled {
					signalEvent = event
					return nil, []*types.Decision{}, nil
				}
			}
		}

		workflowComplete = true
		return nil, []*types.Decision{{
			DecisionType: types.DecisionTypeCompleteWorkflowExecution.Ptr(),
			CompleteWorkflowExecutionDecisionAttributes: &types.CompleteWorkflowExecutionDecisionAttributes{
				Result: []byte("Done."),
			},
		}}, nil
	}

	// activity handler
	atHandler := func(execution *types.WorkflowExecution, activityType *types.ActivityType,
		ActivityID string, input []byte, taskToken []byte) ([]byte, bool, error) {

		return []byte("Activity Result."), false, nil
	}

	poller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.domainName,
		TaskList:        taskList,
		Identity:        identity,
		DecisionHandler: dtHandler,
		ActivityHandler: atHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	// Make first decision to schedule activity
	_, err := poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	// Send first signal using RunID
	signalName := "my signal"
	signalInput := []byte("my signal input.")
	err = s.engine.SignalWorkflowExecution(createContext(), &types.SignalWorkflowExecutionRequest{
		Domain: common.StringPtr(s.domainName),
		WorkflowExecution: &types.WorkflowExecution{
			WorkflowID: common.StringPtr(id),
			RunID:      common.StringPtr(*we.RunID),
		},
		SignalName: common.StringPtr(signalName),
		Input:      signalInput,
		Identity:   common.StringPtr(identity),
	})
	s.Nil(err)

	// Process signal in decider
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	s.False(workflowComplete)
	s.True(signalEvent != nil)
	s.Equal(signalName, *signalEvent.WorkflowExecutionSignaledEventAttributes.SignalName)
	s.Equal(signalInput, signalEvent.WorkflowExecutionSignaledEventAttributes.Input)
	s.Equal(identity, *signalEvent.WorkflowExecutionSignaledEventAttributes.Identity)

	// Send another signal without RunID
	signalName = "another signal"
	signalInput = []byte("another signal input.")
	err = s.engine.SignalWorkflowExecution(createContext(), &types.SignalWorkflowExecutionRequest{
		Domain: common.StringPtr(s.domainName),
		WorkflowExecution: &types.WorkflowExecution{
			WorkflowID: common.StringPtr(id),
		},
		SignalName: common.StringPtr(signalName),
		Input:      signalInput,
		Identity:   common.StringPtr(identity),
	})
	s.Nil(err)

	// Process signal in decider
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	s.False(workflowComplete)
	s.True(signalEvent != nil)
	s.Equal(signalName, *signalEvent.WorkflowExecutionSignaledEventAttributes.SignalName)
	s.Equal(signalInput, signalEvent.WorkflowExecutionSignaledEventAttributes.Input)
	s.Equal(identity, *signalEvent.WorkflowExecutionSignaledEventAttributes.Identity)

	// Terminate workflow execution
	err = s.engine.TerminateWorkflowExecution(createContext(), &types.TerminateWorkflowExecutionRequest{
		Domain: common.StringPtr(s.domainName),
		WorkflowExecution: &types.WorkflowExecution{
			WorkflowID: common.StringPtr(id),
		},
		Reason:   common.StringPtr("test signal"),
		Details:  nil,
		Identity: common.StringPtr(identity),
	})
	s.Nil(err)

	// Send signal to terminated workflow
	err = s.engine.SignalWorkflowExecution(createContext(), &types.SignalWorkflowExecutionRequest{
		Domain: common.StringPtr(s.domainName),
		WorkflowExecution: &types.WorkflowExecution{
			WorkflowID: common.StringPtr(id),
			RunID:      common.StringPtr(*we.RunID),
		},
		SignalName: common.StringPtr("failed signal 1."),
		Input:      nil,
		Identity:   common.StringPtr(identity),
	})
	s.NotNil(err)
	s.IsType(&types.EntityNotExistsError{}, err)
}

func (s *integrationSuite) TestSignalWorkflow_DuplicateRequest() {
	id := "integration-signal-workflow-test-duplicate"
	wt := "integration-signal-workflow-test-duplicate-type"
	tl := "integration-signal-workflow-test-duplicate-tasklist"
	identity := "worker1"
	activityName := "activity_type1"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	// Start workflow execution
	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.domainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            common.StringPtr(identity),
	}

	we, err0 := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err0)
	s.Logger.Info("StartWorkflowExecution", tag.WorkflowRunID(*we.RunID))

	// decider logic
	workflowComplete := false
	activityScheduled := false
	activityData := int32(1)
	var signalEvent *types.HistoryEvent
	numOfSignaledEvent := 0
	dtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {

		if !activityScheduled {
			activityScheduled = true
			buf := new(bytes.Buffer)
			s.Nil(binary.Write(buf, binary.LittleEndian, activityData))

			return nil, []*types.Decision{{
				DecisionType: types.DecisionTypeScheduleActivityTask.Ptr(),
				ScheduleActivityTaskDecisionAttributes: &types.ScheduleActivityTaskDecisionAttributes{
					ActivityID:                    common.StringPtr(strconv.Itoa(int(1))),
					ActivityType:                  &types.ActivityType{Name: common.StringPtr(activityName)},
					TaskList:                      &types.TaskList{Name: &tl},
					Input:                         buf.Bytes(),
					ScheduleToCloseTimeoutSeconds: common.Int32Ptr(100),
					ScheduleToStartTimeoutSeconds: common.Int32Ptr(2),
					StartToCloseTimeoutSeconds:    common.Int32Ptr(50),
					HeartbeatTimeoutSeconds:       common.Int32Ptr(5),
				},
			}}, nil
		} else if previousStartedEventID > 0 {
			numOfSignaledEvent = 0
			for _, event := range history.Events[previousStartedEventID:] {
				if *event.EventType == types.EventTypeWorkflowExecutionSignaled {
					signalEvent = event
					numOfSignaledEvent++
				}
			}
			return nil, []*types.Decision{}, nil
		}

		workflowComplete = true
		return nil, []*types.Decision{{
			DecisionType: types.DecisionTypeCompleteWorkflowExecution.Ptr(),
			CompleteWorkflowExecutionDecisionAttributes: &types.CompleteWorkflowExecutionDecisionAttributes{
				Result: []byte("Done."),
			},
		}}, nil
	}

	// activity handler
	atHandler := func(execution *types.WorkflowExecution, activityType *types.ActivityType,
		ActivityID string, input []byte, taskToken []byte) ([]byte, bool, error) {

		return []byte("Activity Result."), false, nil
	}

	poller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.domainName,
		TaskList:        taskList,
		Identity:        identity,
		DecisionHandler: dtHandler,
		ActivityHandler: atHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	// Make first decision to schedule activity
	_, err := poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	// Send first signal
	signalName := "my signal"
	signalInput := []byte("my signal input.")
	RequestID := uuid.New()
	signalReqest := &types.SignalWorkflowExecutionRequest{
		Domain: common.StringPtr(s.domainName),
		WorkflowExecution: &types.WorkflowExecution{
			WorkflowID: common.StringPtr(id),
			RunID:      common.StringPtr(*we.RunID),
		},
		SignalName: common.StringPtr(signalName),
		Input:      signalInput,
		Identity:   common.StringPtr(identity),
		RequestID:  common.StringPtr(RequestID),
	}
	err = s.engine.SignalWorkflowExecution(createContext(), signalReqest)
	s.Nil(err)

	// Process signal in decider
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	s.False(workflowComplete)
	s.True(signalEvent != nil)
	s.Equal(signalName, *signalEvent.WorkflowExecutionSignaledEventAttributes.SignalName)
	s.Equal(signalInput, signalEvent.WorkflowExecutionSignaledEventAttributes.Input)
	s.Equal(identity, *signalEvent.WorkflowExecutionSignaledEventAttributes.Identity)
	s.Equal(1, numOfSignaledEvent)

	// Send another signal with same request id
	err = s.engine.SignalWorkflowExecution(createContext(), signalReqest)
	s.Nil(err)

	// Process signal in decider
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	s.False(workflowComplete)
	s.True(signalEvent != nil)
	s.Equal(0, numOfSignaledEvent)
}

func (s *integrationSuite) TestSignalExternalWorkflowDecision() {
	id := "integration-signal-external-workflow-test"
	wt := "integration-signal-external-workflow-test-type"
	tl := "integration-signal-external-workflow-test-tasklist"
	identity := "worker1"
	activityName := "activity_type1"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.domainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            common.StringPtr(identity),
	}

	we, err0 := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err0)
	s.Logger.Info("StartWorkflowExecution", tag.WorkflowRunID(*we.RunID))

	foreignRequest := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.foreignDomainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            common.StringPtr(identity),
	}
	we2, err0 := s.engine.StartWorkflowExecution(createContext(), foreignRequest)
	s.Nil(err0)
	s.Logger.Info("StartWorkflowExecution on foreign Domain", tag.WorkflowDomainName(s.foreignDomainName), tag.WorkflowRunID(*we2.RunID))

	activityCount := int32(1)
	activityCounter := int32(0)
	signalName := "my signal"
	signalInput := []byte("my signal input.")
	dtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {
		if activityCounter < activityCount {
			activityCounter++
			buf := new(bytes.Buffer)
			s.Nil(binary.Write(buf, binary.LittleEndian, activityCounter))

			return []byte(strconv.Itoa(int(activityCounter))), []*types.Decision{{
				DecisionType: types.DecisionTypeScheduleActivityTask.Ptr(),
				ScheduleActivityTaskDecisionAttributes: &types.ScheduleActivityTaskDecisionAttributes{
					ActivityID:                    common.StringPtr(strconv.Itoa(int(activityCounter))),
					ActivityType:                  &types.ActivityType{Name: common.StringPtr(activityName)},
					TaskList:                      &types.TaskList{Name: &tl},
					Input:                         buf.Bytes(),
					ScheduleToCloseTimeoutSeconds: common.Int32Ptr(100),
					ScheduleToStartTimeoutSeconds: common.Int32Ptr(10),
					StartToCloseTimeoutSeconds:    common.Int32Ptr(50),
					HeartbeatTimeoutSeconds:       common.Int32Ptr(5),
				},
			}}, nil
		}

		return []byte(strconv.Itoa(int(activityCounter))), []*types.Decision{{
			DecisionType: types.DecisionTypeSignalExternalWorkflowExecution.Ptr(),
			SignalExternalWorkflowExecutionDecisionAttributes: &types.SignalExternalWorkflowExecutionDecisionAttributes{
				Domain: common.StringPtr(s.foreignDomainName),
				Execution: &types.WorkflowExecution{
					WorkflowID: common.StringPtr(id),
					RunID:      common.StringPtr(we2.GetRunID()),
				},
				SignalName: common.StringPtr(signalName),
				Input:      signalInput,
			},
		}}, nil
	}

	atHandler := func(execution *types.WorkflowExecution, activityType *types.ActivityType,
		ActivityID string, input []byte, taskToken []byte) ([]byte, bool, error) {
		return []byte("Activity Result."), false, nil
	}

	poller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.domainName,
		TaskList:        taskList,
		Identity:        identity,
		DecisionHandler: dtHandler,
		ActivityHandler: atHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	workflowComplete := false
	foreignActivityCount := int32(1)
	foreignActivityCounter := int32(0)
	var signalEvent *types.HistoryEvent
	foreignDtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {
		if foreignActivityCounter < foreignActivityCount {
			foreignActivityCounter++
			buf := new(bytes.Buffer)
			s.Nil(binary.Write(buf, binary.LittleEndian, foreignActivityCounter))

			return []byte(strconv.Itoa(int(foreignActivityCounter))), []*types.Decision{{
				DecisionType: types.DecisionTypeScheduleActivityTask.Ptr(),
				ScheduleActivityTaskDecisionAttributes: &types.ScheduleActivityTaskDecisionAttributes{
					ActivityID:                    common.StringPtr(strconv.Itoa(int(foreignActivityCounter))),
					ActivityType:                  &types.ActivityType{Name: common.StringPtr(activityName)},
					TaskList:                      &types.TaskList{Name: &tl},
					Input:                         buf.Bytes(),
					ScheduleToCloseTimeoutSeconds: common.Int32Ptr(100),
					ScheduleToStartTimeoutSeconds: common.Int32Ptr(10),
					StartToCloseTimeoutSeconds:    common.Int32Ptr(50),
					HeartbeatTimeoutSeconds:       common.Int32Ptr(5),
				},
			}}, nil
		} else if previousStartedEventID > 0 {
			for _, event := range history.Events[previousStartedEventID:] {
				if *event.EventType == types.EventTypeWorkflowExecutionSignaled {
					signalEvent = event
					return nil, []*types.Decision{}, nil
				}
			}
		}

		workflowComplete = true
		return nil, []*types.Decision{{
			DecisionType: types.DecisionTypeCompleteWorkflowExecution.Ptr(),
			CompleteWorkflowExecutionDecisionAttributes: &types.CompleteWorkflowExecutionDecisionAttributes{
				Result: []byte("Done."),
			},
		}}, nil
	}

	foreignPoller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.foreignDomainName,
		TaskList:        taskList,
		Identity:        identity,
		DecisionHandler: foreignDtHandler,
		ActivityHandler: atHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	// Start both current and foreign workflows to make some progress.
	_, err := poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	_, err = foreignPoller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("foreign PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	err = foreignPoller.PollAndProcessActivityTask(false)
	s.Logger.Info("foreign PollAndProcessActivityTask", tag.Error(err))
	s.Nil(err)

	// Signal the foreign workflow with this decision request.
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	// in source workflow
	signalSent := false
	intiatedEventID := 10
CheckHistoryLoopForSignalSent:
	for i := 1; i < 10; i++ {
		historyResponse, err := s.engine.GetWorkflowExecutionHistory(createContext(), &types.GetWorkflowExecutionHistoryRequest{
			Domain: common.StringPtr(s.domainName),
			Execution: &types.WorkflowExecution{
				WorkflowID: common.StringPtr(id),
				RunID:      common.StringPtr(*we.RunID),
			},
		})
		s.Nil(err)
		history := historyResponse.History
		//common.PrettyPrintHistory(history, s.Logger)

		signalRequestedEvent := history.Events[len(history.Events)-2]
		if *signalRequestedEvent.EventType != types.EventTypeExternalWorkflowExecutionSignaled {
			s.Logger.Info("Signal still not sent.")
			time.Sleep(100 * time.Millisecond)
			continue CheckHistoryLoopForSignalSent
		}

		ewfeAttributes := signalRequestedEvent.ExternalWorkflowExecutionSignaledEventAttributes
		s.NotNil(ewfeAttributes)
		s.Equal(int64(intiatedEventID), ewfeAttributes.GetInitiatedEventID())
		s.Equal(id, ewfeAttributes.WorkflowExecution.GetWorkflowID())
		s.Equal(we2.RunID, ewfeAttributes.WorkflowExecution.RunID)

		signalSent = true
		break
	}

	s.True(signalSent)

	// process signal in decider for foreign workflow
	_, err = foreignPoller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	s.False(workflowComplete)
	s.True(signalEvent != nil)
	s.Equal(signalName, *signalEvent.WorkflowExecutionSignaledEventAttributes.SignalName)
	s.Equal(signalInput, signalEvent.WorkflowExecutionSignaledEventAttributes.Input)
	s.Equal("history-service", *signalEvent.WorkflowExecutionSignaledEventAttributes.Identity)
}

func (s *integrationSuite) TestSignalWorkflow_Cron_NoDecisionTaskCreated() {
	id := "integration-signal-workflow-test-cron"
	wt := "integration-signal-workflow-test-cron-type"
	tl := "integration-signal-workflow-test-cron-tasklist"
	identity := "worker1"
	cronSpec := "@every 2s"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	// Start workflow execution
	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.domainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            common.StringPtr(identity),
		CronSchedule:                        &cronSpec,
	}
	now := time.Now()

	we, err0 := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err0)

	s.Logger.Info("StartWorkflowExecution", tag.WorkflowRunID(*we.RunID))

	// Send first signal using RunID
	signalName := "my signal"
	signalInput := []byte("my signal input.")
	err := s.engine.SignalWorkflowExecution(createContext(), &types.SignalWorkflowExecutionRequest{
		Domain: common.StringPtr(s.domainName),
		WorkflowExecution: &types.WorkflowExecution{
			WorkflowID: common.StringPtr(id),
			RunID:      common.StringPtr(*we.RunID),
		},
		SignalName: common.StringPtr(signalName),
		Input:      signalInput,
		Identity:   common.StringPtr(identity),
	})
	s.Nil(err)

	// decider logic
	var decisionTaskDelay time.Duration
	dtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {
		decisionTaskDelay = time.Now().Sub(now)

		return nil, []*types.Decision{{
			DecisionType: types.DecisionTypeCompleteWorkflowExecution.Ptr(),
			CompleteWorkflowExecutionDecisionAttributes: &types.CompleteWorkflowExecutionDecisionAttributes{
				Result: []byte("Done."),
			},
		}}, nil
	}

	poller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.domainName,
		TaskList:        taskList,
		Identity:        identity,
		DecisionHandler: dtHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	// Make first decision to schedule activity
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)
	s.True(decisionTaskDelay > time.Second*2)
}

func (s *integrationSuite) TestSignalExternalWorkflowDecision_WithoutRunID() {
	id := "integration-signal-external-workflow-test-without-run-id"
	wt := "integration-signal-external-workflow-test-without-run-id-type"
	tl := "integration-signal-external-workflow-test-without-run-id-tasklist"
	identity := "worker1"
	activityName := "activity_type1"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.domainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            common.StringPtr(identity),
	}

	we, err0 := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err0)
	s.Logger.Info("StartWorkflowExecution", tag.WorkflowRunID(*we.RunID))

	foreignRequest := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.foreignDomainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            common.StringPtr(identity),
	}
	we2, err0 := s.engine.StartWorkflowExecution(createContext(), foreignRequest)
	s.Nil(err0)
	s.Logger.Info("StartWorkflowExecution on foreign Domain", tag.WorkflowDomainName(s.foreignDomainName), tag.WorkflowRunID(*we2.RunID))

	activityCount := int32(1)
	activityCounter := int32(0)
	signalName := "my signal"
	signalInput := []byte("my signal input.")
	dtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {
		if activityCounter < activityCount {
			activityCounter++
			buf := new(bytes.Buffer)
			s.Nil(binary.Write(buf, binary.LittleEndian, activityCounter))

			return []byte(strconv.Itoa(int(activityCounter))), []*types.Decision{{
				DecisionType: types.DecisionTypeScheduleActivityTask.Ptr(),
				ScheduleActivityTaskDecisionAttributes: &types.ScheduleActivityTaskDecisionAttributes{
					ActivityID:                    common.StringPtr(strconv.Itoa(int(activityCounter))),
					ActivityType:                  &types.ActivityType{Name: common.StringPtr(activityName)},
					TaskList:                      &types.TaskList{Name: &tl},
					Input:                         buf.Bytes(),
					ScheduleToCloseTimeoutSeconds: common.Int32Ptr(100),
					ScheduleToStartTimeoutSeconds: common.Int32Ptr(10),
					StartToCloseTimeoutSeconds:    common.Int32Ptr(50),
					HeartbeatTimeoutSeconds:       common.Int32Ptr(5),
				},
			}}, nil
		}

		return []byte(strconv.Itoa(int(activityCounter))), []*types.Decision{{
			DecisionType: types.DecisionTypeSignalExternalWorkflowExecution.Ptr(),
			SignalExternalWorkflowExecutionDecisionAttributes: &types.SignalExternalWorkflowExecutionDecisionAttributes{
				Domain: common.StringPtr(s.foreignDomainName),
				Execution: &types.WorkflowExecution{
					WorkflowID: common.StringPtr(id),
					// No RunID in decision
				},
				SignalName: common.StringPtr(signalName),
				Input:      signalInput,
			},
		}}, nil
	}

	atHandler := func(execution *types.WorkflowExecution, activityType *types.ActivityType,
		ActivityID string, input []byte, taskToken []byte) ([]byte, bool, error) {
		return []byte("Activity Result."), false, nil
	}

	poller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.domainName,
		TaskList:        taskList,
		Identity:        identity,
		DecisionHandler: dtHandler,
		ActivityHandler: atHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	workflowComplete := false
	foreignActivityCount := int32(1)
	foreignActivityCounter := int32(0)
	var signalEvent *types.HistoryEvent
	foreignDtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {
		if foreignActivityCounter < foreignActivityCount {
			foreignActivityCounter++
			buf := new(bytes.Buffer)
			s.Nil(binary.Write(buf, binary.LittleEndian, foreignActivityCounter))

			return []byte(strconv.Itoa(int(foreignActivityCounter))), []*types.Decision{{
				DecisionType: types.DecisionTypeScheduleActivityTask.Ptr(),
				ScheduleActivityTaskDecisionAttributes: &types.ScheduleActivityTaskDecisionAttributes{
					ActivityID:                    common.StringPtr(strconv.Itoa(int(foreignActivityCounter))),
					ActivityType:                  &types.ActivityType{Name: common.StringPtr(activityName)},
					TaskList:                      &types.TaskList{Name: &tl},
					Input:                         buf.Bytes(),
					ScheduleToCloseTimeoutSeconds: common.Int32Ptr(100),
					ScheduleToStartTimeoutSeconds: common.Int32Ptr(10),
					StartToCloseTimeoutSeconds:    common.Int32Ptr(50),
					HeartbeatTimeoutSeconds:       common.Int32Ptr(5),
				},
			}}, nil
		} else if previousStartedEventID > 0 {
			for _, event := range history.Events[previousStartedEventID:] {
				if *event.EventType == types.EventTypeWorkflowExecutionSignaled {
					signalEvent = event
					return nil, []*types.Decision{}, nil
				}
			}
		}

		workflowComplete = true
		return nil, []*types.Decision{{
			DecisionType: types.DecisionTypeCompleteWorkflowExecution.Ptr(),
			CompleteWorkflowExecutionDecisionAttributes: &types.CompleteWorkflowExecutionDecisionAttributes{
				Result: []byte("Done."),
			},
		}}, nil
	}

	foreignPoller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.foreignDomainName,
		TaskList:        taskList,
		Identity:        identity,
		DecisionHandler: foreignDtHandler,
		ActivityHandler: atHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	// Start both current and foreign workflows to make some progress.
	_, err := poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	_, err = foreignPoller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("foreign PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	err = foreignPoller.PollAndProcessActivityTask(false)
	s.Logger.Info("foreign PollAndProcessActivityTask", tag.Error(err))
	s.Nil(err)

	// Signal the foreign workflow with this decision request.
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	// in source workflow
	signalSent := false
	intiatedEventID := 10
CheckHistoryLoopForSignalSent:
	for i := 1; i < 10; i++ {
		historyResponse, err := s.engine.GetWorkflowExecutionHistory(createContext(), &types.GetWorkflowExecutionHistoryRequest{
			Domain: common.StringPtr(s.domainName),
			Execution: &types.WorkflowExecution{
				WorkflowID: common.StringPtr(id),
				RunID:      common.StringPtr(*we.RunID),
			},
		})
		s.Nil(err)
		history := historyResponse.History

		signalRequestedEvent := history.Events[len(history.Events)-2]
		if *signalRequestedEvent.EventType != types.EventTypeExternalWorkflowExecutionSignaled {
			s.Logger.Info("Signal still not sent.")
			time.Sleep(100 * time.Millisecond)
			continue CheckHistoryLoopForSignalSent
		}

		ewfeAttributes := signalRequestedEvent.ExternalWorkflowExecutionSignaledEventAttributes
		s.NotNil(ewfeAttributes)
		s.Equal(int64(intiatedEventID), ewfeAttributes.GetInitiatedEventID())
		s.Equal(id, ewfeAttributes.WorkflowExecution.GetWorkflowID())
		s.Equal("", ewfeAttributes.WorkflowExecution.GetRunID())

		signalSent = true
		break
	}

	s.True(signalSent)

	// process signal in decider for foreign workflow
	_, err = foreignPoller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	s.False(workflowComplete)
	s.True(signalEvent != nil)
	s.Equal(signalName, *signalEvent.WorkflowExecutionSignaledEventAttributes.SignalName)
	s.Equal(signalInput, signalEvent.WorkflowExecutionSignaledEventAttributes.Input)
	s.Equal("history-service", *signalEvent.WorkflowExecutionSignaledEventAttributes.Identity)
}

func (s *integrationSuite) TestSignalExternalWorkflowDecision_UnKnownTarget() {
	id := "integration-signal-unknown-workflow-decision-test"
	wt := "integration-signal-unknown-workflow-decision-test-type"
	tl := "integration-signal-unknown-workflow-decision-test-tasklist"
	identity := "worker1"
	activityName := "activity_type1"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.domainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            common.StringPtr(identity),
	}
	we, err0 := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err0)
	s.Logger.Info("StartWorkflowExecution", tag.WorkflowRunID(*we.RunID))

	activityCount := int32(1)
	activityCounter := int32(0)
	signalName := "my signal"
	signalInput := []byte("my signal input.")
	dtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {
		if activityCounter < activityCount {
			activityCounter++
			buf := new(bytes.Buffer)
			s.Nil(binary.Write(buf, binary.LittleEndian, activityCounter))

			return []byte(strconv.Itoa(int(activityCounter))), []*types.Decision{{
				DecisionType: types.DecisionTypeScheduleActivityTask.Ptr(),
				ScheduleActivityTaskDecisionAttributes: &types.ScheduleActivityTaskDecisionAttributes{
					ActivityID:                    common.StringPtr(strconv.Itoa(int(activityCounter))),
					ActivityType:                  &types.ActivityType{Name: common.StringPtr(activityName)},
					TaskList:                      &types.TaskList{Name: &tl},
					Input:                         buf.Bytes(),
					ScheduleToCloseTimeoutSeconds: common.Int32Ptr(100),
					ScheduleToStartTimeoutSeconds: common.Int32Ptr(10),
					StartToCloseTimeoutSeconds:    common.Int32Ptr(50),
					HeartbeatTimeoutSeconds:       common.Int32Ptr(5),
				},
			}}, nil
		}

		return []byte(strconv.Itoa(int(activityCounter))), []*types.Decision{{
			DecisionType: types.DecisionTypeSignalExternalWorkflowExecution.Ptr(),
			SignalExternalWorkflowExecutionDecisionAttributes: &types.SignalExternalWorkflowExecutionDecisionAttributes{
				Domain: common.StringPtr(s.foreignDomainName),
				Execution: &types.WorkflowExecution{
					WorkflowID: common.StringPtr("workflow_not_exist"),
					RunID:      common.StringPtr(we.GetRunID()),
				},
				SignalName: common.StringPtr(signalName),
				Input:      signalInput,
			},
		}}, nil
	}

	atHandler := func(execution *types.WorkflowExecution, activityType *types.ActivityType,
		ActivityID string, input []byte, taskToken []byte) ([]byte, bool, error) {
		return []byte("Activity Result."), false, nil
	}

	poller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.domainName,
		TaskList:        taskList,
		Identity:        identity,
		DecisionHandler: dtHandler,
		ActivityHandler: atHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	// Start workflows to make some progress.
	_, err := poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	// Signal the foreign workflow with this decision request.
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	signalSentFailed := false
	intiatedEventID := 10
CheckHistoryLoopForCancelSent:
	for i := 1; i < 10; i++ {
		historyResponse, err := s.engine.GetWorkflowExecutionHistory(createContext(), &types.GetWorkflowExecutionHistoryRequest{
			Domain: common.StringPtr(s.domainName),
			Execution: &types.WorkflowExecution{
				WorkflowID: common.StringPtr(id),
				RunID:      common.StringPtr(*we.RunID),
			},
		})
		s.Nil(err)
		history := historyResponse.History

		signalFailedEvent := history.Events[len(history.Events)-2]
		if *signalFailedEvent.EventType != types.EventTypeSignalExternalWorkflowExecutionFailed {
			s.Logger.Info("Cancellaton not cancelled yet.")
			time.Sleep(100 * time.Millisecond)
			continue CheckHistoryLoopForCancelSent
		}

		signalExternalWorkflowExecutionFailedEventAttributes := signalFailedEvent.SignalExternalWorkflowExecutionFailedEventAttributes
		s.Equal(int64(intiatedEventID), *signalExternalWorkflowExecutionFailedEventAttributes.InitiatedEventID)
		s.Equal("workflow_not_exist", *signalExternalWorkflowExecutionFailedEventAttributes.WorkflowExecution.WorkflowID)
		s.Equal(we.RunID, signalExternalWorkflowExecutionFailedEventAttributes.WorkflowExecution.RunID)

		signalSentFailed = true
		break
	}

	s.True(signalSentFailed)
}

func (s *integrationSuite) TestSignalExternalWorkflowDecision_SignalSelf() {
	id := "integration-signal-self-workflow-decision-test"
	wt := "integration-signal-self-workflow-decision-test-type"
	tl := "integration-signal-self-workflow-decision-test-tasklist"
	identity := "worker1"
	activityName := "activity_type1"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.domainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            common.StringPtr(identity),
	}
	we, err0 := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err0)
	s.Logger.Info("StartWorkflowExecution", tag.WorkflowRunID(*we.RunID))

	activityCount := int32(1)
	activityCounter := int32(0)
	signalName := "my signal"
	signalInput := []byte("my signal input.")
	dtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {
		if activityCounter < activityCount {
			activityCounter++
			buf := new(bytes.Buffer)
			s.Nil(binary.Write(buf, binary.LittleEndian, activityCounter))

			return []byte(strconv.Itoa(int(activityCounter))), []*types.Decision{{
				DecisionType: types.DecisionTypeScheduleActivityTask.Ptr(),
				ScheduleActivityTaskDecisionAttributes: &types.ScheduleActivityTaskDecisionAttributes{
					ActivityID:                    common.StringPtr(strconv.Itoa(int(activityCounter))),
					ActivityType:                  &types.ActivityType{Name: common.StringPtr(activityName)},
					TaskList:                      &types.TaskList{Name: &tl},
					Input:                         buf.Bytes(),
					ScheduleToCloseTimeoutSeconds: common.Int32Ptr(100),
					ScheduleToStartTimeoutSeconds: common.Int32Ptr(10),
					StartToCloseTimeoutSeconds:    common.Int32Ptr(50),
					HeartbeatTimeoutSeconds:       common.Int32Ptr(5),
				},
			}}, nil
		}

		return []byte(strconv.Itoa(int(activityCounter))), []*types.Decision{{
			DecisionType: types.DecisionTypeSignalExternalWorkflowExecution.Ptr(),
			SignalExternalWorkflowExecutionDecisionAttributes: &types.SignalExternalWorkflowExecutionDecisionAttributes{
				Domain: common.StringPtr(s.domainName),
				Execution: &types.WorkflowExecution{
					WorkflowID: common.StringPtr(id),
					RunID:      common.StringPtr(we.GetRunID()),
				},
				SignalName: common.StringPtr(signalName),
				Input:      signalInput,
			},
		}}, nil
	}

	atHandler := func(execution *types.WorkflowExecution, activityType *types.ActivityType,
		ActivityID string, input []byte, taskToken []byte) ([]byte, bool, error) {
		return []byte("Activity Result."), false, nil
	}

	poller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.domainName,
		TaskList:        taskList,
		Identity:        identity,
		DecisionHandler: dtHandler,
		ActivityHandler: atHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	// Start workflows to make some progress.
	_, err := poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	// Signal the foreign workflow with this decision request.
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	signalSentFailed := false
	intiatedEventID := 10
CheckHistoryLoopForCancelSent:
	for i := 1; i < 10; i++ {
		historyResponse, err := s.engine.GetWorkflowExecutionHistory(createContext(), &types.GetWorkflowExecutionHistoryRequest{
			Domain: common.StringPtr(s.domainName),
			Execution: &types.WorkflowExecution{
				WorkflowID: common.StringPtr(id),
				RunID:      common.StringPtr(*we.RunID),
			},
		})
		s.Nil(err)
		history := historyResponse.History

		signalFailedEvent := history.Events[len(history.Events)-2]
		if *signalFailedEvent.EventType != types.EventTypeSignalExternalWorkflowExecutionFailed {
			s.Logger.Info("Cancellaton not cancelled yet.")
			time.Sleep(100 * time.Millisecond)
			continue CheckHistoryLoopForCancelSent
		}

		signalExternalWorkflowExecutionFailedEventAttributes := signalFailedEvent.SignalExternalWorkflowExecutionFailedEventAttributes
		s.Equal(int64(intiatedEventID), *signalExternalWorkflowExecutionFailedEventAttributes.InitiatedEventID)
		s.Equal(id, *signalExternalWorkflowExecutionFailedEventAttributes.WorkflowExecution.WorkflowID)
		s.Equal(we.RunID, signalExternalWorkflowExecutionFailedEventAttributes.WorkflowExecution.RunID)

		signalSentFailed = true
		break
	}

	s.True(signalSentFailed)

}

func (s *integrationSuite) TestSignalWithStartWorkflow() {
	id := "integration-signal-with-start-workflow-test"
	wt := "integration-signal-with-start-workflow-test-type"
	tl := "integration-signal-with-start-workflow-test-tasklist"
	identity := "worker1"
	activityName := "activity_type1"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	header := &types.Header{
		Fields: map[string][]byte{"tracing": []byte("sample data")},
	}

	// Start a workflow
	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.domainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            common.StringPtr(identity),
	}

	we, err0 := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err0)

	s.Logger.Info("StartWorkflowExecution", tag.WorkflowRunID(*we.RunID))

	// decider logic
	workflowComplete := false
	activityScheduled := false
	activityData := int32(1)
	newWorkflowStarted := false
	var signalEvent, startedEvent *types.HistoryEvent
	dtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {

		if !activityScheduled {
			activityScheduled = true
			buf := new(bytes.Buffer)
			s.Nil(binary.Write(buf, binary.LittleEndian, activityData))

			return nil, []*types.Decision{{
				DecisionType: types.DecisionTypeScheduleActivityTask.Ptr(),
				ScheduleActivityTaskDecisionAttributes: &types.ScheduleActivityTaskDecisionAttributes{
					ActivityID:                    common.StringPtr(strconv.Itoa(int(1))),
					ActivityType:                  &types.ActivityType{Name: common.StringPtr(activityName)},
					TaskList:                      &types.TaskList{Name: &tl},
					Input:                         buf.Bytes(),
					ScheduleToCloseTimeoutSeconds: common.Int32Ptr(100),
					ScheduleToStartTimeoutSeconds: common.Int32Ptr(2),
					StartToCloseTimeoutSeconds:    common.Int32Ptr(50),
					HeartbeatTimeoutSeconds:       common.Int32Ptr(5),
				},
			}}, nil
		} else if previousStartedEventID > 0 {
			for _, event := range history.Events[previousStartedEventID:] {
				if *event.EventType == types.EventTypeWorkflowExecutionSignaled {
					signalEvent = event
					return nil, []*types.Decision{}, nil
				}
			}
		} else if newWorkflowStarted {
			newWorkflowStarted = false
			signalEvent = nil
			startedEvent = nil
			for _, event := range history.Events[previousStartedEventID:] {
				if *event.EventType == types.EventTypeWorkflowExecutionSignaled {
					signalEvent = event
				}
				if *event.EventType == types.EventTypeWorkflowExecutionStarted {
					startedEvent = event
				}
			}
			if signalEvent != nil && startedEvent != nil {
				return nil, []*types.Decision{}, nil
			}
		}

		workflowComplete = true
		return nil, []*types.Decision{{
			DecisionType: types.DecisionTypeCompleteWorkflowExecution.Ptr(),
			CompleteWorkflowExecutionDecisionAttributes: &types.CompleteWorkflowExecutionDecisionAttributes{
				Result: []byte("Done."),
			},
		}}, nil
	}

	// activity handler
	atHandler := func(execution *types.WorkflowExecution, activityType *types.ActivityType,
		ActivityID string, input []byte, taskToken []byte) ([]byte, bool, error) {

		return []byte("Activity Result."), false, nil
	}

	poller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.domainName,
		TaskList:        taskList,
		Identity:        identity,
		DecisionHandler: dtHandler,
		ActivityHandler: atHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	// Make first decision to schedule activity
	_, err := poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	// Send a signal
	signalName := "my signal"
	signalInput := []byte("my signal input.")
	wfIDReusePolicy := types.WorkflowIDReusePolicyAllowDuplicate
	sRequest := &types.SignalWithStartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.domainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		Header:                              header,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		SignalName:                          common.StringPtr(signalName),
		SignalInput:                         signalInput,
		Identity:                            common.StringPtr(identity),
		WorkflowIDReusePolicy:               &wfIDReusePolicy,
	}
	resp, err := s.engine.SignalWithStartWorkflowExecution(createContext(), sRequest)
	s.Nil(err)
	s.Equal(we.GetRunID(), resp.GetRunID())

	// Process signal in decider
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	s.False(workflowComplete)
	s.True(signalEvent != nil)
	s.Equal(signalName, *signalEvent.WorkflowExecutionSignaledEventAttributes.SignalName)
	s.Equal(signalInput, signalEvent.WorkflowExecutionSignaledEventAttributes.Input)
	s.Equal(identity, *signalEvent.WorkflowExecutionSignaledEventAttributes.Identity)

	// Terminate workflow execution
	err = s.engine.TerminateWorkflowExecution(createContext(), &types.TerminateWorkflowExecutionRequest{
		Domain: common.StringPtr(s.domainName),
		WorkflowExecution: &types.WorkflowExecution{
			WorkflowID: common.StringPtr(id),
		},
		Reason:   common.StringPtr("test signal"),
		Details:  nil,
		Identity: common.StringPtr(identity),
	})
	s.Nil(err)

	// Send signal to terminated workflow
	signalName = "signal to terminate"
	signalInput = []byte("signal to terminate input.")
	sRequest.SignalName = common.StringPtr(signalName)
	sRequest.SignalInput = signalInput
	sRequest.WorkflowID = common.StringPtr(id)

	resp, err = s.engine.SignalWithStartWorkflowExecution(createContext(), sRequest)
	s.Nil(err)
	s.NotNil(resp.GetRunID())
	s.NotEqual(we.GetRunID(), resp.GetRunID())
	newWorkflowStarted = true

	// Process signal in decider
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	s.False(workflowComplete)
	s.True(signalEvent != nil)
	s.Equal(signalName, *signalEvent.WorkflowExecutionSignaledEventAttributes.SignalName)
	s.Equal(signalInput, signalEvent.WorkflowExecutionSignaledEventAttributes.Input)
	s.Equal(identity, *signalEvent.WorkflowExecutionSignaledEventAttributes.Identity)
	s.True(startedEvent != nil)
	s.Equal(header, startedEvent.WorkflowExecutionStartedEventAttributes.Header)

	// Send signal to not existed workflow
	id = "integration-signal-with-start-workflow-test-non-exist"
	signalName = "signal to non exist"
	signalInput = []byte("signal to non exist input.")
	sRequest.SignalName = common.StringPtr(signalName)
	sRequest.SignalInput = signalInput
	sRequest.WorkflowID = common.StringPtr(id)
	resp, err = s.engine.SignalWithStartWorkflowExecution(createContext(), sRequest)
	s.Nil(err)
	s.NotNil(resp.GetRunID())
	newWorkflowStarted = true

	// Process signal in decider
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)

	s.False(workflowComplete)
	s.True(signalEvent != nil)
	s.Equal(signalName, *signalEvent.WorkflowExecutionSignaledEventAttributes.SignalName)
	s.Equal(signalInput, signalEvent.WorkflowExecutionSignaledEventAttributes.Input)
	s.Equal(identity, *signalEvent.WorkflowExecutionSignaledEventAttributes.Identity)

	// Assert visibility is correct
	listOpenRequest := &types.ListOpenWorkflowExecutionsRequest{
		Domain:          common.StringPtr(s.domainName),
		MaximumPageSize: common.Int32Ptr(100),
		StartTimeFilter: &types.StartTimeFilter{
			EarliestTime: common.Int64Ptr(0),
			LatestTime:   common.Int64Ptr(time.Now().UnixNano()),
		},
		ExecutionFilter: &types.WorkflowExecutionFilter{
			WorkflowID: common.StringPtr(id),
		},
	}
	listResp, err := s.engine.ListOpenWorkflowExecutions(createContext(), listOpenRequest)
	s.NoError(err)
	s.Equal(1, len(listResp.Executions))

	// Terminate workflow execution and assert visibility is correct
	err = s.engine.TerminateWorkflowExecution(createContext(), &types.TerminateWorkflowExecutionRequest{
		Domain: common.StringPtr(s.domainName),
		WorkflowExecution: &types.WorkflowExecution{
			WorkflowID: common.StringPtr(id),
		},
		Reason:   common.StringPtr("kill workflow"),
		Details:  nil,
		Identity: common.StringPtr(identity),
	})
	s.Nil(err)

	for i := 0; i < 10; i++ { // retry
		listResp, err = s.engine.ListOpenWorkflowExecutions(createContext(), listOpenRequest)
		s.NoError(err)
		if len(listResp.Executions) == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.Equal(0, len(listResp.Executions))

	listClosedRequest := &types.ListClosedWorkflowExecutionsRequest{
		Domain:          common.StringPtr(s.domainName),
		MaximumPageSize: common.Int32Ptr(100),
		StartTimeFilter: &types.StartTimeFilter{
			EarliestTime: common.Int64Ptr(0),
			LatestTime:   common.Int64Ptr(time.Now().UnixNano()),
		},
		ExecutionFilter: &types.WorkflowExecutionFilter{
			WorkflowID: common.StringPtr(id),
		},
	}
	listClosedResp, err := s.engine.ListClosedWorkflowExecutions(createContext(), listClosedRequest)
	s.NoError(err)
	s.Equal(1, len(listClosedResp.Executions))
}

func (s *integrationSuite) TestSignalWithStartWorkflow_IDReusePolicy() {
	id := "integration-signal-with-start-workflow-id-reuse-test"
	wt := "integration-signal-with-start-workflow-id-reuse-test-type"
	tl := "integration-signal-with-start-workflow-id-reuse-test-tasklist"
	identity := "worker1"
	activityName := "activity_type1"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	// Start a workflow
	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.domainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            common.StringPtr(identity),
	}

	we, err0 := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err0)

	s.Logger.Info("StartWorkflowExecution", tag.WorkflowRunID(*we.RunID))

	workflowComplete := false
	activityCount := int32(1)
	activityCounter := int32(0)
	dtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {
		if activityCounter < activityCount {
			activityCounter++
			buf := new(bytes.Buffer)
			s.Nil(binary.Write(buf, binary.LittleEndian, activityCounter))

			return []byte(strconv.Itoa(int(activityCounter))), []*types.Decision{{
				DecisionType: types.DecisionTypeScheduleActivityTask.Ptr(),
				ScheduleActivityTaskDecisionAttributes: &types.ScheduleActivityTaskDecisionAttributes{
					ActivityID:                    common.StringPtr(strconv.Itoa(int(activityCounter))),
					ActivityType:                  &types.ActivityType{Name: common.StringPtr(activityName)},
					TaskList:                      &types.TaskList{Name: &tl},
					Input:                         buf.Bytes(),
					ScheduleToCloseTimeoutSeconds: common.Int32Ptr(100),
					ScheduleToStartTimeoutSeconds: common.Int32Ptr(10),
					StartToCloseTimeoutSeconds:    common.Int32Ptr(50),
					HeartbeatTimeoutSeconds:       common.Int32Ptr(5),
				},
			}}, nil
		}

		workflowComplete = true
		return []byte(strconv.Itoa(int(activityCounter))), []*types.Decision{{
			DecisionType: types.DecisionTypeCompleteWorkflowExecution.Ptr(),
			CompleteWorkflowExecutionDecisionAttributes: &types.CompleteWorkflowExecutionDecisionAttributes{
				Result: []byte("Done."),
			},
		}}, nil
	}

	atHandler := func(execution *types.WorkflowExecution, activityType *types.ActivityType,
		ActivityID string, input []byte, taskToken []byte) ([]byte, bool, error) {
		return []byte("Activity Result."), false, nil
	}

	poller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.domainName,
		TaskList:        taskList,
		Identity:        identity,
		DecisionHandler: dtHandler,
		ActivityHandler: atHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	// Start workflows, make some progress and complete workflow
	_, err := poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)
	_, err = poller.PollAndProcessDecisionTask(false, false)
	s.Logger.Info("PollAndProcessDecisionTask", tag.Error(err))
	s.Nil(err)
	s.True(workflowComplete)

	// test policy WorkflowIDReusePolicyRejectDuplicate
	signalName := "my signal"
	signalInput := []byte("my signal input.")
	wfIDReusePolicy := types.WorkflowIDReusePolicyRejectDuplicate
	sRequest := &types.SignalWithStartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              common.StringPtr(s.domainName),
		WorkflowID:                          common.StringPtr(id),
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		SignalName:                          common.StringPtr(signalName),
		SignalInput:                         signalInput,
		Identity:                            common.StringPtr(identity),
		WorkflowIDReusePolicy:               &wfIDReusePolicy,
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := s.engine.SignalWithStartWorkflowExecution(ctx, sRequest)
	s.Nil(resp)
	s.Error(err)
	errMsg := err.(*types.WorkflowExecutionAlreadyStartedError).GetMessage()
	s.True(strings.Contains(errMsg, "reject duplicate workflow ID"))

	// test policy WorkflowIDReusePolicyAllowDuplicateFailedOnly
	wfIDReusePolicy = types.WorkflowIDReusePolicyAllowDuplicateFailedOnly
	ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)
	resp, err = s.engine.SignalWithStartWorkflowExecution(ctx, sRequest)
	s.Nil(resp)
	s.Error(err)
	errMsg = err.(*types.WorkflowExecutionAlreadyStartedError).GetMessage()
	s.True(strings.Contains(errMsg, "allow duplicate workflow ID if last run failed"))

	// test policy WorkflowIDReusePolicyAllowDuplicate
	wfIDReusePolicy = types.WorkflowIDReusePolicyAllowDuplicate
	ctx, _ = context.WithTimeout(context.Background(), 5*time.Second)
	resp, err = s.engine.SignalWithStartWorkflowExecution(ctx, sRequest)
	s.Nil(err)
	s.NotEmpty(resp.GetRunID())

	// Terminate workflow execution
	terminateWorkflow := func() {
		err = s.engine.TerminateWorkflowExecution(createContext(), &types.TerminateWorkflowExecutionRequest{
			Domain: common.StringPtr(s.domainName),
			WorkflowExecution: &types.WorkflowExecution{
				WorkflowID: common.StringPtr(id),
			},
			Reason:   common.StringPtr("test WorkflowIDReusePolicyAllowDuplicateFailedOnly"),
			Details:  nil,
			Identity: common.StringPtr(identity),
		})
		s.Nil(err)
	}
	terminateWorkflow()

	// test policy WorkflowIDReusePolicyAllowDuplicateFailedOnly success start
	wfIDReusePolicy = types.WorkflowIDReusePolicyAllowDuplicateFailedOnly
	resp, err = s.engine.SignalWithStartWorkflowExecution(createContext(), sRequest)
	s.Nil(err)
	s.NotEmpty(resp.GetRunID())

	// test policy WorkflowIDReusePolicyTerminateIfRunning
	wfIDReusePolicy = types.WorkflowIDReusePolicyTerminateIfRunning
	sRequest.RequestID = common.StringPtr(uuid.New())
	resp1, err1 := s.engine.SignalWithStartWorkflowExecution(createContext(), sRequest)
	s.Nil(err1)
	s.NotEmpty(resp1)
	// verify terminate status
	executionTerminated := false
GetHistoryLoop:
	for i := 0; i < 10; i++ {
		historyResponse, err := s.engine.GetWorkflowExecutionHistory(createContext(), &types.GetWorkflowExecutionHistoryRequest{
			Domain: common.StringPtr(s.domainName),
			Execution: &types.WorkflowExecution{
				WorkflowID: common.StringPtr(id),
				RunID:      resp.RunID,
			},
		})
		s.Nil(err)
		history := historyResponse.History

		lastEvent := history.Events[len(history.Events)-1]
		if lastEvent.GetEventType() != types.EventTypeWorkflowExecutionTerminated {
			s.Logger.Warn("Execution not terminated yet.")
			time.Sleep(100 * time.Millisecond)
			continue GetHistoryLoop
		}

		terminateEventAttributes := lastEvent.WorkflowExecutionTerminatedEventAttributes
		s.Equal(cadencehistory.TerminateIfRunningReason, terminateEventAttributes.GetReason())
		s.Equal(fmt.Sprintf(cadencehistory.TerminateIfRunningDetailsTemplate, resp1.GetRunID()), string(terminateEventAttributes.Details))
		s.Equal(execution.IdentityHistoryService, terminateEventAttributes.GetIdentity())
		executionTerminated = true
		break GetHistoryLoop
	}
	s.True(executionTerminated)
	// terminate current run
	terminateWorkflow()
	// test clean start with WorkflowIDReusePolicyTerminateIfRunning
	sRequest.RequestID = common.StringPtr(uuid.New())
	resp2, err2 := s.engine.SignalWithStartWorkflowExecution(createContext(), sRequest)
	s.Nil(err2)
	s.NotEmpty(resp2)
	s.NotEqual(resp1.GetRunID(), resp2.GetRunID())
}
