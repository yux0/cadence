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

import "github.com/gocql/gocql"

var _ Session = (*session)(nil)

type (
	session struct {
		*gocql.Session
	}
)

func (s *session) Query(
	stmt string,
	values ...interface{},
) Query {
	return &query{
		Query: s.Session.Query(stmt, values...),
	}
}

func (s *session) NewBatch(
	batchType BatchType,
) Batch {
	return &batch{
		Batch: s.Session.NewBatch(mustConvertBatchType(batchType)),
	}
}

func (s *session) MapExecuteBatchCAS(
	b Batch,
	previous map[string]interface{},
) (bool, Iter, error) {
	return s.Session.MapExecuteBatchCAS(b.(*batch).Batch, previous)
}
