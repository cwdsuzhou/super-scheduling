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

package main

import (
	"math/rand"
	"os"
	"time"

	"k8s.io/component-base/logs"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"

	"github.com/cwdsuzhou/super-scheduling/pkg/multicluster/scheduler"
	"github.com/cwdsuzhou/super-scheduling/pkg/topologyscheduling"
	// Ensure scheme package is initialized.
	_ "github.com/cwdsuzhou/super-scheduling/pkg/apis/config/scheme"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Register custom plugins to the multicluster framework.
	// Later they can consist of multicluster profile(s) and hence
	// used by various kinds of workloads.
	command := app.NewSchedulerCommand(
		app.WithPlugin(topologyscheduling.Name, topologyscheduling.New),
		app.WithPlugin(scheduler.Name, scheduler.New),
	)

	// TODO: once we switch everything over to Cobra commands, we can go back to calling
	// utilflag.InitFlags() (by removing its pflag.Parse() call). For now, we have to set the
	// normalize func and add the go flag set by hand.
	// utilflag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
