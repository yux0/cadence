// Copyright (c) 2017 Uber Technologies, Inc.
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

package metrics

import (
	"time"

	"github.com/uber-go/tally"
)

type nopStopwatchRecorder struct{}

// RecordStopwatch is a nop impl for replay mode
func (n *nopStopwatchRecorder) RecordStopwatch(stopwatchStart time.Time) {}

// NopStopwatch return a fake tally stop watch
func NopStopwatch() tally.Stopwatch {
	return tally.NewStopwatch(time.Now(), &nopStopwatchRecorder{})
}

type noopClientImpl struct{}

func (n noopClientImpl) IncCounter(scope int, counter int) {
	return
}

func (n noopClientImpl) AddCounter(scope int, counter int, delta int64) {
	return
}

func (n noopClientImpl) StartTimer(scope int, timer int) tally.Stopwatch {
	return NopStopwatch()
}

func (n noopClientImpl) RecordTimer(scope int, timer int, d time.Duration) {
	return
}

func (n noopClientImpl) UpdateGauge(scope int, gauge int, value float64) {
	return
}

func (n noopClientImpl) Scope(scope int, tags ...Tag) Scope {
	return NoopScope(Common)
}

func NewNoopMetricsClient() Client {
	return &noopClientImpl{}
}
