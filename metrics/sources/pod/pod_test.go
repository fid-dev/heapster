// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pod

import (
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube_api "k8s.io/client-go/pkg/api/v1"
	. "k8s.io/heapster/metrics/core"
	"testing"
	"time"
)

var (
	flaggedAnnotations = map[string]string{
		ScrapeEnabledAnnotationKey: "true",
	}

	emptyAnnotations = map[string]string{}
)

func TestPodMetricsSource_ScrapeMetrics(t *testing.T) {

	ms := newTestPodMetricsSource(t)

	end := time.Now()
	start := end.Truncate(10 * time.Minute)

	batch := ms.ScrapeMetrics(start, end)

	assert.Equal(t, end, batch.Timestamp)
	assert.Equal(t, 0, len(batch.MetricSets))

	//assertMetricPodLabels(t, v, "test-a", "foo-namespace")
}

func assertMetricPodLabels(t *testing.T, vector model.Vector, podName, namespaceName string) {
	var (
		p       model.LabelValue
		found   bool
		noFound int
	)

	for _, metric := range vector {
		noFound = 0

		if p, found = metric.Metric[model.LabelName("pod_name")]; found {
			assert.Equal(t, podName, string(p))
			noFound++
		}
		if p, found = metric.Metric[model.LabelName("namespace_name")]; found {
			assert.Equal(t, namespaceName, string(p))
			noFound++
		}

		assert.Equal(t, 2, noFound, "Expected two labels to be found for each metric")
	}

}

func newTestPodMetricsSource(t *testing.T) MetricsSource {

	pod1 := newTestPod("test-a", "10.10.10.10", flaggedAnnotations)
	podConfig1, err := getPodConfig(pod1)
	assert.NoError(t, err)

	vector := new(model.Vector)

	client := newTestPromClient()
	client.AddHandler(podConfig1.ip, podConfig1.port, podConfig1.path, vector)

	// TODO inject bucket for storing metrics
	return NewPodMetricsSource(pod1, podConfig1, client)
}

func newTestPromClient() *testPromClient {
	return &testPromClient{}
}

type testPromClient struct {
	handlers []testPromHandler
}

func (p *testPromClient) AddHandler(ip string, port int32, path string, result *model.Vector) {
	p.handlers = append(p.handlers, testPromHandler{
		ip:     ip,
		port:   port,
		path:   path,
		result: result,
	})
}

func (p *testPromClient) GetPodMetrics(pod *kube_api.Pod, ip string, port int32, path string, t time.Time) (*model.Vector, error) {
	for _, handler := range p.handlers {
		if ip == handler.ip && port == handler.port && path == handler.path {
			return handler.result, nil
		}
	}

	return &model.Vector{}, nil
}

type testPromHandler struct {
	ip     string
	port   int32
	path   string
	result *model.Vector
}

func newTestPod(name, podIP string, annotations map[string]string) *kube_api.Pod {
	return &kube_api.Pod{
		Status: kube_api.PodStatus{
			PodIP: podIP,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   "foo-namespace",
			Annotations: annotations,
		},
		Spec: kube_api.PodSpec{
			Containers: []kube_api.Container{
				{
					Image: "testimage",
				},
			},
		},
	}
}

var testDataPrometheus string = `
# HELP http_request_duration_microseconds The HTTP request latencies in microseconds.
# TYPE http_request_duration_microseconds summary
http_request_duration_microseconds{handler="/",quantile="0.5"} 0
http_request_duration_microseconds{handler="/",quantile="0.9"} 0
http_request_duration_microseconds{handler="/",quantile="0.99"} 0
http_request_duration_microseconds_sum{handler="/"} 0
http_request_duration_microseconds_count{handler="/"} 0
http_request_duration_microseconds{handler="/alerts",quantile="0.5"} 0
http_request_duration_microseconds{handler="/alerts",quantile="0.9"} 0
http_request_duration_microseconds{handler="/alerts",quantile="0.99"} 0
http_request_duration_microseconds_sum{handler="/alerts"} 0
http_request_duration_microseconds_count{handler="/alerts"} 0
http_request_duration_microseconds{handler="/api/metrics",quantile="0.5"} 0
http_request_duration_microseconds{handler="/api/metrics",quantile="0.9"} 0
http_request_duration_microseconds{handler="/api/metrics",quantile="0.99"} 0
http_request_duration_microseconds_sum{handler="/api/metrics"} 0
http_request_duration_microseconds_count{handler="/api/metrics"} 0
http_request_duration_microseconds{handler="/api/query",quantile="0.5"} 0
http_request_duration_microseconds{handler="/api/query",quantile="0.9"} 0
http_request_duration_microseconds{handler="/api/query",quantile="0.99"} 0
http_request_duration_microseconds_sum{handler="/api/query"} 0
http_request_duration_microseconds_count{handler="/api/query"} 0
http_request_duration_microseconds{handler="/api/query_range",quantile="0.5"} 0
http_request_duration_microseconds{handler="/api/query_range",quantile="0.9"} 0
http_request_duration_microseconds{handler="/api/query_range",quantile="0.99"} 0
http_request_duration_microseconds_sum{handler="/api/query_range"} 0
http_request_duration_microseconds_count{handler="/api/query_range"} 0
http_request_duration_microseconds{handler="/api/targets",quantile="0.5"} 0
http_request_duration_microseconds{handler="/api/targets",quantile="0.9"} 0
http_request_duration_microseconds{handler="/api/targets",quantile="0.99"} 0
http_request_duration_microseconds_sum{handler="/api/targets"} 0
http_request_duration_microseconds_count{handler="/api/targets"} 0
http_request_duration_microseconds{handler="/consoles/",quantile="0.5"} 0
http_request_duration_microseconds{handler="/consoles/",quantile="0.9"} 0
http_request_duration_microseconds{handler="/consoles/",quantile="0.99"} 0
http_request_duration_microseconds_sum{handler="/consoles/"} 0
http_request_duration_microseconds_count{handler="/consoles/"} 0
http_request_duration_microseconds{handler="/graph",quantile="0.5"} 0
http_request_duration_microseconds{handler="/graph",quantile="0.9"} 0
http_request_duration_microseconds{handler="/graph",quantile="0.99"} 0
http_request_duration_microseconds_sum{handler="/graph"} 0
http_request_duration_microseconds_count{handler="/graph"} 0
http_request_duration_microseconds{handler="/heap",quantile="0.5"} 0
http_request_duration_microseconds{handler="/heap",quantile="0.9"} 0
http_request_duration_microseconds{handler="/heap",quantile="0.99"} 0
http_request_duration_microseconds_sum{handler="/heap"} 0
http_request_duration_microseconds_count{handler="/heap"} 0
http_request_duration_microseconds{handler="/static/",quantile="0.5"} 0
http_request_duration_microseconds{handler="/static/",quantile="0.9"} 0
http_request_duration_microseconds{handler="/static/",quantile="0.99"} 0
http_request_duration_microseconds_sum{handler="/static/"} 0
http_request_duration_microseconds_count{handler="/static/"} 0
http_request_duration_microseconds{handler="prometheus",quantile="0.5"} 1307.275
http_request_duration_microseconds{handler="prometheus",quantile="0.9"} 1858.632
http_request_duration_microseconds{handler="prometheus",quantile="0.99"} 3087.384
http_request_duration_microseconds_sum{handler="prometheus"} 179886.5000000001
http_request_duration_microseconds_count{handler="prometheus"} 119
# HELP http_request_size_bytes The HTTP request sizes in bytes.
# TYPE http_request_size_bytes summary
http_request_size_bytes{handler="/",quantile="0.5"} 0
http_request_size_bytes{handler="/",quantile="0.9"} 0
http_request_size_bytes{handler="/",quantile="0.99"} 0
http_request_size_bytes_sum{handler="/"} 0
http_request_size_bytes_count{handler="/"} 0
http_request_size_bytes{handler="/alerts",quantile="0.5"} 0
http_request_size_bytes{handler="/alerts",quantile="0.9"} 0
http_request_size_bytes{handler="/alerts",quantile="0.99"} 0
http_request_size_bytes_sum{handler="/alerts"} 0
http_request_size_bytes_count{handler="/alerts"} 0
http_request_size_bytes{handler="/api/metrics",quantile="0.5"} 0
http_request_size_bytes{handler="/api/metrics",quantile="0.9"} 0
http_request_size_bytes{handler="/api/metrics",quantile="0.99"} 0
http_request_size_bytes_sum{handler="/api/metrics"} 0
http_request_size_bytes_count{handler="/api/metrics"} 0
http_request_size_bytes{handler="/api/query",quantile="0.5"} 0
http_request_size_bytes{handler="/api/query",quantile="0.9"} 0
http_request_size_bytes{handler="/api/query",quantile="0.99"} 0
http_request_size_bytes_sum{handler="/api/query"} 0
http_request_size_bytes_count{handler="/api/query"} 0
http_request_size_bytes{handler="/api/query_range",quantile="0.5"} 0
http_request_size_bytes{handler="/api/query_range",quantile="0.9"} 0
http_request_size_bytes{handler="/api/query_range",quantile="0.99"} 0
http_request_size_bytes_sum{handler="/api/query_range"} 0
http_request_size_bytes_count{handler="/api/query_range"} 0
http_request_size_bytes{handler="/api/targets",quantile="0.5"} 0
http_request_size_bytes{handler="/api/targets",quantile="0.9"} 0
http_request_size_bytes{handler="/api/targets",quantile="0.99"} 0
http_request_size_bytes_sum{handler="/api/targets"} 0
http_request_size_bytes_count{handler="/api/targets"} 0
http_request_size_bytes{handler="/consoles/",quantile="0.5"} 0
http_request_size_bytes{handler="/consoles/",quantile="0.9"} 0
http_request_size_bytes{handler="/consoles/",quantile="0.99"} 0
http_request_size_bytes_sum{handler="/consoles/"} 0
http_request_size_bytes_count{handler="/consoles/"} 0
http_request_size_bytes{handler="/graph",quantile="0.5"} 0
http_request_size_bytes{handler="/graph",quantile="0.9"} 0
http_request_size_bytes{handler="/graph",quantile="0.99"} 0
http_request_size_bytes_sum{handler="/graph"} 0
http_request_size_bytes_count{handler="/graph"} 0
http_request_size_bytes{handler="/heap",quantile="0.5"} 0
http_request_size_bytes{handler="/heap",quantile="0.9"} 0
http_request_size_bytes{handler="/heap",quantile="0.99"} 0
http_request_size_bytes_sum{handler="/heap"} 0
http_request_size_bytes_count{handler="/heap"} 0
http_request_size_bytes{handler="/static/",quantile="0.5"} 0
http_request_size_bytes{handler="/static/",quantile="0.9"} 0
http_request_size_bytes{handler="/static/",quantile="0.99"} 0
http_request_size_bytes_sum{handler="/static/"} 0
http_request_size_bytes_count{handler="/static/"} 0
http_request_size_bytes{handler="prometheus",quantile="0.5"} 291
http_request_size_bytes{handler="prometheus",quantile="0.9"} 291
http_request_size_bytes{handler="prometheus",quantile="0.99"} 291
http_request_size_bytes_sum{handler="prometheus"} 34488
http_request_size_bytes_count{handler="prometheus"} 119
# HELP http_requests_total Total number of HTTP requests made.
# TYPE http_requests_total counter
http_requests_total{code="200",handler="prometheus",method="get"} 119
# HELP http_response_size_bytes The HTTP response sizes in bytes.
# TYPE http_response_size_bytes summary
http_response_size_bytes{handler="/",quantile="0.5"} 0
http_response_size_bytes{handler="/",quantile="0.9"} 0
http_response_size_bytes{handler="/",quantile="0.99"} 0
http_response_size_bytes_sum{handler="/"} 0
http_response_size_bytes_count{handler="/"} 0
http_response_size_bytes{handler="/alerts",quantile="0.5"} 0
http_response_size_bytes{handler="/alerts",quantile="0.9"} 0
http_response_size_bytes{handler="/alerts",quantile="0.99"} 0
http_response_size_bytes_sum{handler="/alerts"} 0
http_response_size_bytes_count{handler="/alerts"} 0
http_response_size_bytes{handler="/api/metrics",quantile="0.5"} 0
http_response_size_bytes{handler="/api/metrics",quantile="0.9"} 0
http_response_size_bytes{handler="/api/metrics",quantile="0.99"} 0
http_response_size_bytes_sum{handler="/api/metrics"} 0
http_response_size_bytes_count{handler="/api/metrics"} 0
http_response_size_bytes{handler="/api/query",quantile="0.5"} 0
http_response_size_bytes{handler="/api/query",quantile="0.9"} 0
http_response_size_bytes{handler="/api/query",quantile="0.99"} 0
http_response_size_bytes_sum{handler="/api/query"} 0
http_response_size_bytes_count{handler="/api/query"} 0
http_response_size_bytes{handler="/api/query_range",quantile="0.5"} 0
http_response_size_bytes{handler="/api/query_range",quantile="0.9"} 0
http_response_size_bytes{handler="/api/query_range",quantile="0.99"} 0
http_response_size_bytes_sum{handler="/api/query_range"} 0
http_response_size_bytes_count{handler="/api/query_range"} 0
http_response_size_bytes{handler="/api/targets",quantile="0.5"} 0
http_response_size_bytes{handler="/api/targets",quantile="0.9"} 0
http_response_size_bytes{handler="/api/targets",quantile="0.99"} 0
http_response_size_bytes_sum{handler="/api/targets"} 0
http_response_size_bytes_count{handler="/api/targets"} 0
http_response_size_bytes{handler="/consoles/",quantile="0.5"} 0
http_response_size_bytes{handler="/consoles/",quantile="0.9"} 0
http_response_size_bytes{handler="/consoles/",quantile="0.99"} 0
http_response_size_bytes_sum{handler="/consoles/"} 0
http_response_size_bytes_count{handler="/consoles/"} 0
http_response_size_bytes{handler="/graph",quantile="0.5"} 0
http_response_size_bytes{handler="/graph",quantile="0.9"} 0
http_response_size_bytes{handler="/graph",quantile="0.99"} 0
http_response_size_bytes_sum{handler="/graph"} 0
http_response_size_bytes_count{handler="/graph"} 0
http_response_size_bytes{handler="/heap",quantile="0.5"} 0
http_response_size_bytes{handler="/heap",quantile="0.9"} 0
http_response_size_bytes{handler="/heap",quantile="0.99"} 0
http_response_size_bytes_sum{handler="/heap"} 0
http_response_size_bytes_count{handler="/heap"} 0
http_response_size_bytes{handler="/static/",quantile="0.5"} 0
http_response_size_bytes{handler="/static/",quantile="0.9"} 0
http_response_size_bytes{handler="/static/",quantile="0.99"} 0
http_response_size_bytes_sum{handler="/static/"} 0
http_response_size_bytes_count{handler="/static/"} 0
http_response_size_bytes{handler="prometheus",quantile="0.5"} 2049
http_response_size_bytes{handler="prometheus",quantile="0.9"} 2058
http_response_size_bytes{handler="prometheus",quantile="0.99"} 2064
http_response_size_bytes_sum{handler="prometheus"} 247001
http_response_size_bytes_count{handler="prometheus"} 119
# HELP process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 0.55
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 70
# HELP process_max_fds Maximum number of open file descriptors.
# TYPE process_max_fds gauge
process_max_fds 8192
# HELP process_open_fds Number of open file descriptors.
# TYPE process_open_fds gauge
process_open_fds 29
# HELP process_resident_memory_bytes Resident memory size in bytes.
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 5.3870592e+07
# HELP process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE process_start_time_seconds gauge
process_start_time_seconds 1.42236894836e+09
# HELP process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes 5.41478912e+08
# HELP prometheus_dns_sd_lookup_failures_total The number of DNS-SD lookup failures.
# TYPE prometheus_dns_sd_lookup_failures_total counter
prometheus_dns_sd_lookup_failures_total 0
# HELP prometheus_dns_sd_lookups_total The number of DNS-SD lookups.
# TYPE prometheus_dns_sd_lookups_total counter
prometheus_dns_sd_lookups_total 7
# HELP prometheus_evaluator_duration_milliseconds The duration for all evaluations to execute.
# TYPE prometheus_evaluator_duration_milliseconds summary
prometheus_evaluator_duration_milliseconds{quantile="0.01"} 0
prometheus_evaluator_duration_milliseconds{quantile="0.05"} 0
prometheus_evaluator_duration_milliseconds{quantile="0.5"} 0
prometheus_evaluator_duration_milliseconds{quantile="0.9"} 1
prometheus_evaluator_duration_milliseconds{quantile="0.99"} 1
prometheus_evaluator_duration_milliseconds_sum 12
prometheus_evaluator_duration_milliseconds_count 23
# HELP prometheus_local_storage_checkpoint_duration_milliseconds The duration (in milliseconds) it took to checkpoint in-memory metrics and head chunks.
# TYPE prometheus_local_storage_checkpoint_duration_milliseconds gauge
prometheus_local_storage_checkpoint_duration_milliseconds 0
# HELP prometheus_local_storage_chunk_ops_total The total number of chunk operations by their type.
# TYPE prometheus_local_storage_chunk_ops_total counter
prometheus_local_storage_chunk_ops_total{type="create"} 598
prometheus_local_storage_chunk_ops_total{type="persist"} 174
prometheus_local_storage_chunk_ops_total{type="pin"} 920
prometheus_local_storage_chunk_ops_total{type="transcode"} 415
prometheus_local_storage_chunk_ops_total{type="unpin"} 920
# HELP prometheus_local_storage_indexing_batch_latency_milliseconds Quantiles for batch indexing latencies in milliseconds.
# TYPE prometheus_local_storage_indexing_batch_latency_milliseconds summary
prometheus_local_storage_indexing_batch_latency_milliseconds{quantile="0.5"} 0
prometheus_local_storage_indexing_batch_latency_milliseconds{quantile="0.9"} 0
prometheus_local_storage_indexing_batch_latency_milliseconds{quantile="0.99"} 0
prometheus_local_storage_indexing_batch_latency_milliseconds_sum 0
prometheus_local_storage_indexing_batch_latency_milliseconds_count 1
# HELP prometheus_local_storage_indexing_batch_sizes Quantiles for indexing batch sizes (number of metrics per batch).
# TYPE prometheus_local_storage_indexing_batch_sizes summary
prometheus_local_storage_indexing_batch_sizes{quantile="0.5"} 2
prometheus_local_storage_indexing_batch_sizes{quantile="0.9"} 2
prometheus_local_storage_indexing_batch_sizes{quantile="0.99"} 2
prometheus_local_storage_indexing_batch_sizes_sum 2
prometheus_local_storage_indexing_batch_sizes_count 1
# HELP prometheus_local_storage_indexing_queue_capacity The capacity of the indexing queue.
# TYPE prometheus_local_storage_indexing_queue_capacity gauge
prometheus_local_storage_indexing_queue_capacity 16384
# HELP prometheus_local_storage_indexing_queue_length The number of metrics waiting to be indexed.
# TYPE prometheus_local_storage_indexing_queue_length gauge
prometheus_local_storage_indexing_queue_length 0
# HELP prometheus_local_storage_ingested_samples_total The total number of samples ingested.
# TYPE prometheus_local_storage_ingested_samples_total counter
prometheus_local_storage_ingested_samples_total 30473
# HELP prometheus_local_storage_invalid_preload_requests_total The total number of preload requests referring to a non-existent series. This is an indication of outdated label indexes.
# TYPE prometheus_local_storage_invalid_preload_requests_total counter
prometheus_local_storage_invalid_preload_requests_total 0
# HELP prometheus_local_storage_memory_chunkdescs The current number of chunk descriptors in memory.
# TYPE prometheus_local_storage_memory_chunkdescs gauge
prometheus_local_storage_memory_chunkdescs 1059
# HELP prometheus_local_storage_memory_chunks The current number of chunks in memory, excluding cloned chunks (i.e. chunks without a descriptor).
# TYPE prometheus_local_storage_memory_chunks gauge
prometheus_local_storage_memory_chunks 1020
# HELP prometheus_local_storage_memory_series The current number of series in memory.
# TYPE prometheus_local_storage_memory_series gauge
prometheus_local_storage_memory_series 424
# HELP prometheus_local_storage_persist_latency_microseconds A summary of latencies for persisting each chunk.
# TYPE prometheus_local_storage_persist_latency_microseconds summary
prometheus_local_storage_persist_latency_microseconds{quantile="0.5"} 30.377
prometheus_local_storage_persist_latency_microseconds{quantile="0.9"} 203.539
prometheus_local_storage_persist_latency_microseconds{quantile="0.99"} 2626.463
prometheus_local_storage_persist_latency_microseconds_sum 20424.415
prometheus_local_storage_persist_latency_microseconds_count 174
# HELP prometheus_local_storage_persist_queue_capacity The total capacity of the persist queue.
# TYPE prometheus_local_storage_persist_queue_capacity gauge
prometheus_local_storage_persist_queue_capacity 1024
# HELP prometheus_local_storage_persist_queue_length The current number of chunks waiting in the persist queue.
# TYPE prometheus_local_storage_persist_queue_length gauge
prometheus_local_storage_persist_queue_length 0
# HELP prometheus_local_storage_series_ops_total The total number of series operations by their type.
# TYPE prometheus_local_storage_series_ops_total counter
prometheus_local_storage_series_ops_total{type="create"} 2
prometheus_local_storage_series_ops_total{type="maintenance_in_memory"} 11
# HELP prometheus_notifications_latency_milliseconds Latency quantiles for sending alert notifications (not including dropped notifications).
# TYPE prometheus_notifications_latency_milliseconds summary
prometheus_notifications_latency_milliseconds{quantile="0.5"} 0
prometheus_notifications_latency_milliseconds{quantile="0.9"} 0
prometheus_notifications_latency_milliseconds{quantile="0.99"} 0
prometheus_notifications_latency_milliseconds_sum 0
prometheus_notifications_latency_milliseconds_count 0
# HELP prometheus_notifications_queue_capacity The capacity of the alert notifications queue.
# TYPE prometheus_notifications_queue_capacity gauge
prometheus_notifications_queue_capacity 100
# HELP prometheus_notifications_queue_length The number of alert notifications in the queue.
# TYPE prometheus_notifications_queue_length gauge
prometheus_notifications_queue_length 0
# HELP prometheus_rule_evaluation_duration_milliseconds The duration for a rule to execute.
# TYPE prometheus_rule_evaluation_duration_milliseconds summary
prometheus_rule_evaluation_duration_milliseconds{rule_type="alerting",quantile="0.5"} 0
prometheus_rule_evaluation_duration_milliseconds{rule_type="alerting",quantile="0.9"} 0
prometheus_rule_evaluation_duration_milliseconds{rule_type="alerting",quantile="0.99"} 2
prometheus_rule_evaluation_duration_milliseconds_sum{rule_type="alerting"} 12
prometheus_rule_evaluation_duration_milliseconds_count{rule_type="alerting"} 115
prometheus_rule_evaluation_duration_milliseconds{rule_type="recording",quantile="0.5"} 0
prometheus_rule_evaluation_duration_milliseconds{rule_type="recording",quantile="0.9"} 0
prometheus_rule_evaluation_duration_milliseconds{rule_type="recording",quantile="0.99"} 3
prometheus_rule_evaluation_duration_milliseconds_sum{rule_type="recording"} 15
prometheus_rule_evaluation_duration_milliseconds_count{rule_type="recording"} 115
# HELP prometheus_rule_evaluation_failures_total The total number of rule evaluation failures.
# TYPE prometheus_rule_evaluation_failures_total counter
prometheus_rule_evaluation_failures_total 0
# HELP prometheus_samples_queue_capacity Capacity of the queue for unwritten samples.
# TYPE prometheus_samples_queue_capacity gauge
prometheus_samples_queue_capacity 4096
# HELP prometheus_samples_queue_length Current number of items in the queue for unwritten samples. Each item comprises all samples exposed by one target as one metric family (i.e. metrics of the same name).
# TYPE prometheus_samples_queue_length gauge
prometheus_samples_queue_length 0
# HELP prometheus_target_interval_length_seconds Actual intervals between scrapes.
# TYPE prometheus_target_interval_length_seconds summary
prometheus_target_interval_length_seconds{interval="15s",quantile="0.01"} 14
prometheus_target_interval_length_seconds{interval="15s",quantile="0.05"} 14
prometheus_target_interval_length_seconds{interval="15s",quantile="0.5"} 15
prometheus_target_interval_length_seconds{interval="15s",quantile="0.9"} 15
prometheus_target_interval_length_seconds{interval="15s",quantile="0.99"} 15
prometheus_target_interval_length_seconds_sum{interval="15s"} 175
prometheus_target_interval_length_seconds_count{interval="15s"} 12
prometheus_target_interval_length_seconds{interval="1s",quantile="0.01"} 0
prometheus_target_interval_length_seconds{interval="1s",quantile="0.05"} 0
prometheus_target_interval_length_seconds{interval="1s",quantile="0.5"} 0
prometheus_target_interval_length_seconds{interval="1s",quantile="0.9"} 1
prometheus_target_interval_length_seconds{interval="1s",quantile="0.99"} 1
prometheus_target_interval_length_seconds_sum{interval="1s"} 55
prometheus_target_interval_length_seconds_count{interval="1s"} 117
`
