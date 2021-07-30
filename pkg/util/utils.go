package util

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"

	"github.com/cwdsuzhou/super-scheduling/pkg/apis/scheduling/v1alpha1"
)

// ClustersNodeSelection is a struct including some scheduling parameters
type ClustersNodeSelection struct {
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Affinity     *corev1.Affinity    `json:"affinity,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
}

const (
	// NodeType is define the node type key
	NodeType = "type"
	// VirtualPodLabel is the label of virtual pod
	VirtualPodLabel = "virtual-pod"
	// VirtualKubeletLabel is the label of virtual kubelet
	VirtualKubeletLabel = "virtual-kubelet"
	// TrippedLabels is the label of tripped labels
	TrippedLabels = "tripped-labels"
	// ClusterID marks the id of a cluster
	ClusterID = "clusterID"
	// SelectorKey is the key of ClusterSelector
	SelectorKey = "clusterSelector"
)

func CovertPolicy(dp []v1alpha1.SchedulePolicy) map[string]int32 {
	policy := make(map[string]int32)
	for _, p := range dp {
		policy[p.Name] = p.Replicas
	}
	return policy
}

// IsVirtualNode defines if a node is virtual node
func IsVirtualNode(node *corev1.Node) bool {
	if node == nil {
		return false
	}
	valStr, exist := node.ObjectMeta.Labels[NodeType]
	if !exist {
		return false
	}
	return valStr == VirtualKubeletLabel
}

// IsVirtualPod defines if a pod is virtual pod
func IsVirtualPod(pod *corev1.Pod) bool {
	if pod.Labels != nil && pod.Labels[VirtualPodLabel] == "true" {
		return true
	}
	return false
}

// GetClusterID return the cluster in node label
func GetClusterID(node *corev1.Node) string {
	if node == nil {
		return ""
	}
	clusterName, exist := node.ObjectMeta.Labels[ClusterID]
	if !exist {
		return ""
	}
	return clusterName
}

// ConvertAnnotations converts annotations to ClustersNodeSelection
func ConvertAnnotations(annotation map[string]string) *ClustersNodeSelection {
	if annotation == nil {
		return nil
	}
	val := annotation[SelectorKey]
	if len(val) == 0 {
		return nil
	}

	var cns ClustersNodeSelection
	err := json.Unmarshal([]byte(val), &cns)
	if err != nil {
		return nil
	}
	return &cns
}
