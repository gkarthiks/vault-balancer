package helper

import (
	"context"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"reflect"
	"strings"
	"vault-balancer/globals"
)

const (
	Attempts int = iota
	Retry    int = iota
)

// GetVaultIPsFromLabelSelectors will extract the IP Addresses for the pods that matches the labelSelectors
func GetVaultIPsFromLabelSelectors(labelSelector string, versionLogger *log.Entry)  map[string]struct{} {
	if len(labelSelector) > 0 {
		labelSelector = strings.Join(strings.Split(labelSelector, ","), ",")
		versionLogger.Infof("Discovering the Vault pods based on the label selector '%v'.", labelSelector)
		pods, err := globals.K8s.Clientset.CoreV1().Pods(globals.Namespace).List(context.Background(), metaV1.ListOptions{
			LabelSelector: strings.TrimSpace(labelSelector),
		})
		if err != nil {
			versionLogger.Fatalf("err while retrieving the pods: %v", err)
		} else {
			ipAddresses := populateIpAddresses(pods)
			versionLogger.Infof("Finalized pods discovery process with label selector. Obtained the IP Address %v", reflect.ValueOf(ipAddresses).MapKeys())
			return ipAddresses
		}
	}
	return nil
}

// GetAttemptsFromContext returns the attempts for a request
func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

// GetRetryFromContext returns the attempts for request
func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}


// extracts the pods IP from the selected pods
func populateIpAddresses(podsList *v1.PodList) map[string]struct{} {
	podNameAddressMap := make(map[string]struct{})
	for _, pod := range podsList.Items {
		if pod.Status.Phase == v1.PodRunning {
			podNameAddressMap[pod.Status.PodIP] = struct{}{}
		}
	}
	return podNameAddressMap
}
