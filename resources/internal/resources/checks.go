package containers

import (
	"context"
	"fmt"

	"github.com/saidsef/pod-resources/resources/internal/notifications"
	"github.com/saidsef/pod-resources/resources/utils"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

const podListPageSize = 500

var api = *notifications.NewSlackClient()

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

	for _, message := range messages {
		utils.LogWithFields(logrus.InfoLevel, nil, message)
	}
}

func CheckResourceRL(info PodInfo, sendOrAppend func(string)) {
	where := fmt.Sprintf("Container %s in pod %s namespace %s", info.Container, info.Name, info.Namespace)

	for _, resourceName := range []v1.ResourceName{v1.ResourceCPU, v1.ResourceMemory} {
		usage := usageOf(info.Usage, resourceName)

		if limit, exists := info.Resources.Limits[resourceName]; !exists {
			sendOrAppend(fmt.Sprintf("WARNING: %s has no %s limit set. Current usage: %s", where, resourceName, formatUsage(usage, resourceName)))
		} else if usage > comparableTo(limit, resourceName) {
			sendOrAppend(fmt.Sprintf("ALERT: %s has %s usage of %s, above its limit of %s", where, resourceName, formatUsage(usage, resourceName), limit.String()))
		}

		if request, exists := info.Resources.Requests[resourceName]; !exists {
			sendOrAppend(fmt.Sprintf("WARNING: %s has no %s request set. Current usage: %s", where, resourceName, formatUsage(usage, resourceName)))
		} else if usage > comparableTo(request, resourceName) {
			sendOrAppend(fmt.Sprintf("ALERT: %s has %s usage of %s, above its request of %s", where, resourceName, formatUsage(usage, resourceName), request.String()))
		}
	}
}

// usageOf returns the measured usage of resourceName in millicores for CPU and mebibytes for memory.
func usageOf(usage UsageInfo, resourceName v1.ResourceName) int64 {
	if resourceName == v1.ResourceCPU {
		return usage.CPU
	}
	return usage.Memory
}

// comparableTo converts quantity into the unit usageOf reports, so the two can be compared as integers.
func comparableTo(quantity resource.Quantity, resourceName v1.ResourceName) int64 {
	if resourceName == v1.ResourceCPU {
		return quantity.MilliValue()
	}
	return quantity.Value() / (1024 * 1024)
}

// formatUsage renders a value returned by usageOf with the unit it is measured in.
func formatUsage(value int64, resourceName v1.ResourceName) string {
	if resourceName == v1.ResourceCPU {
		return fmt.Sprintf("%dm", value)
	}
	return fmt.Sprintf("%dMi", value)
}

func GetPodInfo(clientset *kubernetes.Clientset, metricset *versioned.Clientset) ([]PodInfo, error) {
	var podInfo []PodInfo
	options := metav1.ListOptions{
		FieldSelector: "metadata.namespace!=kube-system",
		Limit:         podListPageSize,
	}

	for {
		pods, err := clientset.CoreV1().Pods("").List(context.Background(), options)
		if err != nil {
			return nil, fmt.Errorf("Cannot get pods: %w", err)
		}

		for _, pod := range pods.Items {
			if pod.Namespace == "kube-system" {
				continue
			}
			podInfo = append(podInfo, containerInfo(metricset, pod)...)
		}

		if pods.Continue == "" {
			return podInfo, nil
		}
		options.Continue = pods.Continue
	}
}

func containerInfo(metricset *versioned.Clientset, pod v1.Pod) []PodInfo {
	utils.LogWithFields(logrus.DebugLevel, nil, fmt.Sprintf("getting metrics for pod %s in namespace %s", pod.Name, pod.Namespace))
	metrics, err := metricset.MetricsV1beta1().PodMetricses(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
	if err != nil {
		utils.LogWithFields(logrus.ErrorLevel, nil, fmt.Sprintf("Error getting metrics for pod %s in namespace %s", pod.Name, pod.Namespace), err)
		return nil
	}

	usage := make(map[string]UsageInfo, len(metrics.Containers))
	for _, containerUsage := range ExtractUsageInfo(metrics) {
		usage[containerUsage.Name] = containerUsage
	}

	info := make([]PodInfo, 0, len(pod.Spec.Containers))
	for _, container := range pod.Spec.Containers {
		info = append(info, PodInfo{
			Name:      pod.Name,
			Container: container.Name,
			Namespace: pod.Namespace,
			Resources: container.Resources,
			Usage:     usage[container.Name],
		})
	}
	return info
}
