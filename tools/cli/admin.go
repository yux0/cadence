// Copyright (c) 2017-2020 Uber Technologies Inc.
// Portions of the Software are attributed to Copyright (c) 2020 Temporal Technologies Inc.
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

package cli

import (
	"strings"
	"time"

	"github.com/urfave/cli"

	"github.com/uber/cadence/common/reconciliation/invariant"
	"github.com/uber/cadence/service/worker/scanner/executions"
)

func newAdminWorkflowCommands() []cli.Command {
	return []cli.Command{
		{
			Name:    "show",
			Aliases: []string{"show"},
			Usage:   "show workflow history from database",
			Flags: append(getDBFlags(),
				// v2 history events
				cli.StringFlag{
					Name:  FlagTreeID,
					Usage: "TreeID",
				},
				cli.StringFlag{
					Name:  FlagBranchID,
					Usage: "BranchID",
				},
				cli.StringFlag{
					Name:  FlagOutputFilenameWithAlias,
					Usage: "output file",
				},
				// support mysql query
				cli.IntFlag{
					Name:  FlagShardIDWithAlias,
					Usage: "ShardID",
				}),
			Action: func(c *cli.Context) {
				AdminShowWorkflow(c)
			},
		},
		{
			Name:    "describe",
			Aliases: []string{"desc"},
			Usage:   "Describe internal information of workflow execution",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagWorkflowIDWithAlias,
					Usage: "WorkflowID",
				},
				cli.StringFlag{
					Name:  FlagRunIDWithAlias,
					Usage: "RunID",
				},
			},
			Action: func(c *cli.Context) {
				AdminDescribeWorkflow(c)
			},
		},
		{
			Name:    "refresh-tasks",
			Aliases: []string{"rt"},
			Usage:   "Refreshes all the tasks of a workflow",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagWorkflowIDWithAlias,
					Usage: "WorkflowID",
				},
				cli.StringFlag{
					Name:  FlagRunIDWithAlias,
					Usage: "RunID",
				},
			},
			Action: func(c *cli.Context) {
				AdminRefreshWorkflowTasks(c)
			},
		},
		{
			Name:    "delete",
			Aliases: []string{"del"},
			Usage:   "Delete current workflow execution and the mutableState record",
			Flags: append(getDBFlags(),
				cli.StringFlag{
					Name:  FlagWorkflowIDWithAlias,
					Usage: "WorkflowID",
				},
				cli.StringFlag{
					Name:  FlagRunIDWithAlias,
					Usage: "RunID",
				},
				cli.BoolFlag{
					Name:  FlagSkipErrorModeWithAlias,
					Usage: "skip errors when deleting history",
				}),
			Action: func(c *cli.Context) {
				AdminDeleteWorkflow(c)
			},
		},
	}
}

func newAdminShardManagementCommands() []cli.Command {
	return []cli.Command{
		{
			Name:    "describe",
			Aliases: []string{"d"},
			Usage:   "Describe shard by Id",
			Flags: append(
				getDBFlags(),
				cli.IntFlag{
					Name:  FlagShardID,
					Usage: "The Id of the shard to describe",
				},
			),
			Action: func(c *cli.Context) {
				AdminDescribeShard(c)
			},
		},
		{
			Name:    "closeShard",
			Aliases: []string{"clsh"},
			Usage:   "close a shard given a shard id",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  FlagShardID,
					Usage: "ShardID for the cadence cluster to manage",
				},
			},
			Action: func(c *cli.Context) {
				AdminCloseShard(c)
			},
		},
		{
			Name:    "removeTask",
			Aliases: []string{"rmtk"},
			Usage:   "remove a task based on shardID, task type, taskID, and task visibility timestamp",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  FlagShardID,
					Usage: "shardID",
				},
				cli.Int64Flag{
					Name:  FlagTaskID,
					Usage: "taskID",
				},
				cli.IntFlag{
					Name:  FlagTaskType,
					Usage: "task type: 2 (transfer task), 3 (timer task) or 4 (replication task)",
				},
				cli.Int64Flag{
					Name:  FlagTaskVisibilityTimestamp,
					Usage: "task visibility timestamp in nano (required for removing timer task)",
				},
			},
			Action: func(c *cli.Context) {
				AdminRemoveTask(c)
			},
		},
		{
			Name:  "timers",
			Usage: "get scheduled timers for a given time range",
			Flags: append(getDBFlags(),
				cli.IntFlag{
					Name:  FlagShardID,
					Usage: "shardID",
				},
				cli.IntFlag{
					Name:  FlagPageSize,
					Usage: "page size used to query db executions table",
					Value: 500,
				},
				cli.IntFlag{
					Name:  FlagRPS,
					Usage: "target rps of database queries",
					Value: 100,
				},
				cli.StringFlag{
					Name:  FlagStartDate,
					Usage: "start date",
					Value: time.Now().UTC().Format(time.RFC3339),
				},
				cli.StringFlag{
					Name:  FlagEndDate,
					Usage: "end date",
					Value: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
				},
				cli.StringFlag{
					Name:  FlagDomainID,
					Usage: "filter tasks by DomainID",
				},
				cli.IntSliceFlag{
					Name: FlagTimerType,
					Usage: "timer types: 0 - DecisionTimeoutTask, 1 - TaskTypeActivityTimeout, " +
						"2 - TaskTypeUserTimer, 3 - TaskTypeWorkflowTimeout, 4 - TaskTypeDeleteHistoryEvent, " +
						"5 - TaskTypeActivityRetryTimer, 6 - TaskTypeWorkflowBackoffTimer",
					Value: &cli.IntSlice{-1},
				},
				cli.BoolFlag{
					Name:  FlagPrintJSON,
					Usage: "print raw json data instead of histogram",
				},

				cli.BoolFlag{
					Name:  FlagSkipErrorMode,
					Usage: "skip errors",
				},
				cli.StringFlag{
					Name:  FlagInputFile,
					Usage: "file to use, will not connect to persistence",
				},
				cli.StringFlag{
					Name:  FlagDateFormat,
					Usage: "create buckets using time format. Use Go reference time: Mon Jan 2 15:04:05 MST 2006. If set, --" + FlagBucketSize + " is ignored",
				},
				cli.StringFlag{
					Name:  FlagBucketSize,
					Value: "hour",
					Usage: "group timers by time bucket. Available: day, hour, minute, second",
				},
				cli.IntFlag{
					Name:  FlagShardMultiplier,
					Usage: "multiply timer counters for histogram",
					Value: 16384,
				},
			),
			Action: func(c *cli.Context) {
				AdminTimers(c)
			},
		},
	}
}

func newAdminHistoryHostCommands() []cli.Command {
	return []cli.Command{
		{
			Name:    "describe",
			Aliases: []string{"desc"},
			Usage:   "Describe internal information of history host",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagWorkflowIDWithAlias,
					Usage: "WorkflowID",
				},
				cli.StringFlag{
					Name:  FlagHistoryAddressWithAlias,
					Usage: "History Host address(IP:PORT)",
				},
				cli.IntFlag{
					Name:  FlagShardIDWithAlias,
					Usage: "ShardID",
				},
				cli.BoolFlag{
					Name:  FlagPrintFullyDetailWithAlias,
					Usage: "Print fully detail",
				},
			},
			Action: func(c *cli.Context) {
				AdminDescribeHistoryHost(c)
			},
		},
		{
			Name:    "getshard",
			Aliases: []string{"gsh"},
			Usage:   "Get shardID for a workflowID",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagWorkflowIDWithAlias,
					Usage: "WorkflowID",
				},
				cli.IntFlag{
					Name:  FlagNumberOfShards,
					Usage: "NumberOfShards for the cadence cluster(see config for numHistoryShards)",
				},
			},
			Action: func(c *cli.Context) {
				AdminGetShardID(c)
			},
		},
	}
}

func newAdminDomainCommands() []cli.Command {
	return []cli.Command{
		{
			Name:    "register",
			Aliases: []string{"re"},
			Usage:   "Register workflow domain",
			Flags:   adminRegisterDomainFlags,
			Action: func(c *cli.Context) {
				newDomainCLI(c, true).RegisterDomain(c)
			},
		},
		{
			Name:    "update",
			Aliases: []string{"up", "u"},
			Usage:   "Update existing workflow domain",
			Flags:   adminUpdateDomainFlags,
			Action: func(c *cli.Context) {
				newDomainCLI(c, true).UpdateDomain(c)
			},
		},
		{
			Name:    "describe",
			Aliases: []string{"desc"},
			Usage:   "Describe existing workflow domain",
			Flags:   adminDescribeDomainFlags,
			Action: func(c *cli.Context) {
				newDomainCLI(c, true).DescribeDomain(c)
			},
		},
		{
			Name:    "getdomainidorname",
			Aliases: []string{"getdn"},
			Usage:   "Get domainID or domainName",
			Flags: append(getDBFlags(),
				cli.StringFlag{
					Name:  FlagDomain,
					Usage: "DomainName",
				},
				cli.StringFlag{
					Name:  FlagDomainID,
					Usage: "Domain ID(uuid)",
				}),
			Action: func(c *cli.Context) {
				AdminGetDomainIDOrName(c)
			},
		},
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "List all domains in the cluster",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  FlagPageSizeWithAlias,
					Value: 10,
					Usage: "Result page size",
				},
				cli.BoolFlag{
					Name:  FlagAllWithAlias,
					Usage: "List all domains, by default only domains in REGISTERED status are listed",
				},
				cli.BoolFlag{
					Name:  FlagPrintFullyDetailWithAlias,
					Usage: "Print full domain detail",
				},
			},
			Action: func(c *cli.Context) {
				newDomainCLI(c, false).ListDomains(c)
			},
		},
	}
}

func newAdminKafkaCommands() []cli.Command {
	return []cli.Command{
		{
			Name:    "parse",
			Aliases: []string{"par"},
			Usage:   "Parse replication tasks from kafka messages",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagInputFileWithAlias,
					Usage: "Input file to use, if not present assumes piping",
				},
				cli.StringFlag{
					Name:  FlagWorkflowIDWithAlias,
					Usage: "WorkflowID, if not provided then no filters by WorkflowID are applied",
				},
				cli.StringFlag{
					Name:  FlagRunIDWithAlias,
					Usage: "RunID, if not provided then no filters by RunID are applied",
				},
				cli.StringFlag{
					Name:  FlagOutputFilenameWithAlias,
					Usage: "Output file to write to, if not provided output is written to stdout",
				},
				cli.BoolFlag{
					Name:  FlagSkipErrorModeWithAlias,
					Usage: "Skip errors in parsing messages",
				},
				cli.BoolFlag{
					Name:  FlagHeadersModeWithAlias,
					Usage: "Output headers of messages in format: DomainID, WorkflowID, RunID, FirstEventID, NextEventID",
				},
				cli.IntFlag{
					Name:  FlagMessageTypeWithAlias,
					Usage: "Kafka message type (0: replicationTasks; 1: visibility)",
					Value: 0,
				},
			},
			Action: func(c *cli.Context) {
				AdminKafkaParse(c)
			},
		},
		{
			Name:    "rereplicate",
			Aliases: []string{"rrp"},
			Usage:   "Rereplicate replication tasks to target topic from history tables",
			Flags: append(getDBFlags(),
				cli.StringFlag{
					Name:  FlagSourceCluster,
					Usage: "Name of source cluster to resend the replication task",
				},
				cli.IntFlag{
					Name:  FlagNumberOfShards,
					Usage: "NumberOfShards is required to calculate shardID. (see server config for numHistoryShards)",
				},
				// for one workflow
				cli.Int64Flag{
					Name:  FlagMaxEventID,
					Usage: "MaxEventID Optional, default to all events",
				},
				cli.StringFlag{
					Name:  FlagWorkflowIDWithAlias,
					Usage: "WorkflowID",
				},
				cli.StringFlag{
					Name:  FlagRunIDWithAlias,
					Usage: "RunID",
				},
				cli.StringFlag{
					Name:  FlagDomainID,
					Usage: "DomainID",
				},
				cli.StringFlag{
					Name:  FlagEndEventVersion,
					Usage: "Workflow end event version",
				}),
			Action: func(c *cli.Context) {
				AdminRereplicate(c)
			},
		},
	}
}

func newAdminElasticSearchCommands() []cli.Command {
	return []cli.Command{
		{
			Name:    "catIndex",
			Aliases: []string{"cind"},
			Usage:   "Cat Indices on ElasticSearch",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagURL,
					Usage: "URL of ElasticSearch cluster",
				},
				cli.StringFlag{
					Name:  FlagMuttleyDestinationWithAlias,
					Usage: "Optional muttely destination to ElasticSearch cluster",
				},
			},
			Action: func(c *cli.Context) {
				AdminCatIndices(c)
			},
		},
		{
			Name:    "index",
			Aliases: []string{"ind"},
			Usage:   "Index docs on ElasticSearch",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagURL,
					Usage: "URL of ElasticSearch cluster",
				},
				cli.StringFlag{
					Name:  FlagMuttleyDestinationWithAlias,
					Usage: "Optional muttely destination to ElasticSearch cluster",
				},
				cli.StringFlag{
					Name:  FlagIndex,
					Usage: "ElasticSearch target index",
				},
				cli.StringFlag{
					Name:  FlagInputFileWithAlias,
					Usage: "Input file of indexer.Message in json format, separated by newline",
				},
				cli.IntFlag{
					Name:  FlagBatchSizeWithAlias,
					Usage: "Optional batch size of actions for bulk operations",
					Value: 1000,
				},
			},
			Action: func(c *cli.Context) {
				AdminIndex(c)
			},
		},
		{
			Name:    "delete",
			Aliases: []string{"del"},
			Usage:   "Delete docs on ElasticSearch",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagURL,
					Usage: "URL of ElasticSearch cluster",
				},
				cli.StringFlag{
					Name:  FlagMuttleyDestinationWithAlias,
					Usage: "Optional muttely destination to ElasticSearch cluster",
				},
				cli.StringFlag{
					Name:  FlagIndex,
					Usage: "ElasticSearch target index",
				},
				cli.StringFlag{
					Name: FlagInputFileWithAlias,
					Usage: "Input file name. Redirect cadence wf list result (with tale format) to a file and use as delete input. " +
						"First line should be table header like WORKFLOW TYPE | WORKFLOW ID | RUN ID | ...",
				},
				cli.IntFlag{
					Name:  FlagBatchSizeWithAlias,
					Usage: "Optional batch size of actions for bulk operations",
					Value: 1000,
				},
				cli.IntFlag{
					Name:  FlagRPS,
					Usage: "Optional batch request rate per second",
					Value: 30,
				},
			},
			Action: func(c *cli.Context) {
				AdminDelete(c)
			},
		},
		{
			Name:    "report",
			Aliases: []string{"rep"},
			Usage:   "Generate Report by Aggregation functions on ElasticSearch",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagURL,
					Usage: "URL of ElasticSearch cluster",
				},
				cli.StringFlag{
					Name:  FlagIndex,
					Usage: "ElasticSearch target index",
				},
				cli.StringFlag{
					Name:  FlagListQuery,
					Usage: "SQL query of the report",
				},
				cli.StringFlag{
					Name:  FlagOutputFormat,
					Usage: "Additional output format (html or csv)",
				},
				cli.StringFlag{
					Name:  FlagOutputFilename,
					Usage: "Additional output filename with path",
				},
			},
			Action: func(c *cli.Context) {
				GenerateReport(c)
			},
		},
	}
}

func newAdminTaskListCommands() []cli.Command {
	return []cli.Command{
		{
			Name:    "describe",
			Aliases: []string{"desc"},
			Usage:   "Describe pollers and status information of tasklist",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagTaskListWithAlias,
					Usage: "TaskList description",
				},
				cli.StringFlag{
					Name:  FlagTaskListTypeWithAlias,
					Value: "decision",
					Usage: "Optional TaskList type [decision|activity]",
				},
			},
			Action: func(c *cli.Context) {
				AdminDescribeTaskList(c)
			},
		},
	}
}

func newAdminClusterCommands() []cli.Command {
	return []cli.Command{
		{
			Name:    "add-search-attr",
			Aliases: []string{"asa"},
			Usage:   "whitelist search attribute",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagSearchAttributesKey,
					Usage: "Search Attribute key to be whitelisted",
				},
				cli.IntFlag{
					Name:  FlagSearchAttributesType,
					Value: -1,
					Usage: "Search Attribute value type. [0:String, 1:Keyword, 2:Int, 3:Double, 4:Bool, 5:Datetime]",
				},
				cli.StringFlag{
					Name:  FlagSecurityTokenWithAlias,
					Usage: "Optional token for security check",
				},
			},
			Action: func(c *cli.Context) {
				AdminAddSearchAttribute(c)
			},
		},
		{
			Name:    "describe",
			Aliases: []string{"d"},
			Usage:   "Describe cluster information",
			Action: func(c *cli.Context) {
				AdminDescribeCluster(c)
			},
		},
		{
			Name:    "failover",
			Aliases: []string{"fo"},
			Usage:   "Failover domains with domain data IsManagedByCadence=true to target cluster",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagActiveClusterNameWithAlias,
					Usage: "Target active cluster name",
				},
			},
			Action: func(c *cli.Context) {
				newDomainCLI(c, false).FailoverDomains(c)
			},
		},
	}
}

func newAdminDLQCommands() []cli.Command {
	return []cli.Command{
		{
			Name:    "read",
			Aliases: []string{"r"},
			Usage:   "Read DLQ Messages",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagDLQTypeWithAlias,
					Usage: "Type of DLQ to manage. (Options: domain, history)",
				},
				cli.StringFlag{
					Name:  FlagSourceCluster,
					Usage: "The cluster where the task is generated",
				},
				cli.IntFlag{
					Name:  FlagShardIDWithAlias,
					Usage: "ShardID",
				},
				cli.IntFlag{
					Name:  FlagMaxMessageCountWithAlias,
					Usage: "Max message size to fetch",
				},
				cli.IntFlag{
					Name:  FlagLastMessageIDWithAlias,
					Usage: "The upper boundary of the read message",
				},
				cli.StringFlag{
					Name:  FlagOutputFilenameWithAlias,
					Usage: "Output file to write to, if not provided output is written to stdout",
				},
			},
			Action: func(c *cli.Context) {
				AdminGetDLQMessages(c)
			},
		},
		{
			Name:    "purge",
			Aliases: []string{"p"},
			Usage:   "Delete DLQ messages with equal or smaller ids than the provided task id",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagDLQTypeWithAlias,
					Usage: "Type of DLQ to manage. (Options: domain, history)",
				},
				cli.StringFlag{
					Name:  FlagSourceCluster,
					Usage: "The cluster where the task is generated",
				},
				cli.IntFlag{
					Name:  FlagLowerShardBound,
					Usage: "lower bound of shard to merge (inclusive)",
				},
				cli.IntFlag{
					Name:  FlagUpperShardBound,
					Usage: "upper bound of shard to merge (inclusive)",
				},
				cli.IntFlag{
					Name:  FlagLastMessageIDWithAlias,
					Usage: "The upper boundary of the read message",
				},
			},
			Action: func(c *cli.Context) {
				AdminPurgeDLQMessages(c)
			},
		},
		{
			Name:    "merge",
			Aliases: []string{"m"},
			Usage:   "Merge DLQ messages with equal or smaller ids than the provided task id",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  FlagDLQTypeWithAlias,
					Usage: "Type of DLQ to manage. (Options: domain, history)",
				},
				cli.StringFlag{
					Name:  FlagSourceCluster,
					Usage: "The cluster where the task is generated",
				},
				cli.IntFlag{
					Name:  FlagLowerShardBound,
					Usage: "lower bound of shard to merge (inclusive)",
				},
				cli.IntFlag{
					Name:  FlagUpperShardBound,
					Usage: "upper bound of shard to merge (inclusive)",
				},
				cli.IntFlag{
					Name:  FlagLastMessageIDWithAlias,
					Usage: "The upper boundary of the read message",
				},
			},
			Action: func(c *cli.Context) {
				AdminMergeDLQMessages(c)
			},
		},
	}
}

func newAdminQueueCommands() []cli.Command {
	return []cli.Command{
		{
			Name:  "reset",
			Usage: "reset processing queue states for transfer or timer queue processor",
			Flags: getQueueCommandFlags(),
			Action: func(c *cli.Context) {
				AdminResetQueue(c)
			},
		},
		{
			Name:    "describe",
			Aliases: []string{"desc"},
			Usage:   "describe processing queue states for transfer or timer queue processor",
			Flags:   getQueueCommandFlags(),
			Action: func(c *cli.Context) {
				AdminDescribeQueue(c)
			},
		},
	}
}

func newDBCommands() []cli.Command {
	var collections cli.StringSlice = invariant.CollectionStrings()

	scanFlag := cli.StringFlag{
		Name:     FlagScanType,
		Usage:    "Scan type to use: " + strings.Join(executions.ScanTypeStrings(), ", "),
		Required: true,
	}

	collectionsFlag := cli.StringSliceFlag{
		Name:  FlagInvariantCollection,
		Usage: "Scan collection type to use: " + strings.Join(collections, ", "),
		Value: &collections,
	}

	return []cli.Command{
		{
			Name:  "scan",
			Usage: "scan executions in database and detect corruptions",
			Flags: append(getDBFlags(),
				cli.IntFlag{
					Name:     FlagNumberOfShards,
					Usage:    "NumberOfShards for the cadence cluster (see config for numHistoryShards)",
					Required: true,
				},
				scanFlag,
				collectionsFlag,
				cli.StringFlag{
					Name:  FlagInputFileWithAlias,
					Usage: "Input file of executions to scan in JSON format {\"DomainID\":\"x\",\"WorkflowID\":\"x\",\"RunID\":\"x\"} separated by a newline",
				},
			),

			Action: func(c *cli.Context) {
				AdminDBScan(c)
			},
		},
		{
			Name:  "clean",
			Usage: "clean up corrupted workflows",
			Flags: append(getDBFlags(),
				scanFlag,
				collectionsFlag,
				cli.StringFlag{
					Name:  FlagInputFileWithAlias,
					Usage: "Input file of execution to clean in JSON format. Use `scan` command to generate list of executions.",
				},
			),
			Action: func(c *cli.Context) {
				AdminDBClean(c)
			},
		},
	}
}

// TODO need to support other database: https://github.com/uber/cadence/issues/2777
func getDBFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  FlagDBAddress,
			Value: "127.0.0.1",
			Usage: "persistence address (right now only cassandra is fully supported)",
		},
		cli.IntFlag{
			Name:  FlagDBPort,
			Value: 9042,
			Usage: "persistence port",
		},
		cli.StringFlag{
			Name:  FlagUsername,
			Usage: "cassandra username",
		},
		cli.StringFlag{
			Name:  FlagPassword,
			Usage: "cassandra password",
		},
		cli.StringFlag{
			Name:  FlagKeyspace,
			Value: "cadence",
			Usage: "cassandra keyspace",
		},
		cli.BoolFlag{
			Name:  FlagEnableTLS,
			Usage: "enable TLS over cassandra connection",
		},
		cli.StringFlag{
			Name:  FlagTLSCertPath,
			Usage: "cassandra tls client cert path (tls must be enabled)",
		},
		cli.StringFlag{
			Name:  FlagTLSKeyPath,
			Usage: "cassandra tls client key path (tls must be enabled)",
		},
		cli.StringFlag{
			Name:  FlagTLSCaPath,
			Usage: "cassandra tls client ca path (tls must be enabled)",
		},
		cli.BoolFlag{
			Name:  FlagTLSEnableHostVerification,
			Usage: "cassandra tls verify hostname and server cert (tls must be enabled)",
		},
	}
}

func getQueueCommandFlags() []cli.Flag {
	return []cli.Flag{
		cli.IntFlag{
			Name:  FlagShardIDWithAlias,
			Usage: "shardID",
		},
		cli.StringFlag{
			Name:  FlagCluster,
			Usage: "cluster the task processor is responsible for",
		},
		cli.IntFlag{
			Name:  FlagQueueType,
			Usage: "queue type: 2 (transfer queue) or 3 (timer queue)",
		},
	}
}
