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
	"fmt"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"

	kube_api "k8s.io/client-go/pkg/api/v1"
	. "k8s.io/heapster/metrics/core"
	"time"
)

const MetricNamespaceLabel = "namespace_name"
const MetricPodLabel = "pod_name"

const ScrapeEnabledAnnotationKey = "metro.fid/scrape"
const ScrapePathAnnotationKey = "metro.fid/scrape-path"
const ScrapePortAnnotationKey = "metro.fid/scrape-port"
const DefaultPodMetricsPath = "/metrics"
const DefaultPodMetricsPort int32 = 4001

var (
	// The Kubelet request latencies in microseconds.
	kubeletRequestLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "heapster",
			Subsystem: "pod",
			Name:      "request_duration_microseconds",
			Help:      "The Pod request latencies in microseconds.",
		},
		[]string{"pod"},
	)
)

func init() {
	prometheus.MustRegister(kubeletRequestLatency)
}

type podMetricsSource struct {
	pod           *kube_api.Pod
	promClient    PrometheusClient
	podConfig     *podConfig
	scrapeTimeout time.Duration
}

func NewPodMetricsSource(pod *kube_api.Pod, podConfig *podConfig, promClient PrometheusClient) MetricsSource {
	src := &podMetricsSource{
		pod:        pod,
		podConfig:  podConfig,
		promClient: promClient,
	}

	return src
}

func (s *podMetricsSource) Name() string {
	return PodKey(s.pod.Namespace, s.pod.Name)
}

func (s *podMetricsSource) ScrapeMetrics(start, end time.Time) *DataBatch {
	vector, err := s.scrapeMetrics(start, end)
	if err != nil {
		glog.Errorf("%v", err)
		return &DataBatch{}
	}

	mSet := convertVectorToMetricSetMap(vector, s.pod.Status.StartTime.Time)

	return convertVectorToDataBatch(end, mSet)
}

func (s *podMetricsSource) scrapeMetrics(start, end time.Time) (*model.Vector, error) {
	startTime := time.Now()
	defer kubeletRequestLatency.WithLabelValues(s.podConfig.hostname).Observe(float64(time.Since(startTime)))
	return s.promClient.GetPodMetrics(s.pod, s.podConfig.ip, s.podConfig.port, s.podConfig.path, end)
}

func convertVectorToDataBatch(ts time.Time, mSet map[string]*MetricSet) *DataBatch {
	return &DataBatch{
		Timestamp:  ts,
		MetricSets: mSet,
	}
}
func convertVectorToMetricSetMap(vector *model.Vector, createTime time.Time) map[string]*MetricSet {
	mSets := map[string]*MetricSet{}

	for _, sample := range *vector {
		name, podName, namespace, err := popMetaLabels(sample)
		if err != nil {
			glog.Errorf("Failed to identify metric source: %v", err)
			continue
		}

		var set *MetricSet
		var found bool
		key := PodKey(namespace, podName)
		if set, found = mSets[key]; !found {
			set = &MetricSet{
				CreateTime:   createTime,
				ScrapeTime:   sample.Timestamp.Time(),
				MetricValues: map[string]MetricValue{},
				Labels: map[string]string{
					LabelPodName.Key:       podName,
					LabelNamespaceName.Key: namespace,
				},
				LabeledMetrics: []LabeledMetric{},
			}
			mSets[key] = set
		}

		isLabeledMetric := len(sample.Metric) != 0

		if isLabeledMetric {
			set.LabeledMetrics = append(set.LabeledMetrics, LabeledMetric{
				Name:   name,
				Labels: convertSampleMetricToLabels(sample.Metric),
				// sample.Value is of float64
				MetricValue: convertSampleToMetricValue(sample),
			})
		} else {
			set.MetricValues[name] = convertSampleToMetricValue(sample)
		}
	}

	return mSets
}

func convertSampleToMetricValue(sample *model.Sample) MetricValue {
	return MetricValue{
		FloatValue: float32(sample.Value),
		ValueType:  ValueFloat,
		// TODO check if it is fine
		MetricType: MetricGauge,
	}
}

func convertSampleMetricToLabels(metric model.Metric) map[string]string {
	ret := map[string]string{}

	for k, v := range metric {
		ret[string(k)] = string(v)
	}

	return ret
}

func popMetaLabels(sample *model.Sample) (name, podName, namespace string, err error) {
	var found bool
	var nameLV, podNameLV, namespaceLV model.LabelValue

	if nameLV, found = sample.Metric[model.MetricNameLabel]; !found {
		err = fmt.Errorf("Required label value not found: %s", model.MetricNameLabel)
		return
	}
	if podNameLV, found = sample.Metric[MetricPodLabel]; !found {
		err = fmt.Errorf("Required label value not found: %s", MetricPodLabel)
		return
	}
	if namespaceLV, found = sample.Metric[MetricNamespaceLabel]; !found {
		err = fmt.Errorf("Required label value not found: %s", MetricNamespaceLabel)
		return
	}

	delete(sample.Metric, model.MetricNameLabel)
	delete(sample.Metric, MetricPodLabel)
	delete(sample.Metric, MetricNamespaceLabel)

	name = string(nameLV)
	podName = string(podNameLV)
	namespace = string(namespaceLV)

	return
}
