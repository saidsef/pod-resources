package main

import (
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

	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for range ticker.C {
		podInfo, err := co.GetPodInfo(clientset, metricset)
		if err != nil {
			utils.LogWithFields(logrus.ErrorLevel, nil, "Error retrieving pod info", err)
			continue
		}
		for _, info := range podInfo {
			co.CheckResources(info)
		}
	}
}
