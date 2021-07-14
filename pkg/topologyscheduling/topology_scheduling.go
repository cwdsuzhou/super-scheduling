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

package topologyscheduling

import (
	"context"
	"fmt"
	"sync/atomic"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/helper"

	"github.com/cwdsuzhou/super-scheduling/pkg/apis/config"
	"github.com/cwdsuzhou/super-scheduling/pkg/apis/scheduling/v1alpha1"
	"github.com/cwdsuzhou/super-scheduling/pkg/generated/clientset/versioned"
	schedinformer "github.com/cwdsuzhou/super-scheduling/pkg/generated/informers/externalversions"
	externalv1alpha1 "github.com/cwdsuzhou/super-scheduling/pkg/generated/listers/scheduling/v1alpha1"
)

const (
	// ErrReasonConstraintsNotMatch is used for PodTopologySpread filter error.
	ErrReasonConstraintsNotMatch = "node(s) didn't match pod topology spread constraints"
	// ErrReasonNodeLabelNotMatch is used when the node doesn't hold the required label.
	ErrReasonNodeLabelNotMatch = ErrReasonConstraintsNotMatch + " (missing required label)"

	topologySchedulingLabelKey = "topology-scheduling-policy.scheduling.sigs.k8s.io"
)

// TopologyScheduling is a plugin that implements the mechanism of capacity scheduling.
type TopologyScheduling struct {
	frameworkHandle      framework.Handle
	podLister            corelisters.PodLister
	topologyPolicyLister externalv1alpha1.TopologySchedulingPolicyLister
}

type topologyPair struct {
	key   string
	value string
}

// topologySchedulingConstraint is an internal version for v1.TopologySpreadConstraint
// and where the selector is parsed.
// Fields are exported for comparison during testing.
type topologySchedulingConstraint struct {
	TopologyKey    string
	SchedulePolicy map[string]int32
	Selector       labels.Selector
}

// PreFilterState computed at PreFilter and used at PostFilter or Reserve.
type PreFilterState struct {
	Constraint topologySchedulingConstraint
	// TpPairToMatchNum is keyed with topologyPair, and valued with the number of matching pods.
	TpPairToMatchNum map[topologyPair]*int32
}

func (s *PreFilterState) updateWithPod(updatedPod, preemptorPod *v1.Pod, node *v1.Node, delta int32) {
	if s == nil || updatedPod.Namespace != preemptorPod.Namespace || node == nil {
		return
	}
	if _, ok := node.Labels[s.Constraint.TopologyKey]; !ok {
		return
	}

	podLabelSet := labels.Set(updatedPod.Labels)
	if !s.Constraint.Selector.Matches(podLabelSet) {
		return
	}

	k, v := s.Constraint.TopologyKey, node.Labels[s.Constraint.TopologyKey]
	pair := topologyPair{key: k, value: v}
	*s.TpPairToMatchNum[pair] += delta
}

// Clone the preFilter state.
func (s *PreFilterState) Clone() framework.StateData {
	if s == nil {
		return nil
	}
	stateCopy := PreFilterState{
		// Constraints are shared because they don't change.
		Constraint:       s.Constraint,
		TpPairToMatchNum: make(map[topologyPair]*int32, len(s.TpPairToMatchNum)),
	}
	for tpPair, matchNum := range s.TpPairToMatchNum {
		copyPair := topologyPair{key: tpPair.key, value: tpPair.value}
		copyCount := *matchNum
		stateCopy.TpPairToMatchNum[copyPair] = &copyCount
	}
	return &stateCopy
}

// Prefilter build cache for scheduling cycle
// Filter Checks node filter.
// Score the nodes according to topology
// Reserve reserves the node/pod to cache, if failed run unreserve to clean up cache
var _ framework.PreFilterPlugin = &TopologyScheduling{}
var _ framework.FilterPlugin = &TopologyScheduling{}
var _ framework.ScorePlugin = &TopologyScheduling{}
var _ framework.ReservePlugin = &TopologyScheduling{}

const (
	// Name is the name of the plugin used in Registry and configurations.
	Name = "TopologyScheduling"

	// preFilterStateKey is the key in CycleState to NodeResourcesFit pre-computed data.
	preFilterStateKey = "PreFilter" + Name
)

// Name returns name of the plugin. It is used in logs, etc.
func (ts *TopologyScheduling) Name() string {
	return Name
}

// New initializes a new plugin and returns it.
func New(obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	args, ok := obj.(*config.SchedulingArgs)
	if !ok {
		return nil, fmt.Errorf("want args to be of type CapacitySchedulingArgs, got %T", obj)
	}
	kubeConfigPath := args.KubeConfigPath

	c := &TopologyScheduling{
		frameworkHandle: handle,
		podLister:       handle.SharedInformerFactory().Core().V1().Pods().Lister(),
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}
	client, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	schedSharedInformerFactory := schedinformer.NewSharedInformerFactory(client, 0)
	c.topologyPolicyLister = schedSharedInformerFactory.Scheduling().V1alpha1().TopologySchedulingPolicies().Lister()
	topologyPolicyInformer := schedSharedInformerFactory.Scheduling().V1alpha1().TopologySchedulingPolicies().Informer()

	schedSharedInformerFactory.Start(nil)
	if !cache.WaitForCacheSync(nil, topologyPolicyInformer.HasSynced) {
		return nil, fmt.Errorf("timed out waiting for caches to sync %v", Name)
	}
	klog.Infof("TopologyScheduling start")
	return c, nil
}

// PreFilter performs the following validations.
// 1. Check if the (pod.request + eq.allocated) is less than eq.max.
// 2. Check if the sum(eq's usage) > sum(eq's min).
func (ts *TopologyScheduling) PreFilter(ctx context.Context, state *framework.CycleState,
	pod *v1.Pod) *framework.Status {
	s, err := ts.calculatePreFilterStateCache(pod)
	if err != nil {
		return framework.AsStatus(err)
	}
	state.Write(preFilterStateKey, s)
	return nil
}

// getPreFilterState fetches a pre-computed preFilterState.
func getPreFilterState(cycleState *framework.CycleState) (*PreFilterState, error) {
	c, err := cycleState.Read(preFilterStateKey)
	if err != nil {
		// preFilterState doesn't exist, likely PreFilter wasn't invoked.
		return nil, fmt.Errorf("reading %q from cycleState: %v", preFilterStateKey, err)
	}

	s, ok := c.(*PreFilterState)
	if !ok {
		return nil, fmt.Errorf("%+v convert to podtopologyspread.preFilterState error", c)
	}
	return s, nil
}

func (ts *TopologyScheduling) calculatePreFilterStateCache(pod *v1.Pod) (*PreFilterState, error) {
	policyName, ok := pod.Labels[topologySchedulingLabelKey]
	if !ok {
		return nil, nil
	}

	tsp, err := ts.topologyPolicyLister.TopologySchedulingPolicies(pod.Namespace).Get(policyName)
	if err != nil {
		return nil, fmt.Errorf("obtaining pod's topology policy: %v", err)
	}
	allNodes, err := ts.frameworkHandle.SnapshotSharedLister().NodeInfos().List()
	if err != nil {
		return nil, fmt.Errorf("listing NodeInfos: %w", err)
	}
	for _, n := range allNodes {
		node := n.Node()
		if node == nil {
			klog.Error("node not found")
			continue
		}
		// In accordance to design, if NodeAffinity or NodeSelector is defined,
		// spreading is applied to nodes that pass those filters.
		if !helper.PodMatchesNodeSelectorAndAffinityTerms(pod, node) {
			continue
		}
		// check node labels
		if _, ok := node.Labels[tsp.Spec.TopologyKey]; !ok {
			continue
		}
		continue
	}
	selector, err := metav1.LabelSelectorAsSelector(tsp.Spec.LabelSelector)
	if err != nil {
		return nil, err
	}
	constraint := topologySchedulingConstraint{
		TopologyKey:    tsp.Spec.TopologyKey,
		SchedulePolicy: covertPolicy(tsp.Spec.DeployPlacement),
		Selector:       selector,
	}
	s := PreFilterState{
		Constraint:       constraint,
		TpPairToMatchNum: make(map[topologyPair]*int32),
	}
	processNode := func(i int) {
		nodeInfo := allNodes[i]
		node := nodeInfo.Node()

		pair := topologyPair{key: tsp.Spec.TopologyKey, value: node.Labels[tsp.Spec.TopologyKey]}
		tpCount := s.TpPairToMatchNum[pair]
		if tpCount == nil {
			return
		}
		count := countPodsMatchSelector(nodeInfo.Pods, s.Constraint.Selector, pod.Namespace)
		atomic.AddInt32(tpCount, int32(count))
	}
	workqueue.ParallelizeUntil(context.Background(), 32, len(allNodes), processNode)
	return &s, nil
}

func (ts *TopologyScheduling) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod,
	nodeInfo *framework.NodeInfo) *framework.Status {
	node := nodeInfo.Node()
	if node == nil {
		return framework.AsStatus(fmt.Errorf("node not found"))
	}

	s, err := getPreFilterState(state)
	if err != nil {
		return framework.AsStatus(err)
	}

	policyName, ok := pod.Labels[topologySchedulingLabelKey]
	if !ok {
		return nil
	}

	tsp, err := ts.topologyPolicyLister.TopologySchedulingPolicies(pod.Namespace).Get(policyName)
	if err != nil {
		return framework.AsStatus(fmt.Errorf("obtaining pod's topology policy: %v", err))
	}

	podLabelSet := labels.Set(pod.Labels)
	if !s.Constraint.Selector.Matches(podLabelSet) {
		return framework.NewStatus(framework.Unschedulable, ErrReasonConstraintsNotMatch)
	}
	tpKey := tsp.Spec.TopologyKey
	tpVal, ok := node.Labels[tpKey]
	if !ok {
		klog.V(5).Infof("node '%s' doesn't have required label '%s'", node.Name, tpKey)
		return framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonNodeLabelNotMatch)
	}

	if checkPolicyRequirementReached(tpKey,
		s.Constraint.SchedulePolicy, s.TpPairToMatchNum) {
		return nil
	}

	if !checkTopoValueReached(tpKey, tpVal, s.Constraint.SchedulePolicy, s.TpPairToMatchNum) {
		return nil

	}
	return framework.NewStatus(framework.Unschedulable, ErrReasonConstraintsNotMatch)
}

// PreFilterExtensions returns prefilter extensions, pod add and remove.
func (ts *TopologyScheduling) PreFilterExtensions() framework.PreFilterExtensions {
	return ts
}

// AddPod from pre-computed data in cycleState.
func (ts *TopologyScheduling) AddPod(ctx context.Context, cycleState *framework.CycleState, podToSchedule *v1.Pod,
	podToAdd *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	s, err := getPreFilterState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}

	s.updateWithPod(podToAdd, podToSchedule, nodeInfo.Node(), 1)

	return framework.NewStatus(framework.Success, "")
}

// RemovePod from pre-computed data in cycleState.
func (ts *TopologyScheduling) RemovePod(ctx context.Context, cycleState *framework.CycleState, podToSchedule *v1.Pod,
	podToRemove *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	s, err := getPreFilterState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}

	s.updateWithPod(podToRemove, podToSchedule, nodeInfo.Node(), -1)

	return framework.NewStatus(framework.Success, "")
}

func (ts *TopologyScheduling) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod,
	nodeName string) (int64, *framework.Status) {
	s, err := getPreFilterState(state)
	if err != nil {
		return 0, framework.NewStatus(framework.Success, "")
	}
	node, err := ts.frameworkHandle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, "node not exist")
	}
	// check node labels
	tpval, ok := node.Node().Labels[s.Constraint.TopologyKey]
	if !ok {
		return 0, framework.NewStatus(framework.Success, "")
	}
	pair := topologyPair{
		key:   s.Constraint.TopologyKey,
		value: tpval,
	}
	desired, ok := s.Constraint.SchedulePolicy[tpval]
	if !ok {
		return 0, framework.NewStatus(framework.Success, "")
	}
	count, ok := s.TpPairToMatchNum[pair]
	if count == nil || !ok {
		return 0, framework.NewStatus(framework.Success, "")
	}
	if *count < desired {
		return 0, framework.NewStatus(framework.Success, "")
	}
	return int64(float64(desired)/float64(*count+1-desired))*1000 + int64(desired),
		framework.NewStatus(framework.Success, "")
}

func (ts *TopologyScheduling) ScoreExtensions() framework.ScoreExtensions {
	return ts
}

func (ts *TopologyScheduling) NormalizeScore(ctx context.Context, state *framework.CycleState, p *v1.Pod,
	scores framework.NodeScoreList) *framework.Status {
	return framework.NewStatus(framework.Success)
}

func (ts *TopologyScheduling) Reserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod,
	nodeName string) *framework.Status {
	s, err := getPreFilterState(state)
	if err != nil {
		return framework.AsStatus(err)
	}
	node, err := ts.frameworkHandle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return framework.AsStatus(err)
	}
	s.updateWithPod(pod, pod, node.Node(), 1)
	return framework.NewStatus(framework.Success, "")
}

func (ts *TopologyScheduling) Unreserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	s, err := getPreFilterState(state)
	if err != nil {
		return
	}
	node, err := ts.frameworkHandle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return
	}
	s.updateWithPod(pod, pod, node.Node(), -1)
}

func countPodsMatchSelector(podInfos []*framework.PodInfo, selector labels.Selector, ns string) int {
	count := 0
	for _, p := range podInfos {
		if p == nil {
			continue
		}
		pod := p.Pod
		// Bypass terminating Pod (see #87621).
		if pod.DeletionTimestamp != nil || pod.Namespace != ns {
			continue
		}
		if selector.Matches(labels.Set(pod.Labels)) {
			count++
		}
	}
	return count
}

func covertPolicy(dp []v1alpha1.SchedulePolicy) map[string]int32 {
	policy := make(map[string]int32)
	for _, p := range dp {
		policy[p.Name] = p.Replicas
	}
	return policy
}

func checkPolicyRequirementReached(topoKey string, policy map[string]int32,
	tpPairToMatchNum map[topologyPair]*int32) bool {
	for value, replicas := range policy {
		pair := topologyPair{
			key:   topoKey,
			value: value,
		}
		count, ok := tpPairToMatchNum[pair]
		if count == nil || !ok {
			return false
		}
		if *count < replicas {
			return false
		}
	}
	return true
}

func checkTopoValueReached(topoKey, topoValue string, policy map[string]int32,
	tpPairToMatchNum map[topologyPair]*int32) bool {
	pair := topologyPair{
		key:   topoKey,
		value: topoValue,
	}
	desired, ok := policy[topoValue]
	if !ok {
		return true
	}
	count, ok := tpPairToMatchNum[pair]
	if count == nil || !ok {
		return false
	}
	if *count < desired {
		return false
	}
	return true
}
