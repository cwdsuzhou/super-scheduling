package util

import "github.com/cwdsuzhou/super-scheduling/pkg/apis/scheduling/v1alpha1"

func CovertPolicy(dp []v1alpha1.SchedulePolicy) map[string]int32 {
	policy := make(map[string]int32)
	for _, p := range dp {
		policy[p.Name] = p.Replicas
	}
	return policy
}
