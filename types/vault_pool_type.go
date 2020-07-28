package types

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
	"vault-balancer/globals"
)

// VaultPool holds information about reachable vault pods
type VaultPool struct {
	vaultBackends []*VaultBackend
	current       uint64
}

// AddBackend to the existing vault pool
func (vp *VaultPool) AddBackend(vaultBackend *VaultBackend) {
	vp.vaultBackends = append(vp.vaultBackends, vaultBackend)
}

// AddBackend to the existing vault pool
func (vp *VaultPool) RetireBackend(obsoleteIP string) {
	for index, currBackend := range vp.vaultBackends {
		if currBackend.IP == obsoleteIP {
			log.Infof("Retiring the backend %v from list of active IPs", obsoleteIP)
			vp.vaultBackends = append(vp.vaultBackends[:index], vp.vaultBackends[index+1:]...)
		}
	}
}

// NextIndex atomically increase the counter and return an index
func (vp *VaultPool) NextIndex() int {
	return int(atomic.AddUint64(&vp.current, uint64(1)) % uint64(len(vp.vaultBackends)))
}

// MarkVaultPodStatus changes a status of a vault pod
func (vp *VaultPool) MarkVaultPodStatus(podUrl *url.URL, alive bool) {
	for _, b := range vp.vaultBackends {
		if b.ProxyURL.String() == podUrl.String() {
			b.SetAlive(alive)
			break
		}
	}
}

// GetNextPod returns next active peer to take a connection
func (vp *VaultPool) GetNextPod() *VaultBackend {
	next := vp.NextIndex()
	l := len(vp.vaultBackends) + next
	for i := next; i < l; i++ {
		idx := i % len(vp.vaultBackends)
		if vp.vaultBackends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&vp.current, uint64(idx))
			}
			return vp.vaultBackends[idx]
		}
	}
	return nil
}

// HealthCheck pings the backends and update the status
func (vp *VaultPool) HealthCheck() {
	for _, vaults := range vp.vaultBackends {
		status := "up"
		alive := isBackendAlive(vaults.HealthURL)
		vaults.SetAlive(alive)
		if !alive {
			status = "down"
		}
		log.Infof("Status of the URL %s :  %s", vaults.IP, status)
	}
}

// isBackendAlive checks whether a pod is reachable by establishing a TCP connection
func isBackendAlive(vaultHealthUrl *url.URL) bool {
	timeOutDuration, _ := time.ParseDuration(globals.HttpTimeout)
	client := &http.Client{Timeout: timeOutDuration * time.Minute}

	healthCheckRequest, err := http.NewRequest("GET", vaultHealthUrl.String(), nil)
	if err != nil {
		log.Errorf("error occurred while creating a new request for health check, error: %v", err)
	}
	healthResponse, err := client.Do(healthCheckRequest)
	if err != nil {
		log.Errorf("error while executing the http call, error: %v", err)
		return false
	}
	if healthResponse.Status == "200 OK" {
		defer healthResponse.Body.Close()
		body, err := ioutil.ReadAll(healthResponse.Body)
		if err != nil {
			log.Errorf("error occurred while reading the response body, error %v", err)
			client.CloseIdleConnections()
			return false
		} else {
			var responseBody VaultResponseType
			err = json.Unmarshal(body, &responseBody)
			if err != nil {
				log.Errorf("error occurred while unmarshalling response, error: %v", err)
				client.CloseIdleConnections()
				return false
			} else {
				if responseBody.Sealed != false {
					client.CloseIdleConnections()
					return false
				}
			}
		}
	} else {
		client.CloseIdleConnections()
		return false
	}
	client.CloseIdleConnections()
	return true
}
