// The MIT License (MIT)
//
// Copyright (c) 2020 Uber Technologies, Inc.
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

package common

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/uber/cadence/common/blobstore"
	"github.com/uber/cadence/common/pagination"
)

type (
	blobstoreIterator struct {
		itr pagination.Iterator
	}
)

var scanTypeExecFnMap = map[ScanType]func(data []byte) (*ScanOutputEntity, error){
	ConcreteExecutionType: deserializeConcreteExecution,
	CurrentExecutionType:  deserializeCurrentExecution,
}

// NewBlobstoreIterator constructs a new iterator backed by blobstore.
func NewBlobstoreIterator(
	client blobstore.Client,
	keys Keys,
	scanType ScanType,
) ScanOutputIterator {
	return &blobstoreIterator{
		itr: pagination.NewIterator(keys.MinPage, getBlobstoreFetchPageFn(client, keys, scanTypeExecFnMap[scanType])),
	}
}

// Next returns the next ScanOutputEntity
func (i *blobstoreIterator) Next() (*ScanOutputEntity, error) {
	exec, err := i.itr.Next()
	if exec != nil {
		return exec.(*ScanOutputEntity), err
	}
	return nil, err
}

// HasNext returns true if there is a next ScanOutputEntity false otherwise
func (i *blobstoreIterator) HasNext() bool {
	return i.itr.HasNext()
}

func getBlobstoreFetchPageFn(
	client blobstore.Client,
	keys Keys,
	execDeserializeFunc func(data []byte) (*ScanOutputEntity, error),
) pagination.FetchFn {
	return func(token pagination.PageToken) (pagination.Page, error) {
		index := token.(int)
		key := pageNumberToKey(keys.UUID, keys.Extension, index)
		ctx, cancel := context.WithTimeout(context.Background(), BlobstoreTimeout)
		defer cancel()
		req := &blobstore.GetRequest{
			Key: key,
		}
		resp, err := client.Get(ctx, req)
		if err != nil {
			return pagination.Page{}, err
		}
		parts := bytes.Split(resp.Blob.Body, BlobstoreSeparatorToken)
		var executions []pagination.Entity
		for _, p := range parts {
			if len(p) == 0 {
				continue
			}
			soe, err := execDeserializeFunc(p)
			if err != nil {
				return pagination.Page{}, err
			}
			executions = append(executions, soe)
		}
		var nextPageToken interface{} = index + 1
		if nextPageToken.(int) > keys.MaxPage {
			nextPageToken = nil
		}
		return pagination.Page{
			CurrentToken: token,
			NextToken:    nextPageToken,
			Entities:     executions,
		}, nil
	}
}

func deserializeConcreteExecution(data []byte) (*ScanOutputEntity, error) {
	soe := &ScanOutputEntity{
		Execution: &ConcreteExecution{},
	}
	if err := json.Unmarshal(data, &soe); err != nil {
		return nil, err
	}
	if err := ValidateConcreteExecution(soe.Execution.(*ConcreteExecution)); err != nil {
		return nil, err
	}
	return soe, nil
}

func deserializeCurrentExecution(data []byte) (*ScanOutputEntity, error) {
	soe := &ScanOutputEntity{
		Execution: &CurrentExecution{},
	}
	if err := json.Unmarshal(data, &soe); err != nil {
		return nil, err
	}
	if err := ValidateCurrentExecution(soe.Execution.(*CurrentExecution)); err != nil {
		return nil, err
	}
	return soe, nil
}
