// Copyright (c) 2020 Uber Technologies, Inc.
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

package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"

	workflow "github.com/uber/cadence/.gen/go/shared"
)

type (
	// ElasticVisibilityPageToken holds the paging token for ElasticSearch
	ElasticVisibilityPageToken struct {
		// for ES API From+Size
		From int
		// for ES API searchAfter
		SortValue  interface{}
		TieBreaker string // runID
		// for ES scroll API
		ScrollID string
	}
)

// DeserializePageToken return the structural token
func DeserializePageToken(data []byte) (*ElasticVisibilityPageToken, error) {
	var token ElasticVisibilityPageToken
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	err := dec.Decode(&token)
	if err != nil {
		return nil, &workflow.BadRequestError{
			Message: fmt.Sprintf("unable to deserialize page token. err: %v", err),
		}
	}
	return &token, nil
}

// SerializePageToken return the token blob
func SerializePageToken(token *ElasticVisibilityPageToken) ([]byte, error) {
	data, err := json.Marshal(token)
	if err != nil {
		return nil, &workflow.BadRequestError{
			Message: fmt.Sprintf("unable to serialize page token. err: %v", err),
		}
	}
	return data, nil
}

// GetNextPageToken returns the structural token with nil handling
func GetNextPageToken(token []byte) (*ElasticVisibilityPageToken, error) {
	var result *ElasticVisibilityPageToken
	var err error
	if len(token) > 0 {
		result, err = DeserializePageToken(token)
		if err != nil {
			return nil, err
		}
	} else {
		result = &ElasticVisibilityPageToken{}
	}
	return result, nil
}

// ShouldSearchAfter decides if should search after
func ShouldSearchAfter(token *ElasticVisibilityPageToken) bool {
	return token.TieBreaker != ""
}
