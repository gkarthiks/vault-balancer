package helper

import (
	"context"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
	"time"
	"vault-balancer/globals"
	"vault-balancer/types"
)

const (
	Attempts int = iota
	Retry    int = iota
)

// GetVaultIPsFromLabelSelectors will extract the IP Addresses for the pods that matches the labelSelectors
func GetVaultIPsFromLabelSelectors(labelSelectors string) {
	if len(labelSelectors) > 0 {
		strings.Split(labelSelectors, ",")
		go discoverIPs(labelSelectors)
	}
}

func discoverIPs(labelSelectors string){
	t := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-t.C:
			log.Infof("Discovering pods with label selector...")
			pods, err := globals.K8s.Clientset.CoreV1().Pods(globals.Namespace).List(context.Background(), metaV1.ListOptions{
				LabelSelector: strings.TrimSpace(labelSelectors),
			})
			if err != nil {
				log.Fatalf("err while retrieving the pods: %v", err)
			} else {
				globals.VaultIPList = append(globals.VaultIPList, fetchIpAddress(pods)...)
			}
			log.Infof("Finalized pods discovery process with label selector...")
		}
	}
}

// GetAttemptsFromContext returns the attempts for a request
func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

// GetAttemptsFromContext returns the attempts for request
func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}

// HealthCheck runs a routine for check status of the pods every 2 mins
func HealthCheck(vaultPool *types.VaultPool) {
	t := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-t.C:
			log.Info("Starting health check...")
			vaultPool.HealthCheck()
			log.Info("Health check completed")
		}
	}
}

// extracts the pods IP from the selected pods
func fetchIpAddress(podsList *v1.PodList) []string {
	var podIps []string
	for _, pod := range podsList.Items {
		if pod.Status.Phase == v1.PodRunning {
			podIps = append(podIps, pod.Status.PodIP)
		}
	}
	return podIps
}
