package auth

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

// ClientManager manages the creation and access to Kubernetes and metrics clientsets.
type ClientManager struct {
	k8sClientset     *kubernetes.Clientset // The Kubernetes clientset instance.
	metricsClientset *versioned.Clientset  // The metrics clientset instance.
	k8sOnce          sync.Once             // Ensures that the Kubernetes client is created only once.
	metricsOnce      sync.Once             // Ensures that the metrics client is created only once.
	log              *logrus.Logger        // Logger for logging events and errors.
}

// NewClientManager creates a new instance of ClientManager.
// Parameters:
// - log: A logger instance for logging events and errors.
// Returns:
// - A pointer to a new ClientManager instance.
func NewClientManager(log *logrus.Logger) *ClientManager {
	return &ClientManager{log: log}
}

// GetKubernetesClient returns the Kubernetes clientset instance, creating it if necessary.
// Returns:
// - A pointer to the Kubernetes clientset instance.
// - An error if there was an issue creating the clientset.
func (m *ClientManager) GetKubernetesClient() (*kubernetes.Clientset, error) {
	var err error
	m.k8sOnce.Do(func() {
		config, errConfig := rest.InClusterConfig()
		if errConfig != nil {
			err = fmt.Errorf("failed to get in-cluster Kubernetes config: %w", errConfig)
			m.log.Error(err)
			return
		}

		m.k8sClientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			err = fmt.Errorf("unable to create Kubernetes client set for in-cluster config: %w", err)
			m.log.Error(err)
			return
		}

		m.log.Info("Successfully created Kubernetes clientset")
	})

	if err != nil {
		return nil, err
	}

	return m.k8sClientset, nil
}

// GetMetricsClient returns the metrics clientset instance, creating it if necessary.
// Returns:
// - A pointer to the metrics clientset instance.
// - An error if there was an issue creating the metrics clientset.
func (m *ClientManager) GetMetricsClient() (*versioned.Clientset, error) {
	var err error
	m.metricsOnce.Do(func() {
		config, errConfig := rest.InClusterConfig()
		if errConfig != nil {
			err = fmt.Errorf("failed to get in-cluster metric server config: %w", errConfig)
			m.log.Error(err)
			return
		}

		m.metricsClientset, err = versioned.NewForConfig(config)
		if err != nil {
			err = fmt.Errorf("unable to create metrics client set for in-cluster config: %w", err)
			m.log.Error(err)
			return
		}

		m.log.Info("Successfully created metrics clientset")
	})

	if err != nil {
		return nil, err
	}

	// Additional logging to verify the clientset
	m.log.Info("Metrics clientset created: ", m.metricsClientset)

	return m.metricsClientset, nil
}
