# Super scheduling

## Introduction

This project includes a topology-scheduler and a descheduler extened
from [descheduler](https://github.com/kubernetes-sigs/descheduler.git).

topology-scheduler will help scheduling pods cross zones, regions or clusters.

> We would like this project be merged by upstream in the future, so crd and codes includes xxx.scheduling.sigs.k8s.io

## Why we need this

`TopologySpreadConstraint` helps schedule pods with desired skew, but it can not solve the issue: schedule desired
replicas to a `zone`, `region` or `cluster`, e.g.

```yaml
zoneA: 6 Pods
zoneB: 1 Pods
zoneC: 2 Pods
```

## Install

### kube-scheduler

*1* Apply crd

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: topologyschedulingpolicies.scheduling.sigs.k8s.io
spec:
  conversion:
    strategy: None
  group: scheduling.sigs.k8s.io
  names:
    kind: TopologySchedulingPolicy
    listKind: TopologySchedulingPolicyList
    plural: topologyschedulingpolicies
    shortNames:
      - tsp
      - tsps
    singular: topologyschedulingpolicy
  scope: Namespaced
  version: v1alpha1
  versions:
    - name: v1alpha1
      served: true
      storage: true
```

> if your cluster only support kubescheduler.config.k8s.io/v1, please replace this with v1.

*2* deploy scheduler

Replace the kube-scheudler with this one, and add a config like this when starting scheduler.

```yaml
apiVersion: kubescheduler.config.k8s.io/v1beta1
kind: KubeSchedulerConfiguration
leaderElection:
  leaderElect: true
clientConnection:
  kubeconfig: "REPLACE_ME_WITH_KUBE_CONFIG_PATH"
profiles:
  - schedulerName: default-scheduler
    plugins:
      preFilter:
        enabled:
          - name: TopologyScheduling
      filter:
        enabled:
          - name: TopologyScheduling
        disabled:
          - name: "*"
      score:
        enabled:
          - name: TopologyScheduling
        disabled:
          - name: "*"
      reserve:
        enabled:
          - name: TopologyScheduling
    pluginConfig:
      - name: TopologyScheduling
        args:
          kubeConfigPath: "REPLACE_ME_WITH_KUBE_CONFIG_PATH"
```

*3* deploy descheduler

descheduler should be deployed as deployment in cluster

```yaml

```

## Use Case

### multi zone

```yaml
apiVersion: scheduling.sigs.k8s.io/v1alpha1
kind: TopologySchedulingPolicy
metadata:
  name: policy-zone
spec:
  deployPlacement:
    - name: sh-1
      replicas: 6
    - name: nj-2
      replicas: 3
  labelSelector:
    matchLabels:
      cluster-test: "true"
  topologyKey: failure-domain.beta.kubernetes.io/zone
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: env
  name: test
  namespace: default
spec:
  replicas: 9
  selector:
    matchLabels:
      app: env
  template:
    labels:
      app: env
      cluster-test: "true"
      topology-scheduling-policy.scheduling.sigs.k8s.io: policy-zone
    spec:
      containers:
        - image: nginx:latest
          imagePullPolicy: Always
          name: nginx
          resources: { }
```

### multi region

```yaml
apiVersion: scheduling.sigs.k8s.io/v1alpha1
kind: TopologySchedulingPolicy
metadata:
  name: policy-region
spec:
  deployPlacement:
    - name: nj
      replicas: 6
    - name: sh
      replicas: 3
  labelSelector:
    matchLabels:
      cluster-test: "true"
  topologyKey: failure-domain.beta.kubernetes.io/region
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: env
  name: test
  namespace: default
spec:
  replicas: 9
  selector:
    matchLabels:
      app: env
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: env
        cluster-test: "true"
        topology-scheduling-policy.scheduling.sigs.k8s.io: policy-region
    spec:
      containers:
        - image: nginx:latest
          imagePullPolicy: Always
          name: nginx
          resources: { }
```

### multi cluster

This project also can be used in multi-cluster scene by deploy the
[tensile-kube](https://github.com/virtual-kubelet/tensile-kube), with `descheduler` in `tensile-kube` not deployed.

For example, we add a label `cluster-name..scheduling.sigs.k8s.io: cluster1` to a virtual node.

```yaml
apiVersion: scheduling.sigs.k8s.io/v1alpha1
kind: TopologySchedulingPolicy
metadata:
  name: policy-cluster
spec:
  deployPlacement:
    - name: cluster1
      replicas: 6
    - name: cluster2
      replicas: 3
  labelSelector:
    matchLabels:
      cluster-test: "true"
  topologyKey: cluster-name..scheduling.sigs.k8s.io
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: env
  name: test
  namespace: default
spec:
  replicas: 9
  selector:
    matchLabels:
      app: env
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: env
        cluster-test: "true"
        cluster-name..scheduling.sigs.k8s.io: policy-cluster
    spec:
      containers:
        - image: nginx:latest
          imagePullPolicy: Always
          name: nginx
          resources: { }
```