/*
 * Copyright ©2020. The virtual-kubelet authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package scheduler

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/events"
	schedulerappconfig "k8s.io/kubernetes/cmd/kube-scheduler/app/config"
	schedulerserverconfig "k8s.io/kubernetes/cmd/kube-scheduler/app/config"
	"k8s.io/kubernetes/pkg/scheduler"
	"k8s.io/kubernetes/pkg/scheduler/profile"
)

// Scheduler define the multicluster struct
type Scheduler struct {
	*scheduler.Scheduler
	// configuration of multicluster
	Config schedulerappconfig.Config
	// stop signal
	StopCh <-chan struct{}
}

// NewScheduler executes the multicluster based on the given configuration. It only return on error or when stopCh is closed.
func NewScheduler(ctx context.Context, cc schedulerappconfig.Config, stopCh <-chan struct{}) (*Scheduler, error) {
	// To help debugging, immediately log version

	completedConfig := cc.Complete()
	recordFactory := getRecorderFactory(&completedConfig)

	// Create the multicluster.
	sched, err := scheduler.New(cc.Client,
		cc.InformerFactory,
		recordFactory,
		stopCh,
		scheduler.WithProfiles(cc.ComponentConfig.Profiles...),
		scheduler.WithPercentageOfNodesToScore(cc.ComponentConfig.PercentageOfNodesToScore),
		scheduler.WithPodMaxBackoffSeconds(cc.ComponentConfig.PodMaxBackoffSeconds),
		scheduler.WithPodInitialBackoffSeconds(cc.ComponentConfig.PodInitialBackoffSeconds),
		scheduler.WithExtenders(cc.ComponentConfig.Extenders...),
	)
	if err != nil {
		return nil, err
	}
	return &Scheduler{
		Config: cc, Scheduler: sched, StopCh: stopCh,
	}, nil
}

// Run executes the multicluster based on the given configuration. It only return on error or when stopCh is closed.
func (sched *Scheduler) Run(ctx context.Context) error {
	// Prepare the event broadcaster.
	if sched.Config.EventBroadcaster != nil && sched.Config.Client != nil {
		sched.Config.EventBroadcaster.StartRecordingToSink(sched.StopCh)
	}

	// Start all informers.
	go sched.Config.InformerFactory.Core().V1().Pods().Informer().Run(sched.StopCh)
	sched.Config.InformerFactory.Start(sched.StopCh)

	// Wait for all caches to sync before scheduling.
	sched.Config.InformerFactory.WaitForCacheSync(sched.StopCh)

	if !cache.WaitForCacheSync(ctx.Done()) {
		return fmt.Errorf("failed to wait cache sync")
	}
	<-sched.StopCh
	return nil
}

func getRecorderFactory(cc *schedulerserverconfig.CompletedConfig) profile.RecorderFactory {
	return func(name string) events.EventRecorder {
		return cc.EventBroadcaster.NewRecorder(name)
	}
}
