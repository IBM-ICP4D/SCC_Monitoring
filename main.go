package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	securityv1client "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"

	"k8s.io/client-go/rest"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	securityClient securityv1client.SecurityV1Interface // Security Client to retrieve SCCs

	sccMap = make(map[string]string) // Contains the mappings to compare current scc's with expected sa bindings

	sccGuage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "scc_users",
			Help: "SecurityContextConstraints and user details",
		},
		[]string{"name", "users"})
)

func init() {
	err := json.Unmarshal([]byte(os.Getenv("SCC_MAPPINGS")), &sccMap)
	if err != nil {
		panic(fmt.Errorf("Failed to retrieve scc mappings from env, %s", err.Error()))
	}
	securityClient = getOSSecurityClient()
	prometheus.MustRegister(sccGuage)
}

func getOSSecurityClient() securityv1client.SecurityV1Interface {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(fmt.Errorf("Failed to retrieve kubernetes configuration, %s", err.Error()))
	}

	securityClient, err := securityv1client.NewForConfig(config)
	if err != nil {
		panic(fmt.Errorf("Failed to create a OpenShift Security client, %s", err.Error()))
	}
	return securityClient
}

func getSCC() {
	sccList, err := securityClient.SecurityContextConstraints().List(metav1.ListOptions{})
	if err != nil {
		panic(fmt.Errorf("Failed to retrieve scc's, %s", err.Error()))
	}

	for _, scc := range sccList.Items {
		users := strings.Join(scc.Users, ",")

		sccLabels := prometheus.Labels{
			"name":  scc.Name,
			"users": users,
		}

		if expectedUsers, ok := sccMap[scc.Name]; ok {
			if expectedUsers == users {
				sccGuage.With(sccLabels).Set(1)
			} else {
				sccGuage.With(sccLabels).Set(0)
			}
		} else {
			sccGuage.With(sccLabels).Set(2)
		}
	}
}

func recordMetrics() {
	go func() {
		for {
			getSCC()
			time.Sleep(5 * time.Second)
		}
	}()
}

func main() {
	recordMetrics()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8080", nil)
}
