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
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"io"
	"io/ioutil"
	kube_api "k8s.io/client-go/pkg/api/v1"
	"k8s.io/heapster/version"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const acceptHeader = `application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.7,text/plain;version=0.0.4;q=0.3,*/*;q=0.1`

var userAgentHeader = fmt.Sprintf("Heapster/%s", version.HeapsterVersion)

type PrometheusClient interface {
	GetPodMetrics(pod *kube_api.Pod, ip string, port int32, path string, t time.Time) (*model.Vector, error)
}

type promClient struct {
	httpClient *http.Client
}

func NewPromClient(httpClient *http.Client) *promClient {
	return &promClient{
		httpClient: httpClient,
	}
}

func (p *promClient) GetPodMetrics(pod *kube_api.Pod, ip string, port int32, path string, t time.Time) (*model.Vector, error) {
	uri := p.url(pod, ip, port, path)

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("HTTP Error while creating request for '%s': \n%v", uri.String(), err)
	}

	req.Header.Add("Accept", acceptHeader)
	req.Header.Set("User-Agent", userAgentHeader)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP Error while calling '%s': \n%v", uri.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected response code: %d", resp.StatusCode)
	}

	var (
		allSamples = make(model.Vector, 0, 200)
		decSamples = make(model.Vector, 0, 5)
	)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	sdec := expfmt.SampleDecoder{
		Dec: expfmt.NewDecoder(strings.NewReader(string(body)), expfmt.ResponseFormat(resp.Header)),
		Opts: &expfmt.DecodeOptions{
			Timestamp: model.TimeFromUnixNano(t.UnixNano()),
		},
	}

	for {
		if err = sdec.Decode(&decSamples); err != nil {
			break
		}

		addPodLabels(&decSamples, pod)

		//glog.Infof("Sample: %v", decSamples)
		allSamples = append(allSamples, decSamples...)
		decSamples = decSamples[:0]
	}

	if err == io.EOF {
		// Set err to nil since it is used in the scrape health recording.
		err = nil
	}

	return &allSamples, err
}

func (p *promClient) url(pod *kube_api.Pod, ip string, port int32, path string) *url.URL {
	return &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", ip, port),
		Path:   path,
	}
}

func addPodLabels(samples *model.Vector, pod *kube_api.Pod) {
	for _, sample := range *samples {
		sample.Metric[model.LabelName("pod_name")] = model.LabelValue(pod.Name)
		sample.Metric[model.LabelName("namespace_name")] = model.LabelValue(pod.Namespace)
	}
}
