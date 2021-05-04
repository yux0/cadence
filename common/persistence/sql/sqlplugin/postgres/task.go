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

package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uber/cadence/common/persistence/sql/sqlplugin"
)

const (
	taskListCreatePart = `INTO task_lists(shard_id, domain_id, name, task_type, range_id, data, data_encoding) ` +
		`VALUES (:shard_id, :domain_id, :name, :task_type, :range_id, :data, :data_encoding)`

	// (default range ID: initialRangeID == 1)
	createTaskListQry = `INSERT ` + taskListCreatePart

	updateTaskListQry = `UPDATE task_lists SET
range_id = :range_id,
data = :data,
data_encoding = :data_encoding
WHERE
shard_id = :shard_id AND
domain_id = :domain_id AND
name = :name AND
task_type = :task_type
`

	listTaskListQry = `SELECT domain_id, range_id, name, task_type, data, data_encoding ` +
		`FROM task_lists ` +
		`WHERE shard_id = $1 AND domain_id > $2 AND name > $3 AND task_type > $4 ORDER BY domain_id,name,task_type LIMIT $5`

	getTaskListQry = `SELECT domain_id, range_id, name, task_type, data, data_encoding ` +
		`FROM task_lists ` +
		`WHERE shard_id = $1 AND domain_id = $2 AND name = $3 AND task_type = $4`

	deleteTaskListQry = `DELETE FROM task_lists WHERE shard_id=$1 AND domain_id=$2 AND name=$3 AND task_type=$4 AND range_id=$5`

	lockTaskListQry = `SELECT range_id FROM task_lists ` +
		`WHERE shard_id = $1 AND domain_id = $2 AND name = $3 AND task_type = $4 FOR UPDATE`

	getTaskMinMaxQry = `SELECT task_id, data, data_encoding ` +
		`FROM tasks ` +
		`WHERE domain_id = $1 AND task_list_name = $2 AND task_type = $3 AND task_id > $4 AND task_id <= $5 ` +
		` ORDER BY task_id LIMIT $6`

	getTaskMinQry = `SELECT task_id, data, data_encoding ` +
		`FROM tasks ` +
		`WHERE domain_id = $1 AND task_list_name = $2 AND task_type = $3 AND task_id > $4 ORDER BY task_id LIMIT $5`

	createTaskQry = `INSERT INTO ` +
		`tasks(domain_id, task_list_name, task_type, task_id, data, data_encoding) ` +
		`VALUES(:domain_id, :task_list_name, :task_type, :task_id, :data, :data_encoding)`

	deleteTaskQry = `DELETE FROM tasks ` +
		`WHERE domain_id = $1 AND task_list_name = $2 AND task_type = $3 AND task_id = $4`

	rangeDeleteTaskQry = `DELETE FROM tasks ` +
		`WHERE domain_id = $1 AND task_list_name = $2 AND task_type = $3 AND task_id IN (SELECT task_id FROM
		 tasks WHERE domain_id = $1 AND task_list_name = $2 AND task_type = $3 AND task_id <= $4 ` +
		`ORDER BY domain_id,task_list_name,task_type,task_id LIMIT $5 )`
)

// InsertIntoTasks inserts one or more rows into tasks table
func (pdb *db) InsertIntoTasks(ctx context.Context, rows []sqlplugin.TasksRow) (sql.Result, error) {
	return pdb.conn.NamedExecContext(ctx, createTaskQry, rows)
}

// SelectFromTasks reads one or more rows from tasks table
func (pdb *db) SelectFromTasks(ctx context.Context, filter *sqlplugin.TasksFilter) ([]sqlplugin.TasksRow, error) {
	var err error
	var rows []sqlplugin.TasksRow
	switch {
	case filter.MaxTaskID != nil:
		err = pdb.conn.SelectContext(ctx, &rows, getTaskMinMaxQry, filter.DomainID,
			filter.TaskListName, filter.TaskType, *filter.MinTaskID, *filter.MaxTaskID, *filter.PageSize)
	default:
		err = pdb.conn.SelectContext(ctx, &rows, getTaskMinQry, filter.DomainID,
			filter.TaskListName, filter.TaskType, *filter.MinTaskID, *filter.PageSize)
	}
	if err != nil {
		return nil, err
	}
	return rows, err
}

// DeleteFromTasks deletes one or more rows from tasks table
func (pdb *db) DeleteFromTasks(ctx context.Context, filter *sqlplugin.TasksFilter) (sql.Result, error) {
	if filter.TaskIDLessThanEquals != nil {
		if filter.Limit == nil || *filter.Limit == 0 {
			return nil, fmt.Errorf("missing limit parameter")
		}
		return pdb.conn.ExecContext(ctx, rangeDeleteTaskQry,
			filter.DomainID, filter.TaskListName, filter.TaskType, *filter.TaskIDLessThanEquals, *filter.Limit)
	}
	return pdb.conn.ExecContext(ctx, deleteTaskQry, filter.DomainID, filter.TaskListName, filter.TaskType, *filter.TaskID)
}

// InsertIntoTaskLists inserts one or more rows into task_lists table
func (pdb *db) InsertIntoTaskLists(ctx context.Context, row *sqlplugin.TaskListsRow) (sql.Result, error) {
	return pdb.conn.NamedExecContext(ctx, createTaskListQry, row)
}

// UpdateTaskLists updates a row in task_lists table
func (pdb *db) UpdateTaskLists(ctx context.Context, row *sqlplugin.TaskListsRow) (sql.Result, error) {
	return pdb.conn.NamedExecContext(ctx, updateTaskListQry, row)
}

// SelectFromTaskLists reads one or more rows from task_lists table
func (pdb *db) SelectFromTaskLists(ctx context.Context, filter *sqlplugin.TaskListsFilter) ([]sqlplugin.TaskListsRow, error) {
	switch {
	case filter.DomainID != nil && filter.Name != nil && filter.TaskType != nil:
		return pdb.selectFromTaskLists(ctx, filter)
	case filter.DomainIDGreaterThan != nil && filter.NameGreaterThan != nil && filter.TaskTypeGreaterThan != nil && filter.PageSize != nil:
		return pdb.rangeSelectFromTaskLists(ctx, filter)
	default:
		return nil, fmt.Errorf("invalid set of query filter params")
	}
}

func (pdb *db) selectFromTaskLists(ctx context.Context, filter *sqlplugin.TaskListsFilter) ([]sqlplugin.TaskListsRow, error) {
	var err error
	var row sqlplugin.TaskListsRow
	err = pdb.conn.GetContext(ctx, &row, getTaskListQry, filter.ShardID, *filter.DomainID, *filter.Name, *filter.TaskType)
	if err != nil {
		return nil, err
	}
	return []sqlplugin.TaskListsRow{row}, err
}

func (pdb *db) rangeSelectFromTaskLists(ctx context.Context, filter *sqlplugin.TaskListsFilter) ([]sqlplugin.TaskListsRow, error) {
	var err error
	var rows []sqlplugin.TaskListsRow
	err = pdb.conn.SelectContext(ctx, &rows, listTaskListQry,
		filter.ShardID, *filter.DomainIDGreaterThan, *filter.NameGreaterThan, *filter.TaskTypeGreaterThan, *filter.PageSize)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].ShardID = filter.ShardID
	}
	return rows, nil
}

// DeleteFromTaskLists deletes a row from task_lists table
func (pdb *db) DeleteFromTaskLists(ctx context.Context, filter *sqlplugin.TaskListsFilter) (sql.Result, error) {
	return pdb.conn.ExecContext(ctx, deleteTaskListQry, filter.ShardID, *filter.DomainID, *filter.Name, *filter.TaskType, *filter.RangeID)
}

// LockTaskLists locks a row in task_lists table
func (pdb *db) LockTaskLists(ctx context.Context, filter *sqlplugin.TaskListsFilter) (int64, error) {
	var rangeID int64
	err := pdb.conn.GetContext(ctx, &rangeID, lockTaskListQry, filter.ShardID, *filter.DomainID, *filter.Name, *filter.TaskType)
	return rangeID, err
}

// InsertIntoTasksWithTTL is not supported in Postgres
func (pdb *db) InsertIntoTasksWithTTL(_ context.Context, _ []sqlplugin.TasksRowWithTTL) (sql.Result, error) {
	return nil, sqlplugin.ErrTTLNotSupported
}

// InsertIntoTaskListsWithTTL is not supported in Postgres
func (pdb *db) InsertIntoTaskListsWithTTL(_ context.Context, _ *sqlplugin.TaskListsRowWithTTL) (sql.Result, error) {
	return nil, sqlplugin.ErrTTLNotSupported
}

// UpdateTaskListsWithTTL is not supported in Postgres
func (pdb *db) UpdateTaskListsWithTTL(_ context.Context, _ *sqlplugin.TaskListsRowWithTTL) (sql.Result, error) {
	return nil, sqlplugin.ErrTTLNotSupported
}
