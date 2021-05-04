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

//go:generate enumer -type=ScanType

import (
	"github.com/uber/cadence/common/pagination"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/reconciliation/entity"
	"github.com/uber/cadence/common/reconciliation/fetcher"
	"github.com/uber/cadence/common/reconciliation/invariant"
)

const (
	// ConcreteExecutionType concrete execution entity
	ConcreteExecutionType ScanType = iota
	// CurrentExecutionType current execution entity
	CurrentExecutionType
)

// ScanType is the enum for representing different entity types to scan
type ScanType int

type (
	//InvariantFactory represents a function which returns Invariant
	InvariantFactory func(retryer persistence.Retryer) invariant.Invariant

	//ExecutionFetcher represents a function which returns specific execution entity
	ExecutionFetcher func(retryer persistence.Retryer, request fetcher.ExecutionRequest) (entity.Entity, error)
)

// ToBlobstoreEntity picks struct depending on scanner type.
func (st ScanType) ToBlobstoreEntity() entity.Entity {
	switch st {
	case ConcreteExecutionType:
		return &entity.ConcreteExecution{}
	case CurrentExecutionType:
		return &entity.CurrentExecution{}
	}
	panic("unknown scan type")
}

// ToIterator selects appropriate iterator. It will panic if scan type is unknown.
func (st ScanType) ToIterator() func(retryer persistence.Retryer, pageSize int) pagination.Iterator {
	switch st {
	case ConcreteExecutionType:
		return fetcher.ConcreteExecutionIterator
	case CurrentExecutionType:
		return fetcher.CurrentExecutionIterator
	default:
		panic("unknown scan type")
	}
}

// ToExecutionFetcher selects appropriate execution fetcher. Fetcher returns single execution entity. It will panic if scan type is unknown.
func (st ScanType) ToExecutionFetcher() ExecutionFetcher {
	switch st {
	case ConcreteExecutionType:
		return fetcher.ConcreteExecution
	case CurrentExecutionType:
		return fetcher.CurrentExecution
	default:
		panic("unknown scan type")
	}
}

// ToInvariants returns list of invariants to be checked depending on scan type.
func (st ScanType) ToInvariants(collections []invariant.Collection) []InvariantFactory {
	var fns []InvariantFactory
	switch st {
	case ConcreteExecutionType:
		for _, collection := range collections {
			switch collection {
			case invariant.CollectionHistory:
				fns = append(fns, invariant.NewHistoryExists)
			case invariant.CollectionMutableState:
				fns = append(fns, invariant.NewOpenCurrentExecution)
			}
		}
		return fns
	case CurrentExecutionType:
		for _, collection := range collections {
			switch collection {
			case invariant.CollectionMutableState:
				fns = append(fns, invariant.NewConcreteExecutionExists)
			}
		}
		return fns
	default:
		panic("unknown scan type")
	}
}
