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
	"net/url"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	kube_client "k8s.io/client-go/kubernetes"
	v1listers "k8s.io/client-go/listers/core/v1"
	kube_api "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	. "k8s.io/heapster/metrics/core"
	"net/http"
	"strconv"
	"time"
)

type podProvider struct {
	podLister  v1listers.PodLister
	reflector  *cache.Reflector
	httpClient *http.Client
}

type podConfig struct {
	ip       string
	hostname string
	port     int32
	path     string
}

func NewPodProvider(uri *url.URL) (MetricsSourceProvider, error) {
	// create clients
	kubeConfig, err := GetPodConfigs(uri)
	if err != nil {
		return nil, err
	}
	kubeClient := kube_client.NewForConfigOrDie(kubeConfig)

	// Get nodes to test if the client is configured well. Watch gives less error information.
	if _, err := kubeClient.Nodes().List(metav1.ListOptions{}); err != nil {
		glog.Errorf("Failed to load pods: %v", err)
	}

	// watch pods
	nodeLister, reflector, _ := GetPodLister(kubeClient)

	return &podProvider{
		podLister:  nodeLister,
		reflector:  reflector,
		httpClient: http.DefaultClient,
	}, nil
}

func (p *podProvider) GetMetricsSources() []MetricsSource {
	sources := []MetricsSource{}
	pods, err := p.podLister.List(labels.Everything())
	if err != nil {
		glog.Errorf("error while listing pods: %v", err)
		return sources
	}

	pods = filterAnnotatedPods(pods)

	if len(pods) == 0 {
		glog.Error("No pods received from APIserver.")
		return sources
	}

	for _, pod := range pods {
		podConfig, err := getPodConfig(pod)
		if err != nil {
			glog.Errorf("%v", err)
			continue
		}
		sources = append(sources, NewPodMetricsSource(
			pod,
			podConfig,
			NewPromClient(http.DefaultClient),
		))
	}
	return sources
}

func filterAnnotatedPods(pods []*kube_api.Pod) []*kube_api.Pod {
	annotatedPods := []*kube_api.Pod{}

	for _, pod := range pods {
		if val, found := pod.GetAnnotations()[ScrapeEnabledAnnotationKey]; !found || val != "true" {
			// Scrape not enabled, nothing to do
			glog.V(6).Infof("Skip scraping metrics for Namespace: %s; Pod: %s", pod.Namespace, pod.Name)
			continue
		}

		annotatedPods = append(annotatedPods, pod)
	}

	return annotatedPods
}

func getPodConfig(pod *kube_api.Pod) (*podConfig, error) {
	a := pod.GetAnnotations()

	path := DefaultPodMetricsPath
	port := DefaultPodMetricsPort

	if podPort, found := a[ScrapePathAnnotationKey]; found {
		var err error
		var i64 int64
		if i64, err = strconv.ParseInt(podPort, 10, 32); err != nil {
			return nil, fmt.Errorf("Failed converting port to int: '%s': \n%v", podPort, err)
		} else {
			port = int32(i64)
		}

	} else if len(pod.Spec.Containers) > 0 && len(pod.Spec.Containers[0].Ports) > 0 {
		if pod.Spec.Containers[0].Ports[0].ContainerPort > 0 {
			port = pod.Spec.Containers[0].Ports[0].ContainerPort
		}
	}

	if podPath, found := a[ScrapePortAnnotationKey]; found {
		path = podPath
	}

	return &podConfig{
		ip:       pod.Status.PodIP,
		hostname: pod.Name,
		port:     port,
		path:     path,
	}, nil
}

func GetPodLister(kubeClient *kube_client.Clientset) (v1listers.PodLister, *cache.Reflector, error) {
	lw := cache.NewListWatchFromClient(kubeClient.Core().RESTClient(), "pods", kube_api.NamespaceAll, fields.Everything())
	store := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	podLister := v1listers.NewPodLister(store)
	reflector := cache.NewReflector(lw, &kube_api.Pod{}, store, time.Hour)
	reflector.Run()

	return podLister, reflector, nil
}
