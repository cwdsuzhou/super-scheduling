# Super scheduling

## Introduction

This project includes a topology-scheduler and a descheduler extened from [descheduler](https://github.com/kubernetes-sigs/descheduler.git).

topology-scheduler will help scheduling pods cross zones, regions or clusters.

## Why we need this

`TopologySpreadConstraint` helps schedule pods with desired skew but it can not solve the issue: schedule desired 
replicas to a `zone`, `region` or `cluster`, e.g.

```yaml
zoneA: 6 Pods
zoneB: 1 Pods
zoneC: 2 Pods
```

## Install

## Use Case

### multi zone

### multi region

### multi cluster