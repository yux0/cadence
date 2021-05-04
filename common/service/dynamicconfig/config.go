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

package dynamicconfig

import (
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/types"
)

const (
	errCountLogThreshold = 1000
)

// NewCollection creates a new collection
func NewCollection(
	client Client,
	logger log.Logger,
	filterOptions ...FilterOption,
) *Collection {

	return &Collection{
		client:        client,
		logger:        logger,
		logKeys:       &sync.Map{},
		errCount:      -1,
		filterOptions: filterOptions,
	}
}

// Collection wraps dynamic config client with a closure so that across the code, the config values
// can be directly accessed by calling the function without propagating the client everywhere in
// code
type Collection struct {
	client        Client
	logger        log.Logger
	logKeys       *sync.Map // map of config Keys for logging to capture changes
	errCount      int64
	filterOptions []FilterOption
}

func (c *Collection) logError(key Key, err error) {
	errCount := atomic.AddInt64(&c.errCount, 1)
	if errCount%errCountLogThreshold == 0 {
		// log only every 'x' errors to reduce mem allocs and to avoid log noise
		if _, ok := err.(*types.EntityNotExistsError); ok {
			c.logger.Debug("dynamic config not set, use default value", tag.Key(key.String()))
		} else {
			c.logger.Warn("Failed to fetch key from dynamic config", tag.Key(key.String()), tag.Error(err))
		}
	}
}

func (c *Collection) logValue(
	key Key,
	value, defaultValue interface{},
	cmpValueEquals func(interface{}, interface{}) bool,
) {
	loadedValue, loaded := c.logKeys.LoadOrStore(key, value)
	if !loaded {
		c.logger.Info("First loading dynamic config",
			tag.Key(key.String()), tag.Value(value), tag.DefaultValue(defaultValue))
	} else {
		// it's loaded before, check if the value has changed
		if !cmpValueEquals(loadedValue, value) {
			c.logger.Info("Dynamic config has changed",
				tag.Key(key.String()), tag.Value(value), tag.DefaultValue(loadedValue))
			// update the logKeys so that we can capture the changes again
			// (ignore the racing condition here because it's just for logging, we need a lock if really need to solve it)
			c.logKeys.Store(key, value)
		}
	}
}

// PropertyFn is a wrapper to get property from dynamic config
type PropertyFn func() interface{}

// IntPropertyFn is a wrapper to get int property from dynamic config
type IntPropertyFn func(opts ...FilterOption) int

// IntPropertyFnWithDomainFilter is a wrapper to get int property from dynamic config with domain as filter
type IntPropertyFnWithDomainFilter func(domain string) int

// IntPropertyFnWithTaskListInfoFilters is a wrapper to get int property from dynamic config with three filters: domain, taskList, taskType
type IntPropertyFnWithTaskListInfoFilters func(domain string, taskList string, taskType int) int

// IntPropertyFnWithShardIDFilter is a wrapper to get int property from dynamic config with shardID as filter
type IntPropertyFnWithShardIDFilter func(shardID int) int

// FloatPropertyFn is a wrapper to get float property from dynamic config
type FloatPropertyFn func(opts ...FilterOption) float64

// FloatPropertyFnWithShardIDFilter is a wrapper to get float property from dynamic config with shardID as filter
type FloatPropertyFnWithShardIDFilter func(shardID int) float64

// DurationPropertyFn is a wrapper to get duration property from dynamic config
type DurationPropertyFn func(opts ...FilterOption) time.Duration

// DurationPropertyFnWithDomainFilter is a wrapper to get duration property from dynamic config with domain as filter
type DurationPropertyFnWithDomainFilter func(domain string) time.Duration

// DurationPropertyFnWithDomainIDFilter is a wrapper to get duration property from dynamic config with domainID as filter
type DurationPropertyFnWithDomainIDFilter func(domainID string) time.Duration

// DurationPropertyFnWithTaskListInfoFilters is a wrapper to get duration property from dynamic config  with three filters: domain, taskList, taskType
type DurationPropertyFnWithTaskListInfoFilters func(domain string, taskList string, taskType int) time.Duration

// DurationPropertyFnWithShardIDFilter is a wrapper to get duration property from dynamic config with shardID as filter
type DurationPropertyFnWithShardIDFilter func(shardID int) time.Duration

// BoolPropertyFn is a wrapper to get bool property from dynamic config
type BoolPropertyFn func(opts ...FilterOption) bool

// StringPropertyFn is a wrapper to get string property from dynamic config
type StringPropertyFn func(opts ...FilterOption) string

// MapPropertyFn is a wrapper to get map property from dynamic config
type MapPropertyFn func(opts ...FilterOption) map[string]interface{}

// StringPropertyFnWithDomainFilter is a wrapper to get string property from dynamic config
type StringPropertyFnWithDomainFilter func(domain string) string

// BoolPropertyFnWithDomainFilter is a wrapper to get bool property from dynamic config with domain as filter
type BoolPropertyFnWithDomainFilter func(domain string) bool

// BoolPropertyFnWithDomainIDFilter is a wrapper to get bool property from dynamic config with domainID as filter
type BoolPropertyFnWithDomainIDFilter func(domainID string) bool

// BoolPropertyFnWithTaskListInfoFilters is a wrapper to get bool property from dynamic config with three filters: domain, taskList, taskType
type BoolPropertyFnWithTaskListInfoFilters func(domain string, taskList string, taskType int) bool

// GetProperty gets a interface property and returns defaultValue if property is not found
func (c *Collection) GetProperty(key Key, defaultValue interface{}) PropertyFn {
	return func() interface{} {
		val, err := c.client.GetValue(key, defaultValue)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, reflect.DeepEqual)
		return val
	}
}

func getFilterMap(opts ...FilterOption) map[Filter]interface{} {
	l := len(opts)
	m := make(map[Filter]interface{}, l)
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// GetIntProperty gets property and asserts that it's an integer
func (c *Collection) GetIntProperty(key Key, defaultValue int) IntPropertyFn {
	return func(opts ...FilterOption) int {
		opts = append(opts, c.filterOptions...)
		val, err := c.client.GetIntValue(
			key,
			getFilterMap(opts...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, intCompareEquals)
		return val
	}
}

// GetIntPropertyFilteredByDomain gets property with domain filter and asserts that it's an integer
func (c *Collection) GetIntPropertyFilteredByDomain(key Key, defaultValue int) IntPropertyFnWithDomainFilter {

	return func(domain string) int {
		filters := append([]FilterOption{DomainFilter(domain)}, c.filterOptions...)
		val, err := c.client.GetIntValue(
			key,
			getFilterMap(filters...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, intCompareEquals)
		return val
	}
}

// GetIntPropertyFilteredByTaskListInfo gets property with taskListInfo as filters and asserts that it's an integer
func (c *Collection) GetIntPropertyFilteredByTaskListInfo(key Key, defaultValue int) IntPropertyFnWithTaskListInfoFilters {
	return func(domain string, taskList string, taskType int) int {
		filters := append(
			[]FilterOption{
				DomainFilter(domain),
				TaskListFilter(taskList),
				TaskTypeFilter(taskType),
			},
			c.filterOptions...,
		)
		val, err := c.client.GetIntValue(
			key,
			getFilterMap(filters...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, intCompareEquals)
		return val
	}
}

// GetIntPropertyFilteredByShardID gets property with shardID as filter and asserts that it's an integer
func (c *Collection) GetIntPropertyFilteredByShardID(key Key, defaultValue int) IntPropertyFnWithShardIDFilter {
	return func(shardID int) int {
		filters := append([]FilterOption{ShardIDFilter(shardID)}, c.filterOptions...)
		val, err := c.client.GetIntValue(
			key,
			getFilterMap(filters...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, intCompareEquals)
		return val
	}
}

// GetFloat64Property gets property and asserts that it's a float64
func (c *Collection) GetFloat64Property(key Key, defaultValue float64) FloatPropertyFn {
	return func(opts ...FilterOption) float64 {
		opts = append(opts, c.filterOptions...)
		val, err := c.client.GetFloatValue(
			key,
			getFilterMap(opts...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, float64CompareEquals)
		return val
	}
}

// GetFloat64PropertyFilteredByShardID gets property with shardID filter and asserts that it's a float64
func (c *Collection) GetFloat64PropertyFilteredByShardID(key Key, defaultValue float64) FloatPropertyFnWithShardIDFilter {
	return func(shardID int) float64 {
		filters := append([]FilterOption{ShardIDFilter(shardID)}, c.filterOptions...)
		val, err := c.client.GetFloatValue(
			key,
			getFilterMap(filters...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, float64CompareEquals)
		return val
	}
}

// GetDurationProperty gets property and asserts that it's a duration
func (c *Collection) GetDurationProperty(key Key, defaultValue time.Duration) DurationPropertyFn {
	return func(opts ...FilterOption) time.Duration {
		opts = append(opts, c.filterOptions...)
		val, err := c.client.GetDurationValue(
			key,
			getFilterMap(opts...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, durationCompareEquals)
		return val
	}
}

// GetDurationPropertyFilteredByDomain gets property with domain filter and asserts that it's a duration
func (c *Collection) GetDurationPropertyFilteredByDomain(key Key, defaultValue time.Duration) DurationPropertyFnWithDomainFilter {
	return func(domain string) time.Duration {
		filters := append([]FilterOption{DomainFilter(domain)}, c.filterOptions...)
		val, err := c.client.GetDurationValue(
			key,
			getFilterMap(filters...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, durationCompareEquals)
		return val
	}
}

// GetDurationPropertyFilteredByDomainID gets property with domainID filter and asserts that it's a duration
func (c *Collection) GetDurationPropertyFilteredByDomainID(key Key, defaultValue time.Duration) DurationPropertyFnWithDomainIDFilter {
	return func(domainID string) time.Duration {
		filters := append([]FilterOption{DomainIDFilter(domainID)}, c.filterOptions...)
		val, err := c.client.GetDurationValue(
			key,
			getFilterMap(filters...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, durationCompareEquals)
		return val
	}
}

// GetDurationPropertyFilteredByTaskListInfo gets property with taskListInfo as filters and asserts that it's a duration
func (c *Collection) GetDurationPropertyFilteredByTaskListInfo(key Key, defaultValue time.Duration) DurationPropertyFnWithTaskListInfoFilters {
	return func(domain string, taskList string, taskType int) time.Duration {
		filters := append(
			[]FilterOption{
				DomainFilter(domain),
				TaskListFilter(taskList),
				TaskTypeFilter(taskType),
			},
			c.filterOptions...,
		)
		val, err := c.client.GetDurationValue(
			key,
			getFilterMap(filters...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, durationCompareEquals)
		return val
	}
}

// GetDurationPropertyFilteredByShardID gets property with shardID id as filter and asserts that it's a duration
func (c *Collection) GetDurationPropertyFilteredByShardID(key Key, defaultValue time.Duration) DurationPropertyFnWithShardIDFilter {
	return func(shardID int) time.Duration {
		filters := append([]FilterOption{ShardIDFilter(shardID)}, c.filterOptions...)
		val, err := c.client.GetDurationValue(
			key,
			getFilterMap(filters...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, durationCompareEquals)
		return val
	}
}

// GetBoolProperty gets property and asserts that it's an bool
func (c *Collection) GetBoolProperty(key Key, defaultValue bool) BoolPropertyFn {
	return func(opts ...FilterOption) bool {
		opts = append(opts, c.filterOptions...)
		val, err := c.client.GetBoolValue(
			key,
			getFilterMap(opts...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, boolCompareEquals)
		return val
	}
}

// GetStringProperty gets property and asserts that it's an string
func (c *Collection) GetStringProperty(key Key, defaultValue string) StringPropertyFn {
	return func(opts ...FilterOption) string {
		opts = append(opts, c.filterOptions...)
		val, err := c.client.GetStringValue(
			key,
			getFilterMap(opts...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, stringCompareEquals)
		return val
	}
}

// GetMapProperty gets property and asserts that it's a map
func (c *Collection) GetMapProperty(key Key, defaultValue map[string]interface{}) MapPropertyFn {
	return func(opts ...FilterOption) map[string]interface{} {
		opts = append(opts, c.filterOptions...)
		val, err := c.client.GetMapValue(
			key,
			getFilterMap(opts...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, reflect.DeepEqual)
		return val
	}
}

// GetStringPropertyFilteredByDomain gets property with domain filter and asserts that it's a string
func (c *Collection) GetStringPropertyFilteredByDomain(key Key, defaultValue string) StringPropertyFnWithDomainFilter {
	return func(domain string) string {
		filters := append([]FilterOption{DomainFilter(domain)}, c.filterOptions...)
		val, err := c.client.GetStringValue(
			key,
			getFilterMap(filters...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, stringCompareEquals)
		return val
	}
}

// GetBoolPropertyFilteredByDomain gets property with domain filter and asserts that it's a bool
func (c *Collection) GetBoolPropertyFilteredByDomain(key Key, defaultValue bool) BoolPropertyFnWithDomainFilter {
	return func(domain string) bool {
		filters := append([]FilterOption{DomainFilter(domain)}, c.filterOptions...)
		val, err := c.client.GetBoolValue(
			key,
			getFilterMap(filters...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, boolCompareEquals)
		return val
	}
}

// GetBoolPropertyFilteredByDomainID gets property with domainID filter and asserts that it's a bool
func (c *Collection) GetBoolPropertyFilteredByDomainID(key Key, defaultValue bool) BoolPropertyFnWithDomainIDFilter {
	return func(domainID string) bool {
		filters := append([]FilterOption{DomainIDFilter(domainID)}, c.filterOptions...)
		val, err := c.client.GetBoolValue(
			key,
			getFilterMap(filters...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, boolCompareEquals)
		return val
	}
}

// GetBoolPropertyFilteredByTaskListInfo gets property with taskListInfo as filters and asserts that it's an bool
func (c *Collection) GetBoolPropertyFilteredByTaskListInfo(key Key, defaultValue bool) BoolPropertyFnWithTaskListInfoFilters {
	return func(domain string, taskList string, taskType int) bool {
		filters := append(
			[]FilterOption{
				DomainFilter(domain),
				TaskListFilter(taskList),
				TaskTypeFilter(taskType),
			},
			c.filterOptions...,
		)
		val, err := c.client.GetBoolValue(
			key,
			getFilterMap(filters...),
			defaultValue,
		)
		if err != nil {
			c.logError(key, err)
		}
		c.logValue(key, val, defaultValue, boolCompareEquals)
		return val
	}
}
