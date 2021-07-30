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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SchedulingArgs defines the scheduling parameters for CapacityScheduling plugin.
type SchedulingArgs struct {
	metav1.TypeMeta `json:",inline"`

	// KubeConfigPath is the path of kubeconfig.
	KubeConfigPath *string `json:"kubeConfigPath,omitempty"`

	// KubeMaster is the url of kubernetes master.
	KubeMaster *string `json:"kubeMaster,omitempty"`

	// ClusterConfiguration is a key-value map to store configuration
	ClusterConfiguration map[string]Configuration `json:"clusterConfiguration"`
}

// Configuration defines the lower cluster configuration
type Configuration struct {
	// clusterName
	Name *string `json:"name"`
	// Master URL
	KubeMaster *string `json:"kubeMaster,omitempty"`
	// KubeConfig of the cluster
	KubeConfig *string `json:"kubeConfig,omitempty"`
}
