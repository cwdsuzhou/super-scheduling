/*
Copyright 2020 The Kubernetes Authors.

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

package strategies

import (
	"context"
	"sort"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/descheduler/pkg/api"
	"sigs.k8s.io/descheduler/pkg/descheduler/evictions"

	"github.com/cwdsuzhou/super-scheduling/pkg/common"
	"github.com/cwdsuzhou/super-scheduling/pkg/generated/listers/scheduling/v1alpha1"
	"github.com/cwdsuzhou/super-scheduling/pkg/util"
)

type TopoPairCount struct {
	count int32
	pods  []*v1.Pod
}

func RemovePodsViolatingTopologySchedulingPolicy(
	ctx context.Context,
	client clientset.Interface,
	strategy api.DeschedulerStrategy,
	nodes []*v1.Node,
	podEvictor *evictions.PodEvictor,
	tspLister v1alpha1.TopologySchedulingPolicyLister,
) {
	if tspLister == nil {
		return
	}
	nodeMap := make(map[string]*v1.Node)
	for _, node := range nodes {
		nodeMap[node.Name] = node
	}
	tsps, err := tspLister.TopologySchedulingPolicies(v1.NamespaceAll).List(labels.Everything())
	if err != nil {
		klog.Error(err)
		return
	}
	for _, tsp := range tsps {
		selector, err := metav1.LabelSelectorAsSelector(tsp.Spec.LabelSelector)
		if err != nil {
			klog.Error(err)
			continue
		}
		constraint := common.TopologySchedulingConstraint{
			TopologyKey:    tsp.Spec.TopologyKey,
			SchedulePolicy: util.CovertPolicy(tsp.Spec.DeployPlacement),
			Selector:       selector,
		}
		TpPairToMatchNum := make(map[common.TopologyPair]*TopoPairCount)
		pods, err := client.CoreV1().Pods(tsp.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(tsp.Spec.LabelSelector),
		})
		if err != nil {
			klog.Error(err)
			continue
		}
		for _, p := range pods.Items {
			pod := p.DeepCopy()
			if pod.DeletionTimestamp != nil {
				continue
			}
			node := nodeMap[pod.Spec.NodeName]
			if node == nil {
				continue
			}
			pair := common.TopologyPair{Key: tsp.Spec.TopologyKey, Value: node.Labels[tsp.Spec.TopologyKey]}
			counter := TpPairToMatchNum[pair]
			if counter == nil {
				counter = &TopoPairCount{
					count: 1,
					pods:  []*v1.Pod{pod},
				}
				TpPairToMatchNum[pair] = counter
			} else {
				counter.count++
				counter.pods = append(counter.pods, pod)
			}
		}
		if checkTopoReached(TpPairToMatchNum, constraint) {
			continue
		}
		for pair, counter := range TpPairToMatchNum {
			diff := constraint.SchedulePolicy[pair.Value] - counter.count
			if diff >= 0 {
				continue
			}
			// sort pods
			toDeleteCount := -diff
			sort.Sort(ByCostom(counter.pods))
			for i := 0; i < int(toDeleteCount); i++ {
				podEvictor.EvictPod(ctx, counter.pods[i], nodeMap[counter.pods[i].Spec.NodeName],
					"Exceeded the policy desired")
			}
		}
	}
}

func checkTopoReached(pair map[common.TopologyPair]*TopoPairCount,
	constraint common.TopologySchedulingConstraint) bool {
	satisfied := true
	notSatisfied := false
	for pair, counter := range pair {
		diff := constraint.SchedulePolicy[pair.Value] - counter.count
		if diff > 0 {
			notSatisfied = true
			continue
		} else if diff < 0 {
			satisfied = true
		}
	}
	// some keys exceed, some keys not exceed => not reached
	// all keys exceed => reached
	// all key not exceed => not reached
	if satisfied && !notSatisfied {
		return true
	}
	return false
}

type ByCostom []*v1.Pod

var _ sort.Interface = &ByCostom{}

func (p ByCostom) Len() int {
	return len(p)
}

func (p ByCostom) Less(i, j int) bool {
	if p[i].Status.Phase == v1.PodFailed || p[i].Status.Phase == v1.PodSucceeded {
		return true
	}
	if p[i].Status.Phase == v1.PodPending {
		return true
	}
	if p[j].Status.Phase == v1.PodFailed || p[j].Status.Phase == v1.PodSucceeded {
		return false
	}
	if p[j].Status.Phase == v1.PodPending {
		return false
	}
	return p[i].Name < p[j].Name
}

func (p ByCostom) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
