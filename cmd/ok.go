package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	Version        = "0.0.0"
	ipEnv          = getEnv("IP", "0.0.0.0")
	portEnv        = getEnv("PORT", "8080")
	metricsIpEnv   = getEnv("METRICS_IP", ipEnv)
	metricsPortEnv = getEnv("METRICS_PORT", "2112")
	nodeName       = getEnv("NODE_NAME", "")
	podName        = getEnv("POD_NAME", "")
	podIP          = getEnv("POD_IP", "")
	namespace      = getEnv("NAMESPACE", "")
	serviceAccount = getEnv("SERVICE_ACCOUNT", "")
)

// Ok is used to return a json object
type Ok struct {
	sync.Mutex
	Version        string `json:"version"`
	NodeName       string `json:"node_name"`
	PodName        string `json:"pod_name"`
	PodIP          string `json:"pod_ip"`
	Namespace      string `json:"namespace"`
	ServiceAccount string `json:"service_account"`
	Count          int    `json:"count"`
}

// main
func main() {
	var (
		ip          = flag.String("ip", ipEnv, "Server IP address to bind to.")
		port        = flag.String("port", portEnv, "Server port.")
		metricsPort = flag.String("metricsPort", metricsPortEnv, "Metrics port.")
		metricsIp   = flag.String("metricsIP", metricsIpEnv, "Falls back to same IP as server.")
	)

	flag.Parse()

	zapCfg := zap.NewProductionConfig()
	zapCfg.DisableCaller = true
	zapCfg.DisableStacktrace = true

	logger, err := zapCfg.Build()
	if err != nil {
		fmt.Printf("Can not build logger: %s\n", err.Error())
		os.Exit(1)
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())

		logger.Info("Starting Ok Metrics Server. ("+*metricsIp+":"+*metricsPort+"/metrics)",
			zap.String("type", "metrics_startup"),
			zap.String("port", *metricsPort),
			zap.String("ip", *metricsIp),
		)

		err = http.ListenAndServe(*metricsIp+":"+*metricsPort, nil)
		if err != nil {
			logger.Fatal("Error Starting Ok Metrics Server", zap.Error(err))
			os.Exit(1)
		}
	}()

	logger.Info("Starting Ok Server",
		zap.String("type", "startup"),
		zap.String("port", *port),
		zap.String("ip", *ip),
	)

	gin.SetMode(gin.ReleaseMode)

	// gin router
	r := gin.New()

	// gin zap logger middleware
	r.Use(ginzap.Ginzap(logger, time.RFC3339, true))

	// prepare response payload
	ok := &Ok{
		Version:        Version,
		Namespace:      namespace,
		NodeName:       nodeName,
		PodName:        podName,
		PodIP:          podIP,
		ServiceAccount: serviceAccount,
		Count:          0,
	}

	// lash returns ok
	r.GET("/", func(c *gin.Context) {
		ok.Lock()
		defer ok.Unlock()
		ok.Count++

		c.JSON(200, ok)
	})

	s := &http.Server{
		Addr:           *ip + ":" + *port,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	err = s.ListenAndServe()
	if err != nil {
		logger.Fatal("Could not start Ok server", zap.Error(err))
	}
}

// getEnv gets an environment variable or sets a default if
// one does not exist.
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}

	return value
}
