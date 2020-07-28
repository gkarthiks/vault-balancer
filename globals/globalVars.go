package globals

import discovery "github.com/gkarthiks/k8s-discovery"

var (
	VaultIPList   map[string]string
	K8s           *discovery.K8s
	Namespace     string
	HttpTimeout   string
	LabelSelector string
)

const (
	DefaultTimeOut      = "1"
	DefaultBalancerPort = 8000
)
