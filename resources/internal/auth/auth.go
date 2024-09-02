package auth

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

type ClientManager struct {
	k8sClientset     *kubernetes.Clientset
	metricsClientset *versioned.Clientset
	k8sOnce          sync.Once
	metricsOnce      sync.Once
	configOnce       sync.Once
	log              *logrus.Logger
	config           *rest.Config
	configErr        error
}

func NewClientManager(log *logrus.Logger) *ClientManager {
	return &ClientManager{log: log}
}

func (m *ClientManager) getInClusterConfig() (*rest.Config, error) {
	m.configOnce.Do(func() {
		m.config, m.configErr = rest.InClusterConfig()
		if m.configErr != nil {
			m.configErr = fmt.Errorf("failed to get in-cluster Kubernetes config: %w", m.configErr)
		}
	})
	return m.config, m.configErr
}

func (m *ClientManager) GetKubernetesClient() (*kubernetes.Clientset, error) {
	var err error
	m.k8sOnce.Do(func() {
		config, errConfig := m.getInClusterConfig()
		if errConfig != nil {
			m.log.Error(errConfig)
			err = errConfig
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

func (m *ClientManager) GetMetricsClient() (*versioned.Clientset, error) {
	var err error
	m.metricsOnce.Do(func() {
		config, errConfig := m.getInClusterConfig()
		if errConfig != nil {
			m.log.Error(errConfig)
			err = errConfig
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

	m.log.Info("Metrics clientset created: ", m.metricsClientset)

	return m.metricsClientset, nil
}
