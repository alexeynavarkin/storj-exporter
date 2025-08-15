package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	config "github.com/ThomasObenaus/go-conf"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/alexeynavarkin/storj-exporter/internal/storj"
)

type Config struct {
	Nodes []struct {
		BaseURL string `cfg:"{'name':'base_url'}"`
		Name    string `cfg:"{'name':'name'}"`
	} `cfg:"{'name':'nodes'}"`
}

type StorjExporter struct {
	uptime         *prometheus.Desc
	bandwidthBytes *prometheus.Desc
	storageBytes   *prometheus.Desc
	auditScore     *prometheus.Desc

	nodeClients map[string]*storj.Client

	lg *zap.Logger
}

func NewStorjExporter(nodeClients map[string]*storj.Client, lg *zap.Logger) *StorjExporter {
	return &StorjExporter{
		nodeClients: nodeClients,
		uptime: prometheus.NewDesc(
			"storj_node_uptime_seconds",
			"Node uptime in seconds",
			[]string{"node"},
			nil,
		),
		bandwidthBytes: prometheus.NewDesc(
			"storj_bandwidth_by_type",
			"Total bandwidth ingress/egress in bytes.",
			[]string{"node", "satellite", "type"},
			nil,
		),
		storageBytes: prometheus.NewDesc(
			"storj_disk_space",
			"Total space by type in bytes.",
			[]string{"node", "type"},
			nil,
		),
		auditScore: prometheus.NewDesc(
			"storj_audit_score",
			"Node audit score.",
			[]string{"node", "satellite", "type"},
			nil,
		),
		lg: lg,
	}
}

func (e *StorjExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.uptime
	ch <- e.bandwidthBytes
	ch <- e.storageBytes
	ch <- e.auditScore
}

func (e *StorjExporter) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	for name, cl := range e.nodeClients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*90)
			defer cancel()

			snoRes, err := cl.GetSno(ctx)
			if err != nil {
				e.lg.Error("failed to scrape node", zap.Error(err))
				return
			}

			ch <- prometheus.MustNewConstMetric(
				e.storageBytes,
				prometheus.CounterValue,
				float64(snoRes.DiskSpace.Available),
				name, "available",
			)
			ch <- prometheus.MustNewConstMetric(
				e.storageBytes,
				prometheus.CounterValue,
				float64(snoRes.DiskSpace.Used),
				name, "used",
			)
			ch <- prometheus.MustNewConstMetric(
				e.storageBytes,
				prometheus.CounterValue,
				float64(snoRes.DiskSpace.Trash),
				name, "trash",
			)
			ch <- prometheus.MustNewConstMetric(
				e.storageBytes,
				prometheus.CounterValue,
				float64(snoRes.DiskSpace.Overused),
				name, "overused",
			)

			startedAt, err := time.Parse(time.RFC3339, snoRes.StartedAt)
			if err != nil {
				e.lg.Error("failed to parse startedAt", zap.Error(err))
				return
			}
			ch <- prometheus.MustNewConstMetric(
				e.uptime,
				prometheus.CounterValue,
				time.Since(startedAt).Seconds(),
				name,
			)

			for _, sat := range snoRes.Satellites {
				wg.Add(1)
				go func() {
					defer wg.Done()
					satRes, err := cl.GetSnoSattilite(ctx, sat.ID)
					if err != nil {
						e.lg.Error("failed to scrape sat", zap.Error(err))
						return
					}

					ch <- prometheus.MustNewConstMetric(
						e.bandwidthBytes,
						prometheus.CounterValue,
						float64(satRes.IngressSummary),
						name, sat.URL, "ingress",
					)
					ch <- prometheus.MustNewConstMetric(
						e.bandwidthBytes,
						prometheus.CounterValue,
						float64(satRes.EgressSummary),
						name, sat.URL, "egress",
					)

					ch <- prometheus.MustNewConstMetric(
						e.auditScore,
						prometheus.GaugeValue,
						float64(satRes.Audits.AuditScore),
						name, sat.URL, "audit",
					)
					ch <- prometheus.MustNewConstMetric(
						e.auditScore,
						prometheus.GaugeValue,
						float64(satRes.Audits.OnlineScore),
						name, sat.URL, "online",
					)
					ch <- prometheus.MustNewConstMetric(
						e.auditScore,
						prometheus.GaugeValue,
						float64(satRes.Audits.SuspensionScore),
						name, sat.URL, "suspension",
					)
				}()
			}
		}()
	}
	wg.Wait()
}

func main() {
	lg, _ := zap.NewProduction()

	cfg := Config{}
	cfgProvider, err := config.NewConfigProvider(
		&cfg,
		"STORJ_EXPORTER",
		"STORJ_EXPORTER",
	)
	if err != nil {
		log.Fatal("failed to build config provider %w", err)
	}
	err = cfgProvider.ReadConfig(os.Args)
	if err != nil {
		log.Println("failed to load config", err)
		log.Println(cfgProvider.Usage())
		os.Exit(-1)
	}

	nodeClients := make(map[string]*storj.Client)
	for _, node := range cfg.Nodes {
		nodeClients[node.Name] = storj.NewClient(
			storj.Config{
				BaseURL: node.BaseURL,
			},
		)
	}

	exporter := NewStorjExporter(nodeClients, lg)
	prometheus.MustRegister(exporter)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Storj Node Exporter</title></head>
			<body>
			<h1>Storj Node Exporter</h1>
			<p><a href="/metrics">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Printf("Starting Storj node exporter on %s", ":9100")
	log.Fatal(http.ListenAndServe(":9100", nil))
}
