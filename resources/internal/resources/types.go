package containers

import (
	v1 "k8s.io/api/core/v1"
)

// PodInfo represents information about a Kubernetes Pod.
type PodInfo struct {
	// Name is the name of the Pod.
	Name string

	// Namespace is the namespace in which the Pod is running.
	Namespace string

	// Resources represents the resource requirements of the Pod.
	Resources v1.ResourceRequirements

	// Usage is a slice of UsageInfo representing the resource usage of the Pod.
	Usage []UsageInfo
}

// UsageInfo represents the resource usage information of a container within a Pod.
type UsageInfo struct {
	// Name is the name of the container.
	Name string

	// CPU is the CPU usage of the container in millicores.
	CPU int64

	// Memory is the memory usage of the container in bytes.
	Memory int64
}
