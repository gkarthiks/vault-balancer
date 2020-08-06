package main

import (
	"context"
	"fmt"
	discovery "github.com/gkarthiks/k8s-discovery"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"vault-balancer/globals"
	"vault-balancer/helper"
	"vault-balancer/types"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.Infof("Vault Balancer running version: `%v`", BuildVersion)

	globals.K8s, _ = discovery.NewK8s()
	globals.Namespace, _ = globals.K8s.GetNamespace()
	version, _ := globals.K8s.GetVersion()
	log.Infof("Running in %v version of Kubernetes cluster in %s namespace", version, globals.Namespace)

	labelSelector, avail = os.LookupEnv("VAULT_LABEL_SELECTOR")
	if !avail {
		log.Fatalf("No label selector has been provided. Please provide the label selector in `VAULT_LABEL_SELECTOR` key.")
	}

	balancerPortStr, avail := os.LookupEnv("BALANCER_PORT")
	if !avail {
		log.Warnf("Balancer port is not specified. Please provide the balancer port in `BALANCER_PORT` key. Now the default will be used. BALANCER_PORT: %v", globals.DefaultBalancerPort)
		balancerPort = globals.DefaultBalancerPort
	} else {
		balancerPort, _ = strconv.Atoi(balancerPortStr)
	}

	globals.HttpTimeout, avail = os.LookupEnv("HTTP_TIMEOUT")
	if !avail {
		log.Warnf("No http timeout duration is specified. Please provide in `HTTP_TIMEOUT` key. Now the default time out will be used. HTTP_TIMEOUT: %v Minutes", globals.DefaultTimeOut)
		globals.HttpTimeout = globals.DefaultTimeOut
	}
}

var (
	versionLogger = log.WithFields(log.Fields{"vlb_version": BuildVersion})
	BuildVersion  = "dev"
	balancerPort  int
	vaultPool     types.VaultPool
	labelSelector string
	avail         bool
)

const (
	HealthCheckPath = ":8200/v1/sys/seal-status"
	ProxyPath       = ":8200"
)

func main() {
	go startRoutine(context.Background())

	// start the balancer http service
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", balancerPort),
		Handler: http.HandlerFunc(loadBalance),
	}
	//
	versionLogger.Infof("Vault Balancer started and running at :%d", balancerPort)
	if err := server.ListenAndServe(); err != nil {
		versionLogger.Fatalf("error while starting the load balance, %v", err)
	}
}

// startRoutine starts the routine work of collecting IPs, setting up reverse
// proxies and doing health check.
func startRoutine(context context.Context) {
	versionLogger.Info("Starting the routines for discovery, proxy setup and health check")
	t := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-t.C:
			ipAddressMap := helper.GetVaultIPsFromLabelSelectors(labelSelector, versionLogger)
			setUpProxies(ipAddressMap)
			healthCheck(vaultPool)
		}
	}
}

// loadBalance load balances the incoming request
func loadBalance(w http.ResponseWriter, r *http.Request) {
	attempts := helper.GetAttemptsFromContext(r)
	if attempts > 3 {
		versionLogger.Infof("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

	peer := vaultPool.GetNextPod()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

// setUpProxies will create the reverse proxies for the identified IPs
func setUpProxies(serviceNameAndIP map[string]struct{}) {
	for podIP := range serviceNameAndIP {
		if !vaultPool.IsInThePool(podIP) {
			sanitizedIP := strings.TrimSpace(podIP)
			vaultUrl, err := url.Parse("http://" + sanitizedIP + ProxyPath)
			if err != nil {
				versionLogger.Errorf("error occurred while converting string to URL for proxy path. error: %v", err)
			}
			healthUrl, _ := url.Parse("http://" + sanitizedIP + HealthCheckPath)

			proxy := httputil.NewSingleHostReverseProxy(vaultUrl)
			proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
				versionLogger.Infof("[%s] %s\n", vaultUrl.Host, e.Error())
				retries := helper.GetRetryFromContext(request)
				if retries < 3 {
					select {
					case <-time.After(5 * time.Millisecond):
						ctx := context.WithValue(request.Context(), helper.Retry, retries+1)
						proxy.ServeHTTP(writer, request.WithContext(ctx))
					}
					return
				}

				// mark the ip address as not alice after 3 attempts
				vaultPool.MarkVaultPodStatus(vaultUrl, false)

				attempts := helper.GetAttemptsFromContext(request)
				versionLogger.Infof("Retry attempt for the %s(%s): %d\n", request.RemoteAddr, request.URL.Path, attempts)
				ctx := context.WithValue(request.Context(), helper.Attempts, attempts+1)
				loadBalance(writer, request.WithContext(ctx))
			}
			vaultPool.AddBackend(&types.VaultBackend{
				IP:           sanitizedIP,
				ProxyURL:     vaultUrl,
				HealthURL:    healthUrl,
				Alive:        true,
				ReverseProxy: proxy,
			})
			versionLogger.Infof("The service IP %s has been configured", vaultUrl)
		} else {
			versionLogger.Infof("Pod IP %v is already configured.", podIP)
		}
	}

	var toBeRemoved []*types.VaultBackend
	for _, b := range vaultPool.VaultBackends {
		if _, ok := serviceNameAndIP[b.IP]; !ok {
			toBeRemoved = append(toBeRemoved, b)
		}
	}
	for _, b := range toBeRemoved {
		versionLogger.Infof("Retiring the backed with IP %v from load balancing", b.IP)
		vaultPool.RetireBackend(b)
	}
}

// healthCheck runs a routine for check status of the pods every 2 mins
func healthCheck(vaultPool types.VaultPool) {

	versionLogger.Info("Starting health check...")
	vaultPool.HealthCheck()
	versionLogger.Info("Health check completed")
}
