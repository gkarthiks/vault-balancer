package globals

import (
	discovery "github.com/gkarthiks/k8s-discovery"
)

var (
	K8s           *discovery.K8s
	Namespace     string
	HttpTimeout   string
)

const (
	DefaultTimeOut      = "1"
	DefaultBalancerPort = 8000
)
