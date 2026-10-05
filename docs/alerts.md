# Alerts

Every check compares each container's CPU and memory usage with its limit and its request. A container with both set that stays within them produces no message.

## Checks

The monitor runs these checks for each resource in `RESOURCE_TYPE`, in this order:

| Condition | Message |
|-----------|---------|
| No limit set | `WARNING: Container <container> in pod <pod> namespace <namespace> has no <resource> limit set. Current usage: <usage>` |
| Usage above the limit | `ALERT: Container <container> in pod <pod> namespace <namespace> has <resource> usage of <usage>, above its limit of <limit>` |
| No request set | `WARNING: Container <container> in pod <pod> namespace <namespace> has no <resource> request set. Current usage: <usage>` |
| Usage above the request | `ALERT: Container <container> in pod <pod> namespace <namespace> has <resource> usage of <usage>, above its request of <request>` |

`<resource>` is `cpu` or `memory`. The monitor gives CPU usage in millicores, such as `250m`, and memory usage in mebibytes, such as `200Mi`. It prints limits and requests the way Kubernetes writes them, such as `500m` or `1Gi`.

## Repeated messages

The monitor keeps no state between checks, so it repeats every message on every check for as long as the condition holds. A container with no limits and no requests produces four warnings each interval, two for CPU and two for memory.

To cut the volume, set limits and requests on the container, lengthen `DURATION_SECONDS`, or drop a resource from `RESOURCE_TYPE`.

## Requests Kubernetes fills in

If a container sets a limit but no request, the API server copies the limit into the request when it accepts the pod ([Resource Management for Pods and Containers](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)). That container never gets the no-request warning. Its request and limit are the same, so once usage passes one it passes the other, and both alerts arrive together.

A LimitRange with default values fills in a missing limit or request in the same way, before the monitor sees the pod.

## Usage figures

The monitor reads usage from the Metrics API, which metrics-server serves from samples it collects from each kubelet. CPU usage is an average over the sampling window, and memory usage is the working set at the last sample. A spike that starts and ends between two samples never reaches the checks.

The monitor compares memory in whole mebibytes and drops any fraction, so usage of 100.9Mi against a request of 100Mi produces no alert.

A container that is missing from the pod's metrics counts as zero usage. It can still get the no-limit and no-request warnings, but never an alert.

## Delivery

With `SLACK_TOKEN` and `SLACK_CHANNEL` both set, the monitor posts each message to the channel the moment it finds it, one Slack message per alert or warning. None of them go to the log. Without Slack, it writes each message to its log as an `info` line. [Log output](./installation.md#log-output) shows that line.

When a post fails, the monitor logs `Failed to send Slack notification` and drops the message. It does not retry, and it does not fall back to the log.

The [`chat.postMessage`](https://docs.slack.dev/reference/methods/chat.postMessage/) reference says Slack "will generally allow an app to post 1 message per second to a specific channel". The monitor posts as fast as Slack replies, so a check with many messages can go over that limit, and the monitor drops every post Slack turns away.
