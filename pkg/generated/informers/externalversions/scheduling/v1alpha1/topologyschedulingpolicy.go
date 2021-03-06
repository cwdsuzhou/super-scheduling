/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	schedulingv1alpha1 "github.com/cwdsuzhou/super-scheduling/pkg/apis/scheduling/v1alpha1"
	versioned "github.com/cwdsuzhou/super-scheduling/pkg/generated/clientset/versioned"
	internalinterfaces "github.com/cwdsuzhou/super-scheduling/pkg/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/cwdsuzhou/super-scheduling/pkg/generated/listers/scheduling/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// TopologySchedulingPolicyInformer provides access to a shared informer and lister for
// TopologySchedulingPolicies.
type TopologySchedulingPolicyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.TopologySchedulingPolicyLister
}

type topologySchedulingPolicyInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewTopologySchedulingPolicyInformer constructs a new informer for TopologySchedulingPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewTopologySchedulingPolicyInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredTopologySchedulingPolicyInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredTopologySchedulingPolicyInformer constructs a new informer for TopologySchedulingPolicy type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredTopologySchedulingPolicyInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.SchedulingV1alpha1().TopologySchedulingPolicies(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.SchedulingV1alpha1().TopologySchedulingPolicies(namespace).Watch(context.TODO(), options)
			},
		},
		&schedulingv1alpha1.TopologySchedulingPolicy{},
		resyncPeriod,
		indexers,
	)
}

func (f *topologySchedulingPolicyInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredTopologySchedulingPolicyInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *topologySchedulingPolicyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&schedulingv1alpha1.TopologySchedulingPolicy{}, f.defaultInformer)
}

func (f *topologySchedulingPolicyInformer) Lister() v1alpha1.TopologySchedulingPolicyLister {
	return v1alpha1.NewTopologySchedulingPolicyLister(f.Informer().GetIndexer())
}
