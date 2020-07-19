package main

import (
	"context"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
	"vault-balancer/helper"
	"vault-balancer/types"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
}

var vaultPool types.VaultPool

func main() {
	var vaultList string
	var port int
	flag.StringVar(&vaultList, "vaults", "", "Comma separated Vault pod IPs that needs to be load balanced")
	flag.IntVar(&port, "port", 8000, "Port to serve for load balancer")
	flag.Parse()

	if len(vaultList) == 0 {
		log.Fatal("Provide one or more Vault Pod IPs (comma separated) to load balance")
	}

	//Getting the individual Vault Pod IPs
	indVaultIpList := strings.Split(vaultList, ",")
	log.Infof("%d Vault IPs (%s) obtained for the load balancing ", len(indVaultIpList), vaultList)

	for _, individualIP := range indVaultIpList {
		vaultUrl, err := url.Parse(individualIP)
		if err != nil {
			log.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(vaultUrl)
		proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
			log.Printf("[%s] %s\n", vaultUrl.Host, e.Error())
			retries := helper.GetRetryFromContext(request)
			if retries < 3 {
				select {
				case <-time.After(10 * time.Millisecond):
					ctx := context.WithValue(request.Context(), helper.Retry, retries+1)
					proxy.ServeHTTP(writer, request.WithContext(ctx))
				}
				return
			}

			// after 3 retries, mark this pod as down
			vaultPool.MarkVaultPodStatus(vaultUrl, false)

			// if the same request routing for few attempts with different backends, increase the count
			attempts := helper.GetAttemptsFromContext(request)
			log.Infof("Retry attempt for the %s(%s): %d\n", request.RemoteAddr, request.URL.Path, attempts)
			ctx := context.WithValue(request.Context(), helper.Attempts, attempts+1)
			loadBalance(writer, request.WithContext(ctx))
		}
		vaultPool.AddBackend(&types.VaultBackend{
			URL:          vaultUrl,
			Alive:        true,
			ReverseProxy: proxy,
		})
		log.Infof("Configured the server: %s", vaultUrl)
	}

	// create http server
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(loadBalance),
	}

	// start health checking
	go helper.HealthCheck()

	log.Infof("Load Balancer started at :%d", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("error while starting the load balance, %v", err)
	}
}


// loadBalance load balances the incoming request
func loadBalance(w http.ResponseWriter, r *http.Request) {
	attempts := helper.GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Infof("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
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
