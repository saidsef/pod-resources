# Configuration

Every setting is an environment variable on the monitor container. The monitor reads them once, at start-up.

## Environment variables

| Variable | Values | Default | Description |
|----------|--------|---------|-------------|
| `DURATION_SECONDS` | A positive Go duration, such as `90s`, `5m` or `1h30m` | `120s` | The time between checks. The value needs a unit, so `120` fails and `120s` works. If Go cannot parse the value, the monitor logs `Cannot parse duration` and exits. If the value is zero or negative, it panics |
| `RESOURCE_TYPE` | `CPU`, `MEMORY` or both, comma-separated | `CPU,MEMORY` | The resources to check. [Resource types](#resource-types) covers how the monitor reads the list |
| `SLACK_TOKEN` | A Slack bot token | None | The token the monitor posts with. When unset or empty, the monitor writes every message to its log instead |
| `SLACK_CHANNEL` | A channel name, such as `k8s-alerts`, or a channel ID | `k8s-alerts` | The channel the monitor posts to. If set but empty, the monitor writes to its log instead |

## Resource types

The monitor splits `RESOURCE_TYPE` on commas, trims the spaces around each entry and ignores case, so `cpu, Memory` works. It checks CPU before memory, whatever order you list them in.

The monitor ignores any other entry. When no entry names CPU or memory, it logs `RESOURCE_TYPE "<value>" names no supported resource, nothing will be checked` at start-up, keeps running and sends nothing.

## Slack

The monitor posts through Slack's [`chat.postMessage`](https://docs.slack.dev/reference/methods/chat.postMessage/) method. It needs a bot token, which starts with `xoxb-`, from a Slack app with the `chat:write` scope. Invite the app to the channel, or give it the `chat:write.public` scope to post in any public channel without an invite.

Keep the token in a Secret:

```sh
kubectl create secret generic pod-resources-slack -n pod-resources --from-literal=token=xoxb-...
```

In [`deployment/base/deployment.yaml`](https://github.com/saidsef/pod-resources/blob/main/deployment/base/deployment.yaml), replace `env: []` with a reference to it:

```yaml
env:
  - name: SLACK_TOKEN
    valueFrom:
      secretKeyRef:
        name: pod-resources-slack
        key: token
  - name: SLACK_CHANNEL
    value: k8s-alerts
```

With Slack on, the monitor posts each message on its own and writes none of them to its log. [Delivery](./alerts.md#delivery) covers what happens when a post fails.
