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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TopologySchedulingPolicy describes multi clusters/regions/zones scheduling policy
type TopologySchedulingPolicy struct {
	metav1.TypeMeta

	// Standard object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the multi cluster.
	// +optional
	Spec TopologySchedulingPolicySpec `json:"spec,omitempty"`

	// Status represents the current information about multi cluster.
	// This data may not be up to date.
	// +optional
	Status TopologySchedulingPolicyStatus `json:"status,omitempty"`
}

type TopologySchedulingPolicySpec struct {
	// TopologyKey is used when match node topoKey
	TopologyKey string `json:"topologyKey"`

	// ScheduleStrategy is used when schedule pods for a topoKey.
	// It can be Fill or Balance.
	// Fill will schedule pods to satisfy a topo requirements first, then start another.
	// Balance will try to keep Balance across different topo when scheduling pods.
	ScheduleStrategy string `json:"scheduleStrategy"`

	// DeployPlacement describes the topology requirement when scheduling pods.
	DeployPlacement []SchedulePolicy `json:"deployPlacement"`

	// UpdatePlacement describes topology requirements when updating pods.
	// Notice: This is not implemented now.
	UpdatePlacement []UpdatePolicy `json:"updatePlacement"`

	// LabelSelector is used to find matching pods.
	// Pods that match this label selector are counted to determine the number of pods
	// in their corresponding topology domain.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector"`
}

type DeployPlacement struct {
	WhenUnsatisfiable UnsatisfiableConstraintAction `json:"whenUnsatisfiable"`
	Policy            []SchedulePolicy              `json:"policy"`
}

const (
	ScheduleStrategyFill    = "Fill"
	ScheduleStrategyBalance = "Balance"
)

type UnsatisfiableConstraintAction string

const (
	// DoNotSchedule instructs the multicluster not to schedule the pod
	// when constraints are not satisfied.
	DoNotSchedule UnsatisfiableConstraintAction = "DoNotSchedule"
	// ScheduleAnyway instructs the multicluster to schedule the pod
	// even if constraints are not satisfied.
	ScheduleAnyway UnsatisfiableConstraintAction = "ScheduleAnyway"
)

// SchedulePolicy describe the topo value required replicas
type SchedulePolicy struct {
	// Name is the topo value
	Name string `json:"name"`

	// Replicas is the desired the replicas for the topo value
	Replicas int32 `json:"replicas"`
}

type UpdatePolicy struct {
	MaxSkew        int32    `json:"maxSkew"`

	UpdateSequence []string `json:"updateSequence"`
}

// TopologySchedulingPolicyStatus describe the placement results.
type TopologySchedulingPolicyStatus struct {
	Placement []SchedulePolicy `json:"placement"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TopologySchedulingPolicyList is a collection of policy.
type TopologySchedulingPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of MultiClusterPolicy
	Items []TopologySchedulingPolicy `json:"items"`
}
