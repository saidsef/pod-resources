# Installation

## Requirements

- A Kubernetes cluster, and permission to create a ClusterRole and a ClusterRoleBinding in it
- [metrics-server](https://github.com/kubernetes-sigs/metrics-server), or another server for the `metrics.k8s.io` API
- `kubectl` with kustomize support
- Go at the version `go.mod` sets, to build from source

## Kubernetes

The manifests live under [`deployment/`](https://github.com/saidsef/pod-resources/tree/main/deployment). Clone the repository, create the namespace and apply them with kustomize:

```sh
git clone https://github.com/saidsef/pod-resources.git
cd pod-resources
kubectl create namespace pod-resources
kubectl apply -k deployment/ -n pod-resources
```

`deployment/kustomization.yml` sets no namespace, so pass `-n pod-resources` when you apply. The ClusterRoleBinding points at the ServiceAccount in the `pod-resources` namespace. Install it anywhere else and the monitor has no permission to list pods.

The base creates these objects, all named `pod-resources`:

| Kind | Purpose |
|------|---------|
| ServiceAccount | The identity the monitor runs as |
| ClusterRole | `get` and `list` on pods, nodes and `batch` Jobs, and on pod and node metrics in `metrics.k8s.io` |
| ClusterRoleBinding | Grants the ClusterRole to the ServiceAccount across the cluster |
| Deployment | One replica of the monitor |

The Deployment sets no environment variables, so the monitor runs with its defaults and writes what it finds to its log. To set your own values, replace `env: []` in [`deployment/base/deployment.yaml`](https://github.com/saidsef/pod-resources/blob/main/deployment/base/deployment.yaml) with the variables from [Configuration](./configuration.md).

The pod runs as user and group 1000, with a read-only root filesystem, no Linux capabilities and no privilege escalation.

## Container image

CI publishes the image to `ghcr.io/saidsef/pod-resources` for `linux/amd64` and `linux/arm64`. It builds a new image only when a commit changes something under `resources/`.

CI tags every build `vYYYY.MM`, for the year and month it ran. It also tags a build from `main` as `latest`. It tags a build from any other branch with the branch name. It lower-cases the name, turns each run of other characters into a single `-` and cuts it to 50 characters, so `feat/Slack_Retry` becomes `feat-slack-retry`.

The image starts from `scratch` and holds nothing but the static binary.

## Building from source

```sh
go build -o pod-resources ./resources/resources.go
docker build -t pod-resources .
```

The binary only runs inside a pod. It reads the in-cluster service account config, has no kubeconfig support, and exits with `Client initialisation error` anywhere else.

## Log output

```sh
kubectl logs -n pod-resources deploy/pod-resources
```

The monitor writes one JSON object per log line. At start-up it logs a warning for each variable you left unset, naming the default it falls back to:

```json
{"level":"warning","msg":"DURATION_SECONDS environment variable not set, defaulting to 120s","time":"2026-10-05T09:00:00Z"}
```

The first check runs one interval after start-up. Without Slack, each message from a check shows up as an `info` line, with the text in `msg`:

```json
{"level":"info","msg":"WARNING: Container web in pod web-0 namespace shop has no cpu limit set. Current usage: 500m","time":"2026-10-05T09:02:00Z"}
```

[Alerts](./alerts.md) lists every message the monitor sends.

## Next steps

To choose the resources to check or send the messages to Slack, see [Configuration](./configuration.md).
