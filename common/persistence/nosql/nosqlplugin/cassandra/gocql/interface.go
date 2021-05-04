// Copyright (c) 2017-2020 Uber Technologies, Inc.
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

package gocql

import (
	"context"
	"time"

	"github.com/uber/cadence/common/service/config"
)

// Note: this file defines the minimal interface that is needed by Cadence's cassandra
// persistence implementation and should be implemented for all gocql libraries if
// they need to be used.
// Please add more methods to the interface if needed by the cassandra implementation.

type (
	// Client is an interface for all gocql libraries.
	Client interface {
		CreateSession(ClusterConfig) (Session, error)

		IsTimeoutError(error) bool
		IsNotFoundError(error) bool
		IsThrottlingError(error) bool
	}

	// Session is the interface for interacting with the database.
	Session interface {
		Query(string, ...interface{}) Query
		NewBatch(BatchType) Batch
		MapExecuteBatchCAS(Batch, map[string]interface{}) (bool, Iter, error)
		Close()
	}

	// Query is the interface for query object.
	Query interface {
		Exec() error
		Scan(...interface{}) error
		MapScan(map[string]interface{}) error
		MapScanCAS(map[string]interface{}) (bool, error)
		Iter() Iter
		PageSize(int) Query
		PageState([]byte) Query
		WithContext(context.Context) Query
		Consistency(Consistency) Query
	}

	// Batch is the interface for batch operation.
	Batch interface {
		Query(string, ...interface{})
		WithContext(context.Context) Batch
	}

	// Iter is the interface for executing and iterating over all resulting rows.
	Iter interface {
		Scan(...interface{}) bool
		MapScan(map[string]interface{}) bool
		PageState() []byte
		Close() error
	}

	// UUID represents a universally unique identifier
	UUID interface {
		String() string
	}

	// BatchType is the type of the Batch operation
	BatchType byte

	// Consistency is the consistency level used by a Query
	Consistency uint16

	// SerialConsistency is the serial consistency level used by a Query
	SerialConsistency uint16

	// ClusterConfig is the config for cassandra connection
	ClusterConfig struct {
		// TODO: explicitly define all the fields here to remove the dependency on common/service/config package
		config.Cassandra

		ProtoVersion      int
		Consistency       Consistency
		SerialConsistency SerialConsistency
		Timeout           time.Duration
	}
)
