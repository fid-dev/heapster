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

	v1listers "k8s.io/client-go/listers/core/v1"
	kube_api "k8s.io/client-go/pkg/api/v1"
	. "k8s.io/heapster/metrics/core"
	"net"
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
	nodeLister    v1listers.NodeLister
	scrapeTimeout time.Duration
}

func NewPodMetricsSource(pod *kube_api.Pod, podConfig *podConfig, nodeLister v1listers.NodeLister, promClient PrometheusClient) MetricsSource {
	src := &podMetricsSource{
		pod:        pod,
		podConfig:  podConfig,
		nodeLister: nodeLister,
		promClient: promClient,
	}

	return src
}

func (s *podMetricsSource) Name() string {
	return s.String()
}

func (s *podMetricsSource) String() string {
	return fmt.Sprintf("pod/%s", PodKey(s.pod.Namespace, s.pod.Name))
}

func (s *podMetricsSource) ScrapeMetrics(start, end time.Time) *DataBatch {
	vector, err := s.scrapeMetrics(start, end)
	if err != nil {
		glog.Errorf("Failed scaping metrics: %s\n%v", s.Name(), err)
		return &DataBatch{}
	}

	mSet := convertVectorToMetricSetMap(vector, s.pod, s.nodeLister)

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
func convertVectorToMetricSetMap(vector *model.Vector, pod *kube_api.Pod, nodeLister v1listers.NodeLister) map[string]*MetricSet {
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
			node, err := nodeLister.Get(pod.Spec.NodeName)
			if err != nil {
				glog.Error(err)
				continue
			}

			nodeHostname, _, err := getNodeHostnameAndIP(node)
			if err != nil {
				glog.Error(err)
				continue
			}

			set = &MetricSet{
				CreateTime:   pod.Status.StartTime.Time,
				ScrapeTime:   sample.Timestamp.Time(),
				MetricValues: map[string]MetricValue{},
				Labels: map[string]string{
					LabelPodName.Key:       podName,
					LabelNamespaceName.Key: namespace,
					LabelMetricSetType.Key: MetricSetTypePod,
					LabelHostname.Key:      nodeHostname,
					LabelNodename.Key:      node.Name,
					LabelHostID.Key:        node.Spec.ExternalID,
					LabelPodId.Key:         string(pod.UID),
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

func getNodeHostnameAndIP(node *kube_api.Node) (string, string, error) {
	for _, c := range node.Status.Conditions {
		if c.Type == kube_api.NodeReady && c.Status != kube_api.ConditionTrue {
			return "", "", fmt.Errorf("Node %v is not ready", node.Name)
		}
	}
	hostname, ip := node.Name, ""
	for _, addr := range node.Status.Addresses {
		if addr.Type == kube_api.NodeHostName && addr.Address != "" {
			hostname = addr.Address
		}
		if addr.Type == kube_api.NodeInternalIP && addr.Address != "" {
			if net.ParseIP(addr.Address).To4() != nil {
				ip = addr.Address
			}
		}
		if addr.Type == kube_api.NodeLegacyHostIP && addr.Address != "" && ip == "" {
			ip = addr.Address
		}
	}
	if ip != "" {
		return hostname, ip, nil
	}
	return "", "", fmt.Errorf("Node %v has no valid hostname and/or IP address: %v %v", node.Name, hostname, ip)
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
