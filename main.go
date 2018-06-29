package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v2/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/loadbalancers"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "openstack"
)

type Exporter struct {
	up     prometheus.Gauge
	router *prometheus.GaugeVec
	volume *prometheus.GaugeVec
	lb     *prometheus.GaugeVec
	server *prometheus.GaugeVec

	routerDurationSeconds

	computeClient      *gophercloud.ServiceClient
	networkingClient   *gophercloud.ServiceClient
	blockstorageClient *gophercloud.ServiceClient
}

func main() {
	var (
		listenAddress = flag.String("web.listen-address", ":9401", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	)
	flag.Parse()

	prometheus.MustRegister(NewExporter())

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Openstack Status Exporter</title></head>
             <body>
             <h1>OpenStack Status Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	fmt.Println("Starting HTTP server on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}

func NewExporter() *Exporter {
	return &Exporter{
		up: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "up",
				Help:      "OpenStack Status Collection Operational",
			},
		),
		router: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "router",
				Name:      "status_total",
				Help:      "Status of OpenStack Routers",
			},
			[]string{"status"},
		),
		volume: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "volume",
				Name:      "status_total",
				Help:      "Status of OpenStack Volumes",
			},
			[]string{"status"},
		),
		lb: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "lb",
				Name:      "status_total",
				Help:      "Status of OpenStack Load Balancers",
			},
			[]string{"status"},
		),
		server: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "server",
				Name:      "status_total",
				Help:      "Status of OpenStack Servers",
			},
			[]string{"status"},
		),
	}
}

func (e *Exporter) Collect(metrics chan<- prometheus.Metric) {
	e.up.Set(1)

	if err := e.auth(); err != nil {
		log.Printf("Failed to authenticate: %s\n", err)
		e.up.Set(0)
		e.up.Collect(metrics)
		return
	}

	if err := e.collectRouters(); err != nil {
		log.Printf("Failed to collect routers: %s\n", err)
		e.up.Set(0)
	}

	if err := e.collectVolumes(); err != nil {
		log.Printf("Failed to collect volumes: %s\n", err)
		e.up.Set(0)
	}

	//if err := e.collectServers(); err != nil {
	//  log.Printf("Failed to collect volumes: %s\n", err)
	//  e.up.Set(0)
	//}

	if err := e.collectLoadBalancers(); err != nil {
		log.Printf("Failed to collect lbs: %s\n", err)
		e.up.Set(0)
	}

	e.up.Collect(metrics)
	e.router.Collect(metrics)
	e.volume.Collect(metrics)
	e.lb.Collect(metrics)
	e.server.Collect(metrics)
}

func (e *Exporter) Describe(descs chan<- *prometheus.Desc) {
	e.up.Describe(descs)
	e.router.Describe(descs)
	e.server.Describe(descs)
	e.lb.Describe(descs)
	e.volume.Describe(descs)
}

func (e *Exporter) auth() error {
	endpointOpts := gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	}

	authOpts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return fmt.Errorf("could not get auth options from ENV: %v", err)
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return fmt.Errorf("could not authenticate: %v", err)
	}

	e.computeClient, err = openstack.NewComputeV2(provider, endpointOpts)
	if err != nil {
		return fmt.Errorf("could not initialize compute client: %v", err)
	}

	e.networkingClient, err = openstack.NewNetworkV2(provider, endpointOpts)
	if err != nil {
		return fmt.Errorf("could not initialize network client: %v", err)
	}

	e.blockstorageClient, err = openstack.NewBlockStorageV2(provider, endpointOpts)
	if err != nil {
		return fmt.Errorf("could not initialize blockStorage client: %v", err)
	}

	return nil
}

func (e *Exporter) collectRouters() error {
	return routers.List(e.networkingClient, routers.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		if list, err := routers.ExtractRouters(page); err != nil {
			return false, err
		} else {
			for _, router := range list {
				e.router.WithLabelValues(router.Status).Inc()
			}
		}
		return true, nil
	})
}

func (e *Exporter) collectVolumes() error {
	return volumes.List(e.blockstorageClient, volumes.ListOpts{AllTenants: true}).EachPage(func(page pagination.Page) (bool, error) {
		if list, err := volumes.ExtractVolumes(page); err != nil {
			return false, err
		} else {
			for _, volume := range list {
				e.volume.WithLabelValues(volume.Status).Inc()
			}
		}
		return true, nil
	})
}

func (e *Exporter) collectServers() error {
	return servers.List(e.computeClient, servers.ListOpts{AllTenants: true}).EachPage(func(page pagination.Page) (bool, error) {
		if list, err := servers.ExtractServers(page); err != nil {
			return false, err
		} else {
			for _, server := range list {
				e.server.WithLabelValues(server.Status).Inc()
			}
		}
		return true, nil
	})
}

func (e *Exporter) collectLoadBalancers() error {
	return loadbalancers.List(e.networkingClient, loadbalancers.ListOpts{}).EachPage(func(page pagination.Page) (bool, error) {
		if list, err := loadbalancers.ExtractLoadBalancers(page); err != nil {
			return false, err
		} else {
			for _, lb := range list {
				e.lb.WithLabelValues(lb.ProvisioningStatus).Inc()
			}
		}
		return true, nil
	})
}
