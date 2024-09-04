package containers

import (
	"context"
	"fmt"

	"github.com/saidsef/pod-resources/resources/internal/notifications"
	"github.com/saidsef/pod-resources/resources/utils"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	api = *notifications.NewSlackClient()
	log = utils.Logger()
)

func ExtractUsageInfo(metrics *v1beta1.PodMetrics) []UsageInfo {
	var usageInfo []UsageInfo
	for _, mc := range metrics.Containers {
		usageInfo = append(usageInfo, UsageInfo{
			Name:             mc.Name,
			CPU:              mc.Usage.Cpu().MilliValue(),               // value in m
			Memory:           mc.Usage.Memory().Value() / (1024 * 1024), // value in Mi
			EphemeralStorage: mc.Usage.StorageEphemeral().Value(),       //value b
		})
	}
	return usageInfo
}

func CheckResources(info PodInfo) {
	messages := []string{}
	sendOrAppend := func(message string) {
		if notifications.SlackEnabled() {
			notifications.SendSlackNotification(&api, message)
		} else {
			messages = append(messages, message)
		}
	}

	CheckResourceRL(info, sendOrAppend)

	if len(messages) > 0 && !notifications.SlackEnabled() {
		utils.LogWithFields(logrus.InfoLevel, messages, "Resource(s) need adjusting")
	}
}

func CheckResourceRL(info PodInfo, sendOrAppend func(string)) {
	messages := []string{}
	for _, resource := range []v1.ResourceList{info.Resources.Limits, info.Resources.Requests} {
		for resourceName, resourceQuantity := range resource {
			if requestQuantity, exists := resource[resourceName]; exists {
				if resourceQuantity.Cmp(requestQuantity) < 0 {
					messages = append(messages, fmt.Sprintf("ALERT: Container %s in namespace %s has resource %s exceeding its request limit. Current usage: %s", info.Name, info.Namespace, resourceName, resourceQuantity.String()))
				}
				if resourceQuantity.Cmp(requestQuantity) > 0 {
					messages = append(messages, fmt.Sprintf("ALERT: Container %s in namespace %s has resource %s exceeding its limit. Current usage: %s", info.Name, info.Namespace, resourceName, resourceQuantity.String()))
				}
			}

			for _, resourceName := range []v1.ResourceName{v1.ResourceCPU, v1.ResourceMemory} {
				if _, exists := resource[resourceName]; !exists {
					messages = append(messages, fmt.Sprintf("WARNING: Container %s in namespace %s has no %s limit set. Current state: %v", info.Name, info.Namespace, resourceName, info.Usage))
				}
				if _, exists := resource[resourceName]; !exists {
					messages = append(messages, fmt.Sprintf("WARNING: Container %s in namespace %s has no %s request set. Current state: %v", info.Name, info.Namespace, resourceName, info.Usage))
				}
			}
		}
	}
	for _, message := range messages {
		sendOrAppend(message)
	}
}

func GetPodInfo(clientset *kubernetes.Clientset, metricset *versioned.Clientset) ([]PodInfo, error) {
	var podInfo []PodInfo
	pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Cannot get pods: %w", err)
	}

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
			usageInfo := ExtractUsageInfo(metrics)
			podInfo = append(podInfo, PodInfo{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Resources: container.Resources,
				Usage:     usageInfo,
			})
		}
	}
	return podInfo, nil
}
