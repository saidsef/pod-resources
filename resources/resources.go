package main

import (
	"context"
	"fmt"
	"time"

	"github.com/saidsef/pod-resources/resources/internal/auth"
	"github.com/saidsef/pod-resources/resources/internal/notifications"
	co "github.com/saidsef/pod-resources/resources/internal/resources"
	"github.com/saidsef/pod-resources/resources/utils"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// main is the entry point of the application. It sets up the Kubernetes client,
// retrieves pod metrics, and periodically checks resource usage.
func main() {
	log := utils.Logger()
	k8sManager := auth.NewClientManager(log)
	DURATION_SECONDS := utils.GetEnv("DURATION_SECONDS", "120", log)

	clientset, err := k8sManager.GetKubernetesClient()
	if err != nil {
		utils.LogWithFields(logrus.FatalLevel, nil, "Kubernetes config error", err)
		return
	}

	metricset, err := k8sManager.GetMetricsClient()
	if err != nil {
		utils.LogWithFields(logrus.FatalLevel, nil, "Metrics config error", err)
		return
	}

	pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		utils.LogWithFields(logrus.ErrorLevel, nil, "Cannot get pods", err)
		return
	}

	duration, err := time.ParseDuration(DURATION_SECONDS)
	if err != nil {
		utils.LogWithFields(logrus.ErrorLevel, nil, "Cannot parse duration", err)
		return
	}

	ticker := time.NewTicker(duration * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var podInfo []co.PodInfo
		for _, pod := range pods.Items {
			if pod.Namespace == "kube-system" {
				continue
			}
			for _, container := range pod.Spec.Containers {
				utils.LogWithFields(logrus.DebugLevel, nil, fmt.Sprintf("getting metrics for %s in namespace %s", container.Name, pod.Namespace))
				metrics, err := metricset.MetricsV1beta1().PodMetricses(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
				if err != nil {
					utils.LogWithFields(logrus.ErrorLevel, nil, fmt.Sprintf("Error getting metrics for %s in namespace %s", container.Name, pod.Namespace), err)
					continue
				}
				var usageInfo []co.UsageInfo
				for _, mc := range metrics.Containers {
					usageInfo = append(usageInfo, co.UsageInfo{
						Name:   mc.Name,
						CPU:    mc.Usage.Cpu().MilliValue(),
						Memory: mc.Usage.Memory().Value() / (1024 * 1024),
					})
				}
				podInfo = append(podInfo, co.PodInfo{
					Name:      pod.Name,
					Namespace: pod.Namespace,
					Resources: container.Resources,
					Usage:     usageInfo,
				})
			}
		}
		for _, info := range podInfo {
			checkResources(info)
		}
	}
}

// checkResources checks the resource usage of a given pod and sends notifications
// if the usage exceeds defined limits or requests.
func checkResources(info co.PodInfo) {
	api := *notifications.NewSlackClient()
	messages := []string{}

	sendOrAppend := func(message string) {
		if notifications.SlackEnabled() {
			notifications.SendSlackNotification(&api, message)
		} else {
			messages = append(messages, message)
		}
	}

	for resourceName, resourceQuantity := range info.Resources.Limits {
		if requestQuantity, exists := info.Resources.Requests[resourceName]; exists {
			if resourceQuantity.Cmp(requestQuantity) < 0 {
				sendOrAppend(fmt.Sprintf("ALERT: Container %s in namespace %s has resource %s exceeding its request limit. Current usage: %s", info.Name, info.Namespace, resourceName, resourceQuantity.String()))
			}
		} else {
			sendOrAppend(fmt.Sprintf("WARNING: Container %s in namespace %s has resource %s limit set but no request defined. Current usage: %s", info.Name, info.Namespace, resourceName, resourceQuantity.String()))
		}
	}

	for resourceName, resourceQuantity := range info.Resources.Requests {
		if limitQuantity, exists := info.Resources.Limits[resourceName]; exists {
			if resourceQuantity.Cmp(limitQuantity) > 0 {
				sendOrAppend(fmt.Sprintf("ALERT: Container %s in namespace %s has resource %s exceeding its limit. Current usage: %s", info.Name, info.Namespace, resourceName, resourceQuantity.String()))
			}
		} else {
			sendOrAppend(fmt.Sprintf("WARNING: Container %s in namespace %s has resource %s request set but no limit defined. Current usage: %s", info.Name, info.Namespace, resourceName, resourceQuantity.String()))
		}
	}

	for _, resourceName := range []v1.ResourceName{v1.ResourceCPU, v1.ResourceMemory} {
		if _, exists := info.Resources.Limits[resourceName]; !exists {
			sendOrAppend(fmt.Sprintf("WARNING: Container %s in namespace %s has no %s limit set. Current state: %v", info.Name, info.Namespace, resourceName, info.Usage))
		}
		if _, exists := info.Resources.Requests[resourceName]; !exists {
			sendOrAppend(fmt.Sprintf("WARNING: Container %s in namespace %s has no %s request set. Current state: %v", info.Name, info.Namespace, resourceName, info.Usage))
		}
	}

	if len(messages) > 0 && !notifications.SlackEnabled() {
		utils.LogWithFields(logrus.InfoLevel, messages, "Resource(s) need adjusting")
	}
}
