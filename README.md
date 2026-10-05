# Pod Resources

[![CI](https://github.com/saidsef/pod-resources/actions/workflows/ci.yml/badge.svg)](https://github.com/saidsef/pod-resources/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/saidsef/pod-resources)](https://goreportcard.com/report/github.com/saidsef/pod-resources)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/saidsef/pod-resources)
[![GoDoc](https://godoc.org/github.com/saidsef/pod-resources?status.svg)](https://pkg.go.dev/github.com/saidsef/pod-resources?tab=doc)
![GitHub release(latest by date)](https://img.shields.io/github/v/release/saidsef/pod-resources)
![Commits](https://img.shields.io/github/commits-since/saidsef/pod-resources/latest.svg)
![GitHub](https://img.shields.io/github/license/saidsef/pod-resources)

Pod Resources runs inside a Kubernetes cluster and checks each container's CPU and memory usage against its limits and requests, without Prometheus or Datadog. It reads usage from the Metrics API on a fixed interval, and reports every container that runs above its limit or request, or has none set. It writes those reports to its log, or posts them to a Slack channel.

- **Checks** - warns when a container has no CPU or memory limit or request, and alerts when its usage goes above either
- **Scope** - checks every pod outside `kube-system`
- **Resources** - checks CPU, memory or both, as `RESOURCE_TYPE` sets
- **Delivery** - writes each message to the log, or posts it to Slack once you give it a token

## Quick start

```sh
git clone https://github.com/saidsef/pod-resources.git
cd pod-resources
kubectl create namespace pod-resources
kubectl apply -k deployment/ -n pod-resources
kubectl logs -n pod-resources deploy/pod-resources
```

The Deployment in `deployment/base/` sets no environment variables, so the monitor runs with its defaults and writes what it finds to its log. To send its messages to Slack instead, see [Slack](./docs/configuration.md#slack).

## Documentation

[Read the Docs](https://pod-resources.readthedocs.io/en/latest/) hosts the same pages.

| Page | Contents |
|------|----------|
| [Installation](./docs/installation.md) | Requirements, Kubernetes manifests, the container image, building from source, log output |
| [Configuration](./docs/configuration.md) | Environment variables, resource types, Slack |
| [Alerts](./docs/alerts.md) | The checks, message text, delivery to the log or Slack, usage figures |
| [Architecture](./docs/architecture.md) | What one check does, from listing pods to sending messages |

## Requirements

Pod Resources needs a Kubernetes cluster with metrics-server, and permission to create a ClusterRole and a ClusterRoleBinding in it. Building it from source needs the Go version that `go.mod` sets.

## Source

Our latest and greatest source of *pod-resources* can be found on [GitHub](https://github.com/saidsef/pod-resources). [Fork us](https://github.com/saidsef/pod-resources/fork)!

## Contributing

We would :heart: you to contribute by making a [pull request](https://github.com/saidsef/pod-resources/pulls).

Please read the official [Contribution Guide](./CONTRIBUTING.md) for more information on how you can contribute.
