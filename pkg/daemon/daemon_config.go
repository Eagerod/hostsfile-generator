package daemon

import (
	"errors"
	"os"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Everything needed to control what the daemon executes against.
type DaemonConfig struct {
	RestConfig          *rest.Config
	KubernetesClientSet *kubernetes.Clientset

	PiholePodName string
	IngressIp     string
	SearchDomain  string
}

// Assumes that this is running in the same pod as the pihole.
// Uses the pod's own hostname to find the pod's name.
func NewDaemonConfigInCluster(ingressIp string, searchDomain string) (*DaemonConfig, error) {
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); os.IsNotExist(err) {
		return nil, errors.New("cannot find service account token. Maybe it hasn't been attached?")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	daemonConfig := DaemonConfig{config, clientset, hostname, ingressIp, searchDomain}
	return &daemonConfig, nil
}

func NewDaemonConfig(ingressIp, searchDomain, clusterIp, bearerToken, piholePodName string) (*DaemonConfig, error) {
	if ingressIp == "" {
		return nil, errors.New("ingress IP must be provided")
	}

	if clusterIp == "" {
		return nil, errors.New("kubernetes API server host must be provided")
	}

	config := &rest.Config{}
	err := rest.SetKubernetesDefaults(config)
	if err != nil {
		return nil, err
	}

	groupVersion := schema.GroupVersion{}
	url, str, err := rest.DefaultServerURL(clusterIp, "v1", groupVersion, true)
	if err != nil {
		return nil, err
	}

	config.Host = url.String()
	config.APIPath = str
	config.BearerToken = bearerToken
	config.TLSClientConfig.Insecure = true

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	daemonConfig := DaemonConfig{config, clientset, piholePodName, ingressIp, searchDomain}
	return &daemonConfig, nil
}
