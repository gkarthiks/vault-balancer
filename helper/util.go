package helper

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
	"vault-balancer/types"
)

const (
	Attempts int = iota
	Retry int = iota
)

var VaultPool types.VaultPool


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
func HealthCheck() {
	t := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-t.C:
			log.Info("Starting health check...")
			VaultPool.HealthCheck()
			log.Info("Health check completed")
		}
	}
}
