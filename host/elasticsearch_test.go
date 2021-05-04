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

// +build esintegration

// to run locally, make sure kafka and es is running,
// then run cmd `go test -v ./host -run TestElasticsearchIntegrationSuite -tags esintegration`
package host

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/definition"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/types"
	"github.com/uber/cadence/environment"
	"github.com/uber/cadence/host/esutils"
)

const (
	numOfRetry        = 50
	waitTimeInMs      = 400
	waitForESToSettle = 4 * time.Second // wait es shards for some time ensure data consistent
)

type elasticsearchIntegrationSuite struct {
	// override suite.Suite.Assertions with require.Assertions; this means that s.NotNil(nil) will stop the test,
	// not merely log an error
	*require.Assertions
	IntegrationBase
	esClient esutils.ESClient

	testSearchAttributeKey string
	testSearchAttributeVal string
}

// This cluster use customized threshold for history config
func (s *elasticsearchIntegrationSuite) SetupSuite() {
	s.setupSuite("testdata/integration_elasticsearch_" + environment.GetESVersion() + "_cluster.yaml")
	s.esClient = esutils.CreateESClient(s.Suite, s.testClusterConfig.ESConfig.URL.String(), environment.GetESVersion())
	s.esClient.PutIndexTemplate(s.Suite, "testdata/es_"+environment.GetESVersion()+"_index_template.json", "test-visibility-template")
	indexName := s.testClusterConfig.ESConfig.Indices[common.VisibilityAppName]
	s.esClient.CreateIndex(s.Suite, indexName)
	s.putIndexSettings(indexName, defaultTestValueOfESIndexMaxResultWindow)
}

func (s *elasticsearchIntegrationSuite) TearDownSuite() {
	s.tearDownSuite()
	s.esClient.DeleteIndex(s.Suite, s.testClusterConfig.ESConfig.Indices[common.VisibilityAppName])
}

func (s *elasticsearchIntegrationSuite) SetupTest() {
	// Have to define our overridden assertions in the test setup. If we did it earlier, s.T() will return nil
	s.Assertions = require.New(s.T())
	s.testSearchAttributeKey = definition.CustomStringField
	s.testSearchAttributeVal = "test value"
}

func TestElasticsearchIntegrationSuite(t *testing.T) {
	flag.Parse()
	suite.Run(t, new(elasticsearchIntegrationSuite))
}

func (s *elasticsearchIntegrationSuite) TestListOpenWorkflow() {
	id := "es-integration-start-workflow-test"
	wt := "es-integration-start-workflow-test-type"
	tl := "es-integration-start-workflow-test-tasklist"
	request := s.createStartWorkflowExecutionRequest(id, wt, tl)

	attrValBytes, _ := json.Marshal(s.testSearchAttributeVal)
	searchAttr := &types.SearchAttributes{
		IndexedFields: map[string][]byte{
			s.testSearchAttributeKey: attrValBytes,
		},
	}
	request.SearchAttributes = searchAttr

	startTime := time.Now().UnixNano()
	we, err := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err)

	startFilter := &types.StartTimeFilter{}
	startFilter.EarliestTime = common.Int64Ptr(startTime)
	var openExecution *types.WorkflowExecutionInfo
	for i := 0; i < numOfRetry; i++ {
		startFilter.LatestTime = common.Int64Ptr(time.Now().UnixNano())
		resp, err := s.engine.ListOpenWorkflowExecutions(createContext(), &types.ListOpenWorkflowExecutionsRequest{
			Domain:          s.domainName,
			MaximumPageSize: common.Int32Ptr(defaultTestValueOfESIndexMaxResultWindow),
			StartTimeFilter: startFilter,
			ExecutionFilter: &types.WorkflowExecutionFilter{
				WorkflowID: common.StringPtr(id),
			},
		})
		s.Nil(err)
		if len(resp.GetExecutions()) == 1 {
			openExecution = resp.GetExecutions()[0]
			break
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.NotNil(openExecution)
	s.Equal(we.GetRunID(), openExecution.GetExecution().GetRunID())
	s.Equal(attrValBytes, openExecution.SearchAttributes.GetIndexedFields()[s.testSearchAttributeKey])
}

func (s *elasticsearchIntegrationSuite) TestListWorkflow() {
	id := "es-integration-list-workflow-test"
	wt := "es-integration-list-workflow-test-type"
	tl := "es-integration-list-workflow-test-tasklist"
	request := s.createStartWorkflowExecutionRequest(id, wt, tl)

	we, err := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err)

	query := fmt.Sprintf(`WorkflowID = "%s"`, id)
	s.testHelperForReadOnce(we.GetRunID(), query, false)
}

func (s *elasticsearchIntegrationSuite) TestListWorkflow_ExecutionTime() {
	id := "es-integration-list-workflow-execution-time-test"
	wt := "es-integration-list-workflow-execution-time-test-type"
	tl := "es-integration-list-workflow-execution-time-test-tasklist"
	request := s.createStartWorkflowExecutionRequest(id, wt, tl)

	we, err := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err)

	cronID := id + "-cron"
	request.CronSchedule = common.StringPtr("@every 1m")
	request.WorkflowID = cronID

	weCron, err := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err)

	query := fmt.Sprintf(`(WorkflowID = "%s" or WorkflowID = "%s") and ExecutionTime < %v`, id, cronID, time.Now().UnixNano()+int64(time.Minute))
	s.testHelperForReadOnce(weCron.GetRunID(), query, false)

	query = fmt.Sprintf(`WorkflowID = "%s"`, id)
	s.testHelperForReadOnce(we.GetRunID(), query, false)
}

func (s *elasticsearchIntegrationSuite) TestListWorkflow_SearchAttribute() {
	id := "es-integration-list-workflow-by-search-attr-test"
	wt := "es-integration-list-workflow-by-search-attr-test-type"
	tl := "es-integration-list-workflow-by-search-attr-test-tasklist"
	request := s.createStartWorkflowExecutionRequest(id, wt, tl)

	attrValBytes, _ := json.Marshal(s.testSearchAttributeVal)
	searchAttr := &types.SearchAttributes{
		IndexedFields: map[string][]byte{
			s.testSearchAttributeKey: attrValBytes,
		},
	}
	request.SearchAttributes = searchAttr

	we, err := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err)
	query := fmt.Sprintf(`WorkflowID = "%s" and %s = "%s"`, id, s.testSearchAttributeKey, s.testSearchAttributeVal)
	s.testHelperForReadOnce(we.GetRunID(), query, false)

	// test upsert
	dtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {

		upsertDecision := &types.Decision{
			DecisionType: types.DecisionTypeUpsertWorkflowSearchAttributes.Ptr(),
			UpsertWorkflowSearchAttributesDecisionAttributes: &types.UpsertWorkflowSearchAttributesDecisionAttributes{
				SearchAttributes: getUpsertSearchAttributes(),
			}}

		return nil, []*types.Decision{upsertDecision}, nil
	}
	taskList := &types.TaskList{Name: common.StringPtr(tl)}
	poller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.domainName,
		TaskList:        taskList,
		StickyTaskList:  taskList,
		Identity:        "worker1",
		DecisionHandler: dtHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}
	_, newTask, err := poller.PollAndProcessDecisionTaskWithAttemptAndRetryAndForceNewDecision(
		false,
		false,
		true,
		true,
		int64(0),
		1,
		true,
		nil)
	s.Nil(err)
	s.NotNil(newTask)
	s.NotNil(newTask.DecisionTask)

	time.Sleep(waitForESToSettle)

	listRequest := &types.ListWorkflowExecutionsRequest{
		Domain:   s.domainName,
		PageSize: common.Int32Ptr(int32(2)),
		Query:    common.StringPtr(fmt.Sprintf(`WorkflowType = '%s' and CloseTime = missing and BinaryChecksums = 'binary-v1'`, wt)),
	}
	// verify upsert data is on ES
	s.testListResultForUpsertSearchAttributes(listRequest)

	// verify DescribeWorkflowExecution
	descRequest := &types.DescribeWorkflowExecutionRequest{
		Domain: s.domainName,
		Execution: &types.WorkflowExecution{
			WorkflowID: id,
		},
	}
	descResp, err := s.engine.DescribeWorkflowExecution(createContext(), descRequest)
	s.Nil(err)
	expectedSearchAttributes := getUpsertSearchAttributes()
	s.Equal(expectedSearchAttributes, descResp.WorkflowExecutionInfo.GetSearchAttributes())
}

func (s *elasticsearchIntegrationSuite) TestListWorkflow_PageToken() {
	id := "es-integration-list-workflow-token-test"
	wt := "es-integration-list-workflow-token-test-type"
	tl := "es-integration-list-workflow-token-test-tasklist"
	request := s.createStartWorkflowExecutionRequest(id, wt, tl)

	numOfWorkflows := defaultTestValueOfESIndexMaxResultWindow - 1 // == 4
	pageSize := 3

	s.testListWorkflowHelper(numOfWorkflows, pageSize, request, id, wt, false)
}

func (s *elasticsearchIntegrationSuite) TestListWorkflow_SearchAfter() {
	id := "es-integration-list-workflow-searchAfter-test"
	wt := "es-integration-list-workflow-searchAfter-test-type"
	tl := "es-integration-list-workflow-searchAfter-test-tasklist"
	request := s.createStartWorkflowExecutionRequest(id, wt, tl)

	numOfWorkflows := defaultTestValueOfESIndexMaxResultWindow + 1 // == 6
	pageSize := 4

	s.testListWorkflowHelper(numOfWorkflows, pageSize, request, id, wt, false)
}

func (s *elasticsearchIntegrationSuite) TestListWorkflow_OrQuery() {
	id := "es-integration-list-workflow-or-query-test"
	wt := "es-integration-list-workflow-or-query-test-type"
	tl := "es-integration-list-workflow-or-query-test-tasklist"
	request := s.createStartWorkflowExecutionRequest(id, wt, tl)

	// start 3 workflows
	key := definition.CustomIntField
	attrValBytes, _ := json.Marshal(1)
	searchAttr := &types.SearchAttributes{
		IndexedFields: map[string][]byte{
			key: attrValBytes,
		},
	}
	request.SearchAttributes = searchAttr
	we1, err := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err)

	request.RequestID = common.StringPtr(uuid.New())
	request.WorkflowID = id + "-2"
	attrValBytes, _ = json.Marshal(2)
	searchAttr.IndexedFields[key] = attrValBytes
	we2, err := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err)

	request.RequestID = common.StringPtr(uuid.New())
	request.WorkflowID = id + "-3"
	attrValBytes, _ = json.Marshal(3)
	searchAttr.IndexedFields[key] = attrValBytes
	we3, err := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err)

	time.Sleep(waitForESToSettle)

	// query 1 workflow with search attr
	query1 := fmt.Sprintf(`CustomIntField = %d`, 1)
	var openExecution *types.WorkflowExecutionInfo
	listRequest := &types.ListWorkflowExecutionsRequest{
		Domain:   s.domainName,
		PageSize: common.Int32Ptr(defaultTestValueOfESIndexMaxResultWindow),
		Query:    common.StringPtr(query1),
	}
	for i := 0; i < numOfRetry; i++ {
		resp, err := s.engine.ListWorkflowExecutions(createContext(), listRequest)
		s.Nil(err)
		if len(resp.GetExecutions()) == 1 {
			openExecution = resp.GetExecutions()[0]
			break
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.NotNil(openExecution)
	s.Equal(we1.GetRunID(), openExecution.GetExecution().GetRunID())
	s.True(openExecution.GetExecutionTime() >= openExecution.GetStartTime())
	searchValBytes := openExecution.SearchAttributes.GetIndexedFields()[key]
	var searchVal int
	json.Unmarshal(searchValBytes, &searchVal)
	s.Equal(1, searchVal)

	// query with or clause
	query2 := fmt.Sprintf(`CustomIntField = %d or CustomIntField = %d`, 1, 2)
	listRequest.Query = common.StringPtr(query2)
	var openExecutions []*types.WorkflowExecutionInfo
	for i := 0; i < numOfRetry; i++ {
		resp, err := s.engine.ListWorkflowExecutions(createContext(), listRequest)
		s.Nil(err)
		if len(resp.GetExecutions()) == 2 {
			openExecutions = resp.GetExecutions()
			break
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.Equal(2, len(openExecutions))
	e1 := openExecutions[0]
	e2 := openExecutions[1]
	if e1.GetExecution().GetRunID() != we1.GetRunID() {
		// results are sorted by [CloseTime,RunID] desc, so find the correct mapping first
		e1, e2 = e2, e1
	}
	s.Equal(we1.GetRunID(), e1.GetExecution().GetRunID())
	s.Equal(we2.GetRunID(), e2.GetExecution().GetRunID())
	searchValBytes = e2.SearchAttributes.GetIndexedFields()[key]
	json.Unmarshal(searchValBytes, &searchVal)
	s.Equal(2, searchVal)

	// query for open
	query3 := fmt.Sprintf(`(CustomIntField = %d or CustomIntField = %d) and CloseTime = missing`, 2, 3)
	listRequest.Query = common.StringPtr(query3)
	for i := 0; i < numOfRetry; i++ {
		resp, err := s.engine.ListWorkflowExecutions(createContext(), listRequest)
		s.Nil(err)
		if len(resp.GetExecutions()) == 2 {
			openExecutions = resp.GetExecutions()
			break
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.Equal(2, len(openExecutions))
	e1 = openExecutions[0]
	e2 = openExecutions[1]
	s.Equal(we3.GetRunID(), e1.GetExecution().GetRunID())
	s.Equal(we2.GetRunID(), e2.GetExecution().GetRunID())
	searchValBytes = e1.SearchAttributes.GetIndexedFields()[key]
	json.Unmarshal(searchValBytes, &searchVal)
	s.Equal(3, searchVal)
}

// To test last page search trigger max window size error
func (s *elasticsearchIntegrationSuite) TestListWorkflow_MaxWindowSize() {
	id := "es-integration-list-workflow-max-window-size-test"
	wt := "es-integration-list-workflow-max-window-size-test-type"
	tl := "es-integration-list-workflow-max-window-size-test-tasklist"
	startRequest := s.createStartWorkflowExecutionRequest(id, wt, tl)

	for i := 0; i < defaultTestValueOfESIndexMaxResultWindow; i++ {
		startRequest.RequestID = common.StringPtr(uuid.New())
		startRequest.WorkflowID = id + strconv.Itoa(i)
		_, err := s.engine.StartWorkflowExecution(createContext(), startRequest)
		s.Nil(err)
	}

	time.Sleep(waitForESToSettle)

	var listResp *types.ListWorkflowExecutionsResponse
	var nextPageToken []byte

	listRequest := &types.ListWorkflowExecutionsRequest{
		Domain:        s.domainName,
		PageSize:      common.Int32Ptr(int32(defaultTestValueOfESIndexMaxResultWindow)),
		NextPageToken: nextPageToken,
		Query:         common.StringPtr(fmt.Sprintf(`WorkflowType = '%s' and CloseTime = missing`, wt)),
	}
	// get first page
	for i := 0; i < numOfRetry; i++ {
		resp, err := s.engine.ListWorkflowExecutions(createContext(), listRequest)
		s.Nil(err)
		if len(resp.GetExecutions()) == defaultTestValueOfESIndexMaxResultWindow {
			listResp = resp
			break
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.NotNil(listResp)
	s.True(len(listResp.GetNextPageToken()) != 0)

	// the last request
	listRequest.NextPageToken = listResp.GetNextPageToken()
	resp, err := s.engine.ListWorkflowExecutions(createContext(), listRequest)
	s.Nil(err)
	s.True(len(resp.GetExecutions()) == 0)
	s.True(len(resp.GetNextPageToken()) == 0)
}

func (s *elasticsearchIntegrationSuite) TestListWorkflow_OrderBy() {
	id := "es-integration-list-workflow-order-by-test"
	wt := "es-integration-list-workflow-order-by-test-type"
	tl := "es-integration-list-workflow-order-by-test-tasklist"
	startRequest := s.createStartWorkflowExecutionRequest(id, wt, tl)

	for i := 0; i < defaultTestValueOfESIndexMaxResultWindow+1; i++ { // start 6
		startRequest.RequestID = common.StringPtr(uuid.New())
		startRequest.WorkflowID = id + strconv.Itoa(i)

		if i < defaultTestValueOfESIndexMaxResultWindow-1 { // 4 workflow has search attr
			intVal, _ := json.Marshal(i)
			doubleVal, _ := json.Marshal(float64(i))
			strVal, _ := json.Marshal(strconv.Itoa(i))
			timeVal, _ := json.Marshal(time.Now())
			searchAttr := &types.SearchAttributes{
				IndexedFields: map[string][]byte{
					definition.CustomIntField:      intVal,
					definition.CustomDoubleField:   doubleVal,
					definition.CustomKeywordField:  strVal,
					definition.CustomDatetimeField: timeVal,
				},
			}
			startRequest.SearchAttributes = searchAttr
		} else {
			startRequest.SearchAttributes = &types.SearchAttributes{}
		}

		_, err := s.engine.StartWorkflowExecution(createContext(), startRequest)
		s.Nil(err)
	}

	time.Sleep(waitForESToSettle)

	desc := "desc"
	asc := "asc"
	queryTemplate := `WorkflowType = "%s" order by %s %s`
	pageSize := int32(defaultTestValueOfESIndexMaxResultWindow)

	// order by CloseTime asc
	query1 := fmt.Sprintf(queryTemplate, wt, definition.CloseTime, asc)
	var openExecutions []*types.WorkflowExecutionInfo
	listRequest := &types.ListWorkflowExecutionsRequest{
		Domain:   s.domainName,
		PageSize: common.Int32Ptr(pageSize),
		Query:    common.StringPtr(query1),
	}
	for i := 0; i < numOfRetry; i++ {
		resp, err := s.engine.ListWorkflowExecutions(createContext(), listRequest)
		s.Nil(err)
		if int32(len(resp.GetExecutions())) == listRequest.GetPageSize() {
			openExecutions = resp.GetExecutions()
			break
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.NotNil(openExecutions)
	for i := int32(1); i < pageSize; i++ {
		s.True(openExecutions[i-1].GetCloseTime() <= openExecutions[i].GetCloseTime())
	}

	// greatest effort to reduce duplicate code
	testHelper := func(query, searchAttrKey string, prevVal, currVal interface{}) {
		listRequest.Query = common.StringPtr(query)
		listRequest.NextPageToken = []byte{}
		resp, err := s.engine.ListWorkflowExecutions(createContext(), listRequest)
		s.Nil(err)
		openExecutions = resp.GetExecutions()
		dec := json.NewDecoder(bytes.NewReader(openExecutions[0].GetSearchAttributes().GetIndexedFields()[searchAttrKey]))
		dec.UseNumber()
		err = dec.Decode(&prevVal)
		s.Nil(err)
		for i := int32(1); i < pageSize; i++ {
			indexedFields := openExecutions[i].GetSearchAttributes().GetIndexedFields()
			searchAttrBytes, ok := indexedFields[searchAttrKey]
			if !ok { // last one doesn't have search attr
				s.Equal(pageSize-1, i)
				break
			}
			dec := json.NewDecoder(bytes.NewReader(searchAttrBytes))
			dec.UseNumber()
			err = dec.Decode(&currVal)
			s.Nil(err)
			var v1, v2 interface{}
			switch searchAttrKey {
			case definition.CustomIntField:
				v1, _ = prevVal.(json.Number).Int64()
				v2, _ = currVal.(json.Number).Int64()
				s.True(v1.(int64) >= v2.(int64))
			case definition.CustomDoubleField:
				v1, _ = prevVal.(json.Number).Float64()
				v2, _ = currVal.(json.Number).Float64()
				s.True(v1.(float64) >= v2.(float64))
			case definition.CustomKeywordField:
				s.True(prevVal.(string) >= currVal.(string))
			case definition.CustomDatetimeField:
				v1, _ = time.Parse(time.RFC3339, prevVal.(string))
				v2, _ = time.Parse(time.RFC3339, currVal.(string))
				s.True(v1.(time.Time).After(v2.(time.Time)))
			}
			prevVal = currVal
		}
		listRequest.NextPageToken = resp.GetNextPageToken()
		resp, err = s.engine.ListWorkflowExecutions(createContext(), listRequest) // last page
		s.Nil(err)
		s.Equal(1, len(resp.GetExecutions()))
	}

	// order by CustomIntField desc
	field := definition.CustomIntField
	query := fmt.Sprintf(queryTemplate, wt, field, desc)
	var int1, int2 int
	testHelper(query, field, int1, int2)

	// order by CustomDoubleField desc
	field = definition.CustomDoubleField
	query = fmt.Sprintf(queryTemplate, wt, field, desc)
	var double1, double2 float64
	testHelper(query, field, double1, double2)

	// order by CustomKeywordField desc
	field = definition.CustomKeywordField
	query = fmt.Sprintf(queryTemplate, wt, field, desc)
	var s1, s2 string
	testHelper(query, field, s1, s2)

	// order by CustomDatetimeField desc
	field = definition.CustomDatetimeField
	query = fmt.Sprintf(queryTemplate, wt, field, desc)
	var t1, t2 time.Time
	testHelper(query, field, t1, t2)
}

func (s *elasticsearchIntegrationSuite) testListWorkflowHelper(numOfWorkflows, pageSize int,
	startRequest *types.StartWorkflowExecutionRequest, wid, wType string, isScan bool) {

	// start enough number of workflows
	for i := 0; i < numOfWorkflows; i++ {
		startRequest.RequestID = common.StringPtr(uuid.New())
		startRequest.WorkflowID = wid + strconv.Itoa(i)
		_, err := s.engine.StartWorkflowExecution(createContext(), startRequest)
		s.Nil(err)
	}

	time.Sleep(waitForESToSettle)

	var openExecutions []*types.WorkflowExecutionInfo
	var nextPageToken []byte

	listRequest := &types.ListWorkflowExecutionsRequest{
		Domain:        s.domainName,
		PageSize:      common.Int32Ptr(int32(pageSize)),
		NextPageToken: nextPageToken,
		Query:         common.StringPtr(fmt.Sprintf(`WorkflowType = '%s' and CloseTime = missing`, wType)),
	}
	// test first page
	for i := 0; i < numOfRetry; i++ {
		var resp *types.ListWorkflowExecutionsResponse
		var err error

		if isScan {
			resp, err = s.engine.ScanWorkflowExecutions(createContext(), listRequest)
		} else {
			resp, err = s.engine.ListWorkflowExecutions(createContext(), listRequest)
		}
		s.Nil(err)
		if len(resp.GetExecutions()) == pageSize {
			openExecutions = resp.GetExecutions()
			nextPageToken = resp.GetNextPageToken()
			break
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.NotNil(openExecutions)
	s.NotNil(nextPageToken)
	s.True(len(nextPageToken) > 0)

	// test last page
	listRequest.NextPageToken = nextPageToken
	inIf := false
	for i := 0; i < numOfRetry; i++ {
		var resp *types.ListWorkflowExecutionsResponse
		var err error

		if isScan {
			resp, err = s.engine.ScanWorkflowExecutions(createContext(), listRequest)
		} else {
			resp, err = s.engine.ListWorkflowExecutions(createContext(), listRequest)
		}
		s.Nil(err)
		if len(resp.GetExecutions()) == numOfWorkflows-pageSize {
			inIf = true
			openExecutions = resp.GetExecutions()
			nextPageToken = resp.GetNextPageToken()
			break
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.True(inIf)
	s.NotNil(openExecutions)
	s.Nil(nextPageToken)
}

func (s *elasticsearchIntegrationSuite) testHelperForReadOnce(runID, query string, isScan bool) {
	var openExecution *types.WorkflowExecutionInfo
	listRequest := &types.ListWorkflowExecutionsRequest{
		Domain:   s.domainName,
		PageSize: common.Int32Ptr(defaultTestValueOfESIndexMaxResultWindow),
		Query:    common.StringPtr(query),
	}
	for i := 0; i < numOfRetry; i++ {
		var resp *types.ListWorkflowExecutionsResponse
		var err error

		if isScan {
			resp, err = s.engine.ScanWorkflowExecutions(createContext(), listRequest)
		} else {
			resp, err = s.engine.ListWorkflowExecutions(createContext(), listRequest)
		}

		s.Nil(err)
		if len(resp.GetExecutions()) == 1 {
			openExecution = resp.GetExecutions()[0]
			break
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.NotNil(openExecution)
	s.Equal(runID, openExecution.GetExecution().GetRunID())
	s.True(openExecution.GetExecutionTime() >= openExecution.GetStartTime())
	if openExecution.SearchAttributes != nil && len(openExecution.SearchAttributes.GetIndexedFields()) > 0 {
		searchValBytes := openExecution.SearchAttributes.GetIndexedFields()[s.testSearchAttributeKey]
		var searchVal string
		json.Unmarshal(searchValBytes, &searchVal)
		s.Equal(s.testSearchAttributeVal, searchVal)
	}
}

func (s *elasticsearchIntegrationSuite) TestScanWorkflow() {
	id := "es-integration-scan-workflow-test"
	wt := "es-integration-scan-workflow-test-type"
	tl := "es-integration-scan-workflow-test-tasklist"
	identity := "worker1"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              s.domainName,
		WorkflowID:                          id,
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            identity,
	}

	we, err := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err)
	query := fmt.Sprintf(`WorkflowID = "%s"`, id)
	s.testHelperForReadOnce(we.GetRunID(), query, true)
}

func (s *elasticsearchIntegrationSuite) TestScanWorkflow_SearchAttribute() {
	id := "es-integration-scan-workflow-search-attr-test"
	wt := "es-integration-scan-workflow-search-attr-test-type"
	tl := "es-integration-scan-workflow-search-attr-test-tasklist"
	request := s.createStartWorkflowExecutionRequest(id, wt, tl)

	attrValBytes, _ := json.Marshal(s.testSearchAttributeVal)
	searchAttr := &types.SearchAttributes{
		IndexedFields: map[string][]byte{
			s.testSearchAttributeKey: attrValBytes,
		},
	}
	request.SearchAttributes = searchAttr

	we, err := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err)
	query := fmt.Sprintf(`WorkflowID = "%s" and %s = "%s"`, id, s.testSearchAttributeKey, s.testSearchAttributeVal)
	s.testHelperForReadOnce(we.GetRunID(), query, true)
}

func (s *elasticsearchIntegrationSuite) TestScanWorkflow_PageToken() {
	id := "es-integration-scan-workflow-token-test"
	wt := "es-integration-scan-workflow-token-test-type"
	tl := "es-integration-scan-workflow-token-test-tasklist"
	identity := "worker1"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	request := &types.StartWorkflowExecutionRequest{
		Domain:                              s.domainName,
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            identity,
	}

	numOfWorkflows := 4
	pageSize := 3

	s.testListWorkflowHelper(numOfWorkflows, pageSize, request, id, wt, true)
}

func (s *elasticsearchIntegrationSuite) TestCountWorkflow() {
	id := "es-integration-count-workflow-test"
	wt := "es-integration-count-workflow-test-type"
	tl := "es-integration-count-workflow-test-tasklist"
	request := s.createStartWorkflowExecutionRequest(id, wt, tl)

	attrValBytes, _ := json.Marshal(s.testSearchAttributeVal)
	searchAttr := &types.SearchAttributes{
		IndexedFields: map[string][]byte{
			s.testSearchAttributeKey: attrValBytes,
		},
	}
	request.SearchAttributes = searchAttr

	_, err := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err)

	query := fmt.Sprintf(`WorkflowID = "%s" and %s = "%s"`, id, s.testSearchAttributeKey, s.testSearchAttributeVal)
	countRequest := &types.CountWorkflowExecutionsRequest{
		Domain: s.domainName,
		Query:  common.StringPtr(query),
	}
	var resp *types.CountWorkflowExecutionsResponse
	for i := 0; i < numOfRetry; i++ {
		resp, err = s.engine.CountWorkflowExecutions(createContext(), countRequest)
		s.Nil(err)
		if resp.GetCount() == int64(1) {
			break
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.Equal(int64(1), resp.GetCount())

	query = fmt.Sprintf(`WorkflowID = "%s" and %s = "%s"`, id, s.testSearchAttributeKey, "noMatch")
	countRequest.Query = common.StringPtr(query)
	resp, err = s.engine.CountWorkflowExecutions(createContext(), countRequest)
	s.Nil(err)
	s.Equal(int64(0), resp.GetCount())
}

func (s *elasticsearchIntegrationSuite) createStartWorkflowExecutionRequest(id, wt, tl string) *types.StartWorkflowExecutionRequest {
	identity := "worker1"
	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              s.domainName,
		WorkflowID:                          id,
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            identity,
	}
	return request
}

func (s *elasticsearchIntegrationSuite) TestUpsertWorkflowExecution() {
	id := "es-integration-upsert-workflow-test"
	wt := "es-integration-upsert-workflow-test-type"
	tl := "es-integration-upsert-workflow-test-tasklist"
	identity := "worker1"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              s.domainName,
		WorkflowID:                          id,
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            identity,
	}

	we, err0 := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err0)

	s.Logger.Info("StartWorkflowExecution", tag.WorkflowRunID(we.RunID))

	decisionCount := 0
	dtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {

		upsertDecision := &types.Decision{
			DecisionType: types.DecisionTypeUpsertWorkflowSearchAttributes.Ptr(),
			UpsertWorkflowSearchAttributesDecisionAttributes: &types.UpsertWorkflowSearchAttributesDecisionAttributes{}}

		// handle first upsert
		if decisionCount == 0 {
			decisionCount++

			attrValBytes, _ := json.Marshal(s.testSearchAttributeVal)
			upsertSearchAttr := &types.SearchAttributes{
				IndexedFields: map[string][]byte{
					s.testSearchAttributeKey: attrValBytes,
				},
			}
			upsertDecision.UpsertWorkflowSearchAttributesDecisionAttributes.SearchAttributes = upsertSearchAttr
			return nil, []*types.Decision{upsertDecision}, nil
		}
		// handle second upsert, which update existing field and add new field
		if decisionCount == 1 {
			decisionCount++
			upsertDecision.UpsertWorkflowSearchAttributesDecisionAttributes.SearchAttributes = getUpsertSearchAttributes()
			return nil, []*types.Decision{upsertDecision}, nil
		}

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
		StickyTaskList:  taskList,
		Identity:        identity,
		DecisionHandler: dtHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	// process 1st decision and assert decision is handled correctly.
	_, newTask, err := poller.PollAndProcessDecisionTaskWithAttemptAndRetryAndForceNewDecision(
		false,
		false,
		true,
		true,
		int64(0),
		1,
		true,
		nil)
	s.Nil(err)
	s.NotNil(newTask)
	s.NotNil(newTask.DecisionTask)
	s.Equal(int64(3), newTask.DecisionTask.GetPreviousStartedEventID())
	s.Equal(int64(7), newTask.DecisionTask.GetStartedEventID())
	s.Equal(4, len(newTask.DecisionTask.History.Events))
	s.Equal(types.EventTypeDecisionTaskCompleted, newTask.DecisionTask.History.Events[0].GetEventType())
	s.Equal(types.EventTypeUpsertWorkflowSearchAttributes, newTask.DecisionTask.History.Events[1].GetEventType())
	s.Equal(types.EventTypeDecisionTaskScheduled, newTask.DecisionTask.History.Events[2].GetEventType())
	s.Equal(types.EventTypeDecisionTaskStarted, newTask.DecisionTask.History.Events[3].GetEventType())

	time.Sleep(waitForESToSettle)

	// verify upsert data is on ES
	listRequest := &types.ListWorkflowExecutionsRequest{
		Domain:   s.domainName,
		PageSize: common.Int32Ptr(int32(2)),
		Query:    common.StringPtr(fmt.Sprintf(`WorkflowType = '%s' and CloseTime = missing`, wt)),
	}
	verified := false
	for i := 0; i < numOfRetry; i++ {
		resp, err := s.engine.ListWorkflowExecutions(createContext(), listRequest)
		s.Nil(err)
		if len(resp.GetExecutions()) == 1 {
			execution := resp.GetExecutions()[0]
			retrievedSearchAttr := execution.SearchAttributes
			if retrievedSearchAttr != nil && len(retrievedSearchAttr.GetIndexedFields()) > 0 {
				searchValBytes := retrievedSearchAttr.GetIndexedFields()[s.testSearchAttributeKey]
				var searchVal string
				json.Unmarshal(searchValBytes, &searchVal)
				s.Equal(s.testSearchAttributeVal, searchVal)
				verified = true
				break
			}
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.True(verified)

	// process 2nd decision and assert decision is handled correctly.
	_, newTask, err = poller.PollAndProcessDecisionTaskWithAttemptAndRetryAndForceNewDecision(
		false,
		false,
		true,
		true,
		int64(0),
		1,
		true,
		nil)
	s.Nil(err)
	s.NotNil(newTask)
	s.NotNil(newTask.DecisionTask)
	s.Equal(4, len(newTask.DecisionTask.History.Events))
	s.Equal(types.EventTypeDecisionTaskCompleted, newTask.DecisionTask.History.Events[0].GetEventType())
	s.Equal(types.EventTypeUpsertWorkflowSearchAttributes, newTask.DecisionTask.History.Events[1].GetEventType())
	s.Equal(types.EventTypeDecisionTaskScheduled, newTask.DecisionTask.History.Events[2].GetEventType())
	s.Equal(types.EventTypeDecisionTaskStarted, newTask.DecisionTask.History.Events[3].GetEventType())

	time.Sleep(waitForESToSettle)

	// verify upsert data is on ES
	s.testListResultForUpsertSearchAttributes(listRequest)
}

func (s *elasticsearchIntegrationSuite) testListResultForUpsertSearchAttributes(listRequest *types.ListWorkflowExecutionsRequest) {
	verified := false
	for i := 0; i < numOfRetry; i++ {
		resp, err := s.engine.ListWorkflowExecutions(createContext(), listRequest)
		s.Nil(err)
		if len(resp.GetExecutions()) == 1 {
			execution := resp.GetExecutions()[0]
			retrievedSearchAttr := execution.SearchAttributes
			if retrievedSearchAttr != nil && len(retrievedSearchAttr.GetIndexedFields()) == 3 {
				fields := retrievedSearchAttr.GetIndexedFields()
				searchValBytes := fields[s.testSearchAttributeKey]
				var searchVal string
				err := json.Unmarshal(searchValBytes, &searchVal)
				s.Nil(err)
				s.Equal("another string", searchVal)

				searchValBytes2 := fields[definition.CustomIntField]
				var searchVal2 int
				err = json.Unmarshal(searchValBytes2, &searchVal2)
				s.Nil(err)
				s.Equal(123, searchVal2)

				binaryChecksumsBytes := fields[definition.BinaryChecksums]
				var binaryChecksums []string
				err = json.Unmarshal(binaryChecksumsBytes, &binaryChecksums)
				s.Nil(err)
				s.Equal([]string{"binary-v1", "binary-v2"}, binaryChecksums)

				verified = true
				break
			}
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.True(verified)
}

func getUpsertSearchAttributes() *types.SearchAttributes {
	attrValBytes1, _ := json.Marshal("another string")
	attrValBytes2, _ := json.Marshal(123)
	binaryChecksums, _ := json.Marshal([]string{"binary-v1", "binary-v2"})
	upsertSearchAttr := &types.SearchAttributes{
		IndexedFields: map[string][]byte{
			definition.CustomStringField: attrValBytes1,
			definition.CustomIntField:    attrValBytes2,
			definition.BinaryChecksums:   binaryChecksums,
		},
	}
	return upsertSearchAttr
}

func (s *elasticsearchIntegrationSuite) TestUpsertWorkflowExecution_InvalidKey() {
	id := "es-integration-upsert-workflow-failed-test"
	wt := "es-integration-upsert-workflow-failed-test-type"
	tl := "es-integration-upsert-workflow-failed-test-tasklist"
	identity := "worker1"

	workflowType := &types.WorkflowType{}
	workflowType.Name = common.StringPtr(wt)

	taskList := &types.TaskList{}
	taskList.Name = common.StringPtr(tl)

	request := &types.StartWorkflowExecutionRequest{
		RequestID:                           common.StringPtr(uuid.New()),
		Domain:                              s.domainName,
		WorkflowID:                          id,
		WorkflowType:                        workflowType,
		TaskList:                            taskList,
		Input:                               nil,
		ExecutionStartToCloseTimeoutSeconds: common.Int32Ptr(100),
		TaskStartToCloseTimeoutSeconds:      common.Int32Ptr(1),
		Identity:                            identity,
	}

	we, err0 := s.engine.StartWorkflowExecution(createContext(), request)
	s.Nil(err0)

	s.Logger.Info("StartWorkflowExecution", tag.WorkflowRunID(we.RunID))

	dtHandler := func(execution *types.WorkflowExecution, wt *types.WorkflowType,
		previousStartedEventID, startedEventID int64, history *types.History) ([]byte, []*types.Decision, error) {

		upsertDecision := &types.Decision{
			DecisionType: types.DecisionTypeUpsertWorkflowSearchAttributes.Ptr(),
			UpsertWorkflowSearchAttributesDecisionAttributes: &types.UpsertWorkflowSearchAttributesDecisionAttributes{
				SearchAttributes: &types.SearchAttributes{
					IndexedFields: map[string][]byte{
						"INVALIDKEY": []byte(`1`),
					},
				},
			}}
		return nil, []*types.Decision{upsertDecision}, nil
	}

	poller := &TaskPoller{
		Engine:          s.engine,
		Domain:          s.domainName,
		TaskList:        taskList,
		StickyTaskList:  taskList,
		Identity:        identity,
		DecisionHandler: dtHandler,
		Logger:          s.Logger,
		T:               s.T(),
	}

	_, err := poller.PollAndProcessDecisionTask(false, false)
	s.Nil(err)

	historyResponse, err := s.engine.GetWorkflowExecutionHistory(createContext(), &types.GetWorkflowExecutionHistoryRequest{
		Domain: s.domainName,
		Execution: &types.WorkflowExecution{
			WorkflowID: id,
			RunID:      we.RunID,
		},
	})
	s.Nil(err)
	history := historyResponse.History
	decisionFailedEvent := history.GetEvents()[3]
	s.Equal(types.EventTypeDecisionTaskFailed, decisionFailedEvent.GetEventType())
	failedDecisionAttr := decisionFailedEvent.DecisionTaskFailedEventAttributes
	s.Equal(types.DecisionTaskFailedCauseBadSearchAttributes, failedDecisionAttr.GetCause())
	s.True(len(failedDecisionAttr.GetDetails()) > 0)
}

func (s *elasticsearchIntegrationSuite) putIndexSettings(indexName string, maxResultWindowSize int) {
	err := s.esClient.PutMaxResultWindow(indexName, maxResultWindowSize)
	s.Require().NoError(err)
	s.verifyMaxResultWindowSize(indexName, maxResultWindowSize)
}

func (s *elasticsearchIntegrationSuite) verifyMaxResultWindowSize(indexName string, targetSize int) {
	for i := 0; i < numOfRetry; i++ {
		currentWindow, err := s.esClient.GetMaxResultWindow(indexName)
		s.Require().NoError(err)
		if currentWindow == strconv.Itoa(targetSize) {
			return
		}
		time.Sleep(waitTimeInMs * time.Millisecond)
	}
	s.FailNow(fmt.Sprintf("ES max result window size hasn't reach target size within %v.", (numOfRetry*waitTimeInMs)*time.Millisecond))
}
