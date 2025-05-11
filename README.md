[![Go Report Card](https://goreportcard.com/badge/github.com/saidsef/pod-resources)](https://goreportcard.com/report/github.com/saidsef/pod-resources)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/saidsef/pod-resources)
[![GoDoc](https://godoc.org/github.com/saidsef/pod-resources?status.svg)](https://pkg.go.dev/github.com/saidsef/pod-resources?tab=doc)
![GitHub release(latest by date)](https://img.shields.io/github/v/release/saidsef/pod-resources)
![Commits](https://img.shields.io/github/commits-since/saidsef/pod-resources/latest.svg)
![GitHub](https://img.shields.io/github/license/saidsef/pod-resources)

# Pod Resources Monitoring

Monitor you app without Prometheus or Datadog. This project is a Kubernetes resource monitoring application that retrieves pod metrics and checks resource usage periodically. It sends alerts and warnings based on the defined resource limits and requests for each container within the pods.

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [License](#license)

## Features

- Monitors CPU and memory usage of Kubernetes pods.
- Sends alerts if resource usage exceeds defined limits or requests.
- Supports Slack notifications for alerts.
- Configurable monitoring duration.

## Requirements

- Go >= 1.21
- Kubernetes cluster
- Access to Kubernetes API
- Slack account (optional for notifications)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/saidsef/pod-resources.git
cd pod-resources
```

2. Install the required Go modules:
```bash
go mod tidy
```

## Configuration

Before running the application, you need to set up the following environment variables - all optional:

- `DURATION_SECONDS`: The duration (in seconds) for which the application will check resource usage. Default is `120s`.
- `SLACK_TOKEN`: The token for your Slack app to send notifications (optional).
- `SLACK_CHANNEL`: The Slack channel where notifications will be sent (optional).

## Usage

To run the application, execute the following command:

```bash
go run resources.go
```

The application will connect to the Kubernetes cluster, retrieve the list of pods, and start monitoring their resource usage based on the specified duration.

### Alerts and Notifications

- If a container exceeds its resource request, an alert will be sent.
- If a container has a limit set but no request defined, a warning will be logged.
- If a container exceeds its resource limit, an alert will be sent.
- Warnings will be logged if no limits or requests are defined for CPU or memory.

## Source

Our latest and greatest source of *Reverse Geocoding* can be found on [GitHub]. [Fork us](https://github.com/saidsef/pod-resources/fork)!

## Contributing

We would :heart: you to contribute by making a [pull request](https://github.com/saidsef/pod-resources/pulls).

Please read the official [Contribution Guide](./CONTRIBUTING.md) for more information on how you can contribute.
