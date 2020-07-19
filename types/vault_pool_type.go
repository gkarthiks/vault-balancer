package types

import (
	log "github.com/sirupsen/logrus"
	"net"
	"net/url"
	"sync/atomic"
	"time"
)

// VaultPool holds information about reachable vault pods
type VaultPool struct {
	vaultBackends []*VaultBackend
	current  uint64
}

// AddBackend to the existing vault pool
func (vp *VaultPool) AddBackend(vaultBackend *VaultBackend) {
	vp.vaultBackends = append(vp.vaultBackends, vaultBackend)
}

// NextIndex atomically increase the counter and return an index
func (vp *VaultPool) NextIndex() int {
	return int(atomic.AddUint64(&vp.current, uint64(1)) % uint64(len(vp.vaultBackends)))
}

// MarkVaultPodStatus changes a status of a vault pod
func (vp *VaultPool) MarkVaultPodStatus(podUrl *url.URL, alive bool) {
	for _, b := range vp.vaultBackends {
		if b.URL.String() == podUrl.String() {
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
		alive := isBackendAlive(vaults.URL)
		vaults.SetAlive(alive)
		if !alive {
			status = "down"
		}
		log.Infof("Status of the URL %s :  %s", vaults.URL, status)
	}
}


// isBackendAlive checks whether a pod is reachable by establishing a TCP connection
func isBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		log.Errorf("site is not reachable with the following error: %v", err.Error())
		return false
	}
	_ = conn.Close()
	return true
}
