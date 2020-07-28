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
	"vault-balancer/types"
)

const (
	Attempts int = iota
	Retry    int = iota
)

// GetVaultIPsFromLabelSelectors will extract the IP Addresses for the pods that matches the labelSelectors
func GetVaultIPsFromLabelSelectors(vaultPool *types.VaultPool) {
	if len(globals.LabelSelector) > 0 {
		log.Infof("Discovering the Vault pods based on the label selector '%v'.", globals.LabelSelector)
		strings.Split(globals.LabelSelector, ",")
		log.Infof("Discovering pods with label selector...")
		pods, err := globals.K8s.Clientset.CoreV1().Pods(globals.Namespace).List(context.Background(), metaV1.ListOptions{
			LabelSelector: strings.TrimSpace(globals.LabelSelector),
		})
		if err != nil {
			log.Fatalf("err while retrieving the pods: %v", err)
		} else {
			populateIpAddresses(pods, vaultPool)
		}
		log.Infof("Finalized pods discovery process with label selector. Obtained the IP Address %v", reflect.ValueOf(globals.VaultIPList).MapKeys())
	}

	log.Printf("Vault Pool data at the end of GetVault IPs %v", vaultPool)
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

// HealthCheck runs a routine for check status of the pods every 2 mins
func HealthCheck(vaultPool *types.VaultPool) {
	log.Info("Starting health check...")
	vaultPool.HealthCheck()
	log.Info("Health check completed")
}

// extracts the pods IP from the selected pods
func populateIpAddresses(podsList *v1.PodList, vaultPool *types.VaultPool) {
	currentPodNames := make(map[string]struct{})
	for _, pod := range podsList.Items {
		currentPodNames[pod.Name] = struct{}{}
		if pod.Status.Phase == v1.PodRunning {
			// adding the currently discovered pod ips
			globals.VaultIPList[pod.Name] = pod.Status.PodIP
			//if _, ok := globals.VaultIPList[pod.Status.PodIP]; ok {
			//	log.Infof("%v already added", pod.Status.PodIP)
			//} else {
			//	globals.VaultIPList[pod.Status.PodIP] = struct{}{}
			//}
		}
	}
	for historyPodName, ipAddress := range globals.VaultIPList {
		if _, ok := currentPodNames[historyPodName]; !ok {
			// removing the obsolete pod and its details
			delete(globals.VaultIPList, historyPodName)
			vaultPool.RetireBackend(ipAddress)
		}
	}
}
