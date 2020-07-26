package types

import (
	"net/http/httputil"
	"net/url"
	"sync"
)

// VaultBackend holds the data about a vault pod
type VaultBackend struct {
	IP           string
	ProxyURL     *url.URL
	HealthURL    *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

// SetAlive for represented vault pod
func (v *VaultBackend) SetAlive(alive bool) {
	v.mux.Lock()
	v.Alive = alive
	v.mux.Unlock()
}

// IsAlive returns true when Vault pod is alive
func (v *VaultBackend) IsAlive() (alive bool) {
	v.mux.RLock()
	alive = v.Alive
	v.mux.RUnlock()
	return
}
