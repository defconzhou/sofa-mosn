/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package stats

import (
	"sync"

	"github.com/rcrowley/go-metrics"
	"github.com/alipay/sofa-mosn/pkg/types"
	"sort"
)

// stats memory store
type store struct {
	metrics []types.Metrics
	mutex   sync.RWMutex
}

var defaultStore *store

func init() {
	defaultStore = &store{
		// TODO: default length configurable
		metrics: make([]types.Metrics, 0, 100),
	}
}

// Stats is a wrapper of go-metrics registry, is an implement of types.Metrics
type Stats struct {
	typ       string
	labels    map[string]string
	labelKeys []string
	labelVals []string

	registry metrics.Registry
}

// NewStats returns a Stats
// metrics key prefix is "${type}.${namespace}"
// "@" is the reserved sep, any "@" in type and namespace will be dropped
func NewStats(typ string, labels map[string]string) types.Metrics {
	defaultStore.mutex.Lock()
	defer defaultStore.mutex.Unlock()

	// check existence
	for _, metric := range defaultStore.metrics {
		if metric.Type() == typ && mapEqual(metric.Labels(), labels) {
			return metric
		}
	}

	stats := &Stats{
		typ:      typ,
		labels:   labels,
		registry: metrics.NewRegistry(),
	}

	defaultStore.metrics = append(defaultStore.metrics, stats)

	return stats
}

// NewStats returns a Stats
// metrics key prefix is "${type}.${namespace}"
// "@" is the reserved sep, any "@" in type and namespace will be dropped
func NewStatsAlways(typ string, labels map[string]string) types.Metrics {
	defaultStore.mutex.Lock()
	defer defaultStore.mutex.Unlock()

	stats := &Stats{
		typ:      typ,
		labels:   labels,
		registry: metrics.NewRegistry(),
	}

	defaultStore.metrics = append(defaultStore.metrics, stats)

	return stats
}

func (s *Stats) Type() string {
	return s.typ
}

func (s *Stats) Labels() map[string]string {
	return s.labels
}

func (s *Stats) SortedLabels() (keys, values []string) {
	if s.labelKeys != nil && s.labelVals != nil {
		return s.labelKeys, s.labelVals
	}

	for k := range s.labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		values = append(values, s.labels[k])
	}
	return
}

// Counter creates or returns a go-metrics counter by key
// if the key is registered by other interface, it will be panic
func (s *Stats) Counter(key string) metrics.Counter {
	return s.registry.GetOrRegister(key, metrics.NewCounter).(metrics.Counter)
}

// Gauge creates or returns a go-metrics gauge by key
// if the key is registered by other interface, it will be panic
func (s *Stats) Gauge(key string) metrics.Gauge {
	return s.registry.GetOrRegister(key, metrics.NewGauge).(metrics.Gauge)
}

// Histogram creates or returns a go-metrics histogram by key
// if the key is registered by other interface, it will be panic
func (s *Stats) Histogram(key string) metrics.Histogram {
	return s.registry.GetOrRegister(key, func() metrics.Histogram { return metrics.NewHistogram(metrics.NewUniformSample(100)) }).(metrics.Histogram)
}

func (s *Stats) Each(f func(string, interface{})) {
	s.registry.Each(f)
}

func (s *Stats) UnregisterAll() {
	s.registry.UnregisterAll()
}

// GetAll returns all metrics data
func GetAll() (metrics []types.Metrics) {
	defaultStore.mutex.RLock()
	defer defaultStore.mutex.RUnlock()
	return defaultStore.metrics
}

// ResetAll is only for test and internal usage. DO NOT use this if not sure.
func ResetAll() {
	defaultStore.mutex.Lock()
	defer defaultStore.mutex.Unlock()

	for _, m := range defaultStore.metrics {
		m.UnregisterAll()
	}
	defaultStore.metrics = defaultStore.metrics[:0]
}

func mapEqual(x, y map[string]string) bool {
	if len(x) != len(y) {
		return false
	}
	for k, xv := range x {
		if yv, ok := y[k]; !ok || yv != xv {
			return false
		}
	}
	return true
}