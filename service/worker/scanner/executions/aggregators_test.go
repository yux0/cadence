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

package executions

import (
	c "github.com/uber/cadence/common"
	"github.com/uber/cadence/service/worker/scanner/executions/common"
)

func (s *workflowsSuite) TestShardScanResultAggregator() {
	agg := newShardScanResultAggregator([]int{1, 2, 3}, 1, 3)
	expected := &shardScanResultAggregator{
		minShard: 1,
		maxShard: 3,
		reports:  map[int]common.ShardScanReport{},
		status: map[int]ShardStatus{
			1: ShardStatusRunning,
			2: ShardStatusRunning,
			3: ShardStatusRunning,
		},
		aggregation: AggregateScanReportResult{
			CorruptionByType: make(map[common.InvariantType]int64),
		},
		corruptionKeys:          make(map[int]common.Keys),
		controlFlowFailureCount: 0,
		successCount:            0,
	}
	s.Equal(expected, agg)
	report, err := agg.getReport(1)
	s.Nil(report)
	s.Equal("shard 1 has not finished yet, check back later for report", err.Error())
	report, err = agg.getReport(5)
	s.Nil(report)
	s.Equal("shard 5 is not included in shards which will be processed", err.Error())
	firstReport := common.ShardScanReport{
		ShardID: 1,
		Stats: common.ShardScanStats{
			ExecutionsCount:  10,
			CorruptedCount:   3,
			CheckFailedCount: 1,
			CorruptionByType: map[common.InvariantType]int64{
				common.HistoryExistsInvariantType:        2,
				common.OpenCurrentExecutionInvariantType: 1,
			},
			CorruptedOpenExecutionCount: 1,
		},
		Result: common.ShardScanResult{
			ShardScanKeys: &common.ShardScanKeys{
				Corrupt: &common.Keys{
					UUID: "test_uuid",
				},
			},
		},
	}
	agg.addReport(firstReport)
	expected.status[1] = ShardStatusSuccess
	expected.reports[1] = firstReport
	expected.aggregation.ExecutionsCount = 10
	expected.aggregation.CorruptedCount = 3
	expected.aggregation.CheckFailedCount = 1
	expected.aggregation.CorruptionByType = map[common.InvariantType]int64{
		common.HistoryExistsInvariantType:        2,
		common.OpenCurrentExecutionInvariantType: 1,
	}
	expected.aggregation.CorruptedOpenExecutionCount = 1
	expected.corruptionKeys = map[int]common.Keys{
		1: {
			UUID: "test_uuid",
		},
	}
	expected.successCount = 1
	s.Equal(expected, agg)
	agg.addReport(firstReport)
	s.Equal(expected, agg)
	report, err = agg.getReport(1)
	s.NoError(err)
	s.Equal(firstReport, *report)
	secondReport := common.ShardScanReport{
		ShardID: 2,
		Stats: common.ShardScanStats{
			ExecutionsCount:  10,
			CorruptedCount:   3,
			CheckFailedCount: 1,
			CorruptionByType: map[common.InvariantType]int64{
				common.HistoryExistsInvariantType:        2,
				common.OpenCurrentExecutionInvariantType: 1,
			},
			CorruptedOpenExecutionCount: 1,
		},
		Result: common.ShardScanResult{
			ControlFlowFailure: &common.ControlFlowFailure{},
		},
	}
	agg.addReport(secondReport)
	expected.status[2] = ShardStatusControlFlowFailure
	expected.reports[2] = secondReport
	expected.controlFlowFailureCount = 1
	s.Equal(expected, agg)
	shardStatus, err := agg.getStatusResult(PaginatedShardQueryRequest{
		StartingShardID: c.IntPtr(1),
		LimitShards:     c.IntPtr(2),
	})
	s.NoError(err)
	s.Equal(&ShardStatusQueryResult{
		Result: map[int]ShardStatus{
			1: ShardStatusSuccess,
			2: ShardStatusControlFlowFailure,
		},
		ShardQueryPaginationToken: ShardQueryPaginationToken{
			NextShardID: c.IntPtr(3),
			IsDone:      false,
		},
	}, shardStatus)
	corruptedKeys, err := agg.getCorruptionKeys(PaginatedShardQueryRequest{
		StartingShardID: c.IntPtr(1),
		LimitShards:     c.IntPtr(3),
	})
	s.NoError(err)
	s.Equal(&ShardCorruptKeysQueryResult{
		Result: map[int]common.Keys{
			1: {
				UUID: "test_uuid",
			},
		},
		ShardQueryPaginationToken: ShardQueryPaginationToken{
			NextShardID: nil,
			IsDone:      true,
		},
	}, corruptedKeys)
}

func (s *workflowsSuite) TestShardFixResultAggregator() {
	agg := newShardFixResultAggregator([]CorruptedKeysEntry{{ShardID: 1}, {ShardID: 2}, {ShardID: 3}}, 1, 3)
	expected := &shardFixResultAggregator{
		minShard: 1,
		maxShard: 3,
		reports:  map[int]common.ShardFixReport{},
		status: map[int]ShardStatus{
			1: ShardStatusRunning,
			2: ShardStatusRunning,
			3: ShardStatusRunning,
		},
		aggregation: AggregateFixReportResult{},
	}
	s.Equal(expected, agg)
	report, err := agg.getReport(1)
	s.Nil(report)
	s.Equal("shard 1 has not finished yet, check back later for report", err.Error())
	report, err = agg.getReport(5)
	s.Nil(report)
	s.Equal("shard 5 is not included in shards which will be processed", err.Error())
	firstReport := common.ShardFixReport{
		ShardID: 1,
		Stats: common.ShardFixStats{
			ExecutionCount: 10,
			FixedCount:     3,
			FailedCount:    1,
		},
		Result: common.ShardFixResult{
			ShardFixKeys: &common.ShardFixKeys{
				Fixed: &common.Keys{
					UUID: "test_uuid",
				},
			},
		},
	}
	agg.addReport(firstReport)
	expected.status[1] = ShardStatusSuccess
	expected.reports[1] = firstReport
	expected.aggregation.ExecutionCount = 10
	expected.aggregation.FixedCount = 3
	expected.aggregation.FailedCount = 1
	s.Equal(expected, agg)
	agg.addReport(firstReport)
	s.Equal(expected, agg)
	report, err = agg.getReport(1)
	s.NoError(err)
	s.Equal(firstReport, *report)
	secondReport := common.ShardFixReport{
		ShardID: 2,
		Stats: common.ShardFixStats{
			ExecutionCount: 10,
			FixedCount:     3,
			FailedCount:    1,
		},
		Result: common.ShardFixResult{
			ControlFlowFailure: &common.ControlFlowFailure{},
		},
	}
	agg.addReport(secondReport)
	expected.status[2] = ShardStatusControlFlowFailure
	expected.reports[2] = secondReport
	s.Equal(expected, agg)
	shardStatus, err := agg.getStatusResult(PaginatedShardQueryRequest{
		StartingShardID: c.IntPtr(1),
		LimitShards:     c.IntPtr(2),
	})
	s.NoError(err)
	s.Equal(&ShardStatusQueryResult{
		Result: map[int]ShardStatus{
			1: ShardStatusSuccess,
			2: ShardStatusControlFlowFailure,
		},
		ShardQueryPaginationToken: ShardQueryPaginationToken{
			NextShardID: c.IntPtr(3),
			IsDone:      false,
		},
	}, shardStatus)
}

func (s *workflowsSuite) TestGetStatusResult() {
	testCases := []struct {
		minShardID     int
		maxShardID     int
		req            PaginatedShardQueryRequest
		status         ShardStatusResult
		expectedResult *ShardStatusQueryResult
		expectedError  bool
	}{
		{
			minShardID: 0,
			maxShardID: 5,
			req: PaginatedShardQueryRequest{
				StartingShardID: c.IntPtr(6),
			},
			expectedResult: nil,
			expectedError:  true,
		},
		{
			minShardID: 0,
			maxShardID: 5,
			req: PaginatedShardQueryRequest{
				StartingShardID: c.IntPtr(0),
				LimitShards:     c.IntPtr(10),
			},
			status: map[int]ShardStatus{
				1: ShardStatusRunning,
				2: ShardStatusRunning,
				3: ShardStatusSuccess,
				4: ShardStatusSuccess,
				5: ShardStatusControlFlowFailure,
			},
			expectedResult: &ShardStatusQueryResult{
				Result: map[int]ShardStatus{
					1: ShardStatusRunning,
					2: ShardStatusRunning,
					3: ShardStatusSuccess,
					4: ShardStatusSuccess,
					5: ShardStatusControlFlowFailure,
				},
				ShardQueryPaginationToken: ShardQueryPaginationToken{
					NextShardID: nil,
					IsDone:      true,
				},
			},
			expectedError: false,
		},
		{
			minShardID: 0,
			maxShardID: 5,
			req: PaginatedShardQueryRequest{
				StartingShardID: c.IntPtr(0),
				LimitShards:     c.IntPtr(2),
			},
			status: map[int]ShardStatus{
				1: ShardStatusRunning,
				2: ShardStatusRunning,
				3: ShardStatusSuccess,
				4: ShardStatusSuccess,
				5: ShardStatusControlFlowFailure,
			},
			expectedResult: &ShardStatusQueryResult{
				Result: map[int]ShardStatus{
					1: ShardStatusRunning,
					2: ShardStatusRunning,
				},
				ShardQueryPaginationToken: ShardQueryPaginationToken{
					NextShardID: c.IntPtr(3),
					IsDone:      false,
				},
			},
			expectedError: false,
		},
		{
			minShardID: 0,
			maxShardID: 5,
			req: PaginatedShardQueryRequest{
				StartingShardID: c.IntPtr(0),
				LimitShards:     c.IntPtr(3),
			},
			status: map[int]ShardStatus{
				1: ShardStatusRunning,
				2: ShardStatusRunning,
				4: ShardStatusSuccess,
				5: ShardStatusControlFlowFailure,
			},
			expectedResult: &ShardStatusQueryResult{
				Result: map[int]ShardStatus{
					1: ShardStatusRunning,
					2: ShardStatusRunning,
					4: ShardStatusSuccess,
				},
				ShardQueryPaginationToken: ShardQueryPaginationToken{
					NextShardID: c.IntPtr(5),
					IsDone:      false,
				},
			},
			expectedError: false,
		},
		{
			minShardID: 0,
			maxShardID: 5,
			req: PaginatedShardQueryRequest{
				StartingShardID: c.IntPtr(2),
				LimitShards:     c.IntPtr(3),
			},
			status: map[int]ShardStatus{
				1: ShardStatusRunning,
				2: ShardStatusRunning,
				4: ShardStatusSuccess,
				5: ShardStatusControlFlowFailure,
			},
			expectedResult: &ShardStatusQueryResult{
				Result: map[int]ShardStatus{
					2: ShardStatusRunning,
					4: ShardStatusSuccess,
					5: ShardStatusControlFlowFailure,
				},
				ShardQueryPaginationToken: ShardQueryPaginationToken{
					NextShardID: nil,
					IsDone:      true,
				},
			},
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		result, err := getStatusResult(tc.minShardID, tc.maxShardID, tc.req, tc.status)
		s.Equal(tc.expectedResult, result)
		if tc.expectedError {
			s.Error(err)
		} else {
			s.NoError(err)
		}
	}
}
