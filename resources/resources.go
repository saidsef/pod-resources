package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/saidsef/pod-resources/resources/internal/auth"
	co "github.com/saidsef/pod-resources/resources/internal/resources"
	"github.com/saidsef/pod-resources/resources/utils"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	DURATION_SECONDS = utils.GetEnv("DURATION_SECONDS", "120s", log)
	RESOURCE_TYPE    = strings.Split(utils.GetEnv("RESOURCE_TYPE", "CPU,MEMORY", utils.Logger()), ",")
	k8sManager       = *auth.NewClientManager(log)
	log              = utils.Logger()
)

func initialiseClients() (*kubernetes.Clientset, *versioned.Clientset, error) {
	clientset, err := k8sManager.GetKubernetesClient()
	if err != nil {
		return nil, nil, fmt.Errorf("Kubernetes config error: %w", err)
	}

	metricset, err := k8sManager.GetMetricsClient()
	if err != nil {
		return nil, nil, fmt.Errorf("Metrics config error: %w", err)
	}

	return clientset, metricset, nil
}

// monitor runs check once per interval until ctx is cancelled.
func monitor(ctx context.Context, interval time.Duration, check func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}

func main() {
	clientset, metricset, err := initialiseClients()
	if err != nil {
		utils.LogWithFields(logrus.FatalLevel, nil, "Client initialisation error", err)
		return
	}

	duration, err := time.ParseDuration(DURATION_SECONDS)
	if err != nil {
		utils.LogWithFields(logrus.ErrorLevel, nil, "Cannot parse duration", err)
		return
	}

	monitor(context.Background(), duration, func() {
		podInfo, err := co.GetPodInfo(clientset, metricset)
		if err != nil {
			utils.LogWithFields(logrus.ErrorLevel, nil, "Error retrieving pod info", err)
			return
		}
		for _, info := range podInfo {
			co.CheckResources(info)
		}
	})
}
