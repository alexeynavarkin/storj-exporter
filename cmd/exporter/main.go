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
	up             *prometheus.Desc
	uptime         *prometheus.Desc
	upToDate       *prometheus.Desc
	lastPinged     *prometheus.Desc
	lastQuicPinged *prometheus.Desc
	quicOK         *prometheus.Desc
	nodeInfo       *prometheus.Desc

	payoutCurrent  *prometheus.Desc
	payoutPrevious *prometheus.Desc
	payoutExpected *prometheus.Desc

	bandwidthBytes   *prometheus.Desc
	bandwidthNode    *prometheus.Desc
	storageBytes     *prometheus.Desc
	storageSummary   *prometheus.Desc
	storageAverage   *prometheus.Desc
	auditScore       *prometheus.Desc
	satelliteStatus  *prometheus.Desc
	satellitePrice   *prometheus.Desc

	nodeClients map[string]*storj.Client

	lg *zap.Logger
}

func NewStorjExporter(nodeClients map[string]*storj.Client, lg *zap.Logger) *StorjExporter {
	return &StorjExporter{
		nodeClients: nodeClients,
		up: prometheus.NewDesc(
			"storj_node_up",
			"1 if the node API responded successfully during the last scrape.",
			[]string{"node"}, nil,
		),
		uptime: prometheus.NewDesc(
			"storj_node_uptime_seconds",
			"Node uptime in seconds.",
			[]string{"node"}, nil,
		),
		upToDate: prometheus.NewDesc(
			"storj_node_up_to_date",
			"1 if the node is running a version considered up to date.",
			[]string{"node"}, nil,
		),
		lastPinged: prometheus.NewDesc(
			"storj_node_last_pinged_timestamp_seconds",
			"Unix timestamp of the last successful ping.",
			[]string{"node"}, nil,
		),
		lastQuicPinged: prometheus.NewDesc(
			"storj_node_last_quic_pinged_timestamp_seconds",
			"Unix timestamp of the last successful QUIC ping.",
			[]string{"node"}, nil,
		),
		quicOK: prometheus.NewDesc(
			"storj_node_quic_ok",
			"1 if the node's QUIC status is OK.",
			[]string{"node"}, nil,
		),
		nodeInfo: prometheus.NewDesc(
			"storj_node_info",
			"Static node information, always 1.",
			[]string{"node", "node_id", "version", "allowed_version"}, nil,
		),
		payoutCurrent: prometheus.NewDesc(
			"storj_node_payout_current_month_dollars",
			"Current month payout breakdown in dollars.",
			[]string{"node", "type"}, nil,
		),
		payoutPrevious: prometheus.NewDesc(
			"storj_node_payout_previous_month_dollars",
			"Previous month payout breakdown in dollars.",
			[]string{"node", "type"}, nil,
		),
		payoutExpected: prometheus.NewDesc(
			"storj_node_payout_current_month_expected_dollars",
			"Expected total payout for the current month in dollars.",
			[]string{"node"}, nil,
		),
		bandwidthBytes: prometheus.NewDesc(
			"storj_bandwidth_by_type",
			"Per-satellite bandwidth usage in bytes since the beginning of the month.",
			[]string{"node", "satellite", "type"}, nil,
		),
		bandwidthNode: prometheus.NewDesc(
			"storj_node_bandwidth_bytes",
			"Node-level bandwidth usage in bytes.",
			[]string{"node", "type"}, nil,
		),
		storageBytes: prometheus.NewDesc(
			"storj_disk_space",
			"Disk space by type in bytes. Types: used, free (api `available`), allocated, trash, overused, reclaimable.",
			[]string{"node", "type"}, nil,
		),
		storageSummary: prometheus.NewDesc(
			"storj_satellite_storage_summary_byte_hours",
			"Storage summary for the satellite in byte*hours.",
			[]string{"node", "satellite"}, nil,
		),
		storageAverage: prometheus.NewDesc(
			"storj_satellite_storage_average_bytes",
			"Average stored bytes for the satellite.",
			[]string{"node", "satellite"}, nil,
		),
		auditScore: prometheus.NewDesc(
			"storj_audit_score",
			"Node audit, suspension and online scores.",
			[]string{"node", "satellite", "type"}, nil,
		),
		satelliteStatus: prometheus.NewDesc(
			"storj_satellite_status",
			"Per-satellite status flags. Types: disqualified, suspended, vetted.",
			[]string{"node", "satellite", "type"}, nil,
		),
		satellitePrice: prometheus.NewDesc(
			"storj_satellite_price_model",
			"Per-satellite price model in USD per TB per month (as reported by the node).",
			[]string{"node", "satellite", "type"}, nil,
		),
		lg: lg,
	}
}

func (e *StorjExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.up
	ch <- e.uptime
	ch <- e.upToDate
	ch <- e.lastPinged
	ch <- e.lastQuicPinged
	ch <- e.quicOK
	ch <- e.nodeInfo
	ch <- e.payoutCurrent
	ch <- e.payoutPrevious
	ch <- e.payoutExpected
	ch <- e.bandwidthBytes
	ch <- e.bandwidthNode
	ch <- e.storageBytes
	ch <- e.storageSummary
	ch <- e.storageAverage
	ch <- e.auditScore
	ch <- e.satelliteStatus
	ch <- e.satellitePrice
}

func boolFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func (e *StorjExporter) collectPayout(ctx context.Context, ch chan<- prometheus.Metric, name string, cl *storj.Client) {
	payoutRes, err := cl.GetSnoPayout(ctx)
	if err != nil {
		e.lg.Error("failed to request payout info", zap.String("node", name), zap.Error(err))
		return
	}

	emit := func(desc *prometheus.Desc, m *struct {
		total, disk, egress, egressAudit, held float64
	}) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, m.total, name, "total")
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, m.disk, name, "disk_space")
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, m.egress, name, "egress_bandwidth")
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, m.egressAudit, name, "egress_audit_bandwidth")
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, m.held, name, "held")
	}

	cur := payoutRes.CurrentMonth
	prev := payoutRes.PreviousMonth
	emit(e.payoutCurrent, &struct {
		total, disk, egress, egressAudit, held float64
	}{cur.Payout, cur.DiskSpacePayout, cur.EgressBandwidthPayout, cur.EgressRepairAuditPayout, cur.Held})
	emit(e.payoutPrevious, &struct {
		total, disk, egress, egressAudit, held float64
	}{prev.Payout, prev.DiskSpacePayout, prev.EgressBandwidthPayout, prev.EgressRepairAuditPayout, prev.Held})

	ch <- prometheus.MustNewConstMetric(
		e.payoutExpected, prometheus.GaugeValue,
		payoutRes.CurrentMonthExpectations, name,
	)
}

func (e *StorjExporter) collectNode(ctx context.Context, ch chan<- prometheus.Metric, name string, cl *storj.Client, wg *sync.WaitGroup) {
	snoRes, err := cl.GetSno(ctx)
	if err != nil {
		e.lg.Error("failed to scrape node", zap.String("node", name), zap.Error(err))
		ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, 0, name)
		return
	}
	ch <- prometheus.MustNewConstMetric(e.up, prometheus.GaugeValue, 1, name)

	ds := snoRes.DiskSpace
	for label, v := range map[string]float64{
		"used":        ds.Used,
		"free":        ds.Available,
		"allocated":   ds.Allocated,
		"trash":       ds.Trash,
		"overused":    ds.Overused,
		"reclaimable": ds.Reclaimable,
	} {
		ch <- prometheus.MustNewConstMetric(e.storageBytes, prometheus.GaugeValue, v, name, label)
	}

	ch <- prometheus.MustNewConstMetric(e.bandwidthNode, prometheus.GaugeValue, snoRes.Bandwidth.Used, name, "used")
	ch <- prometheus.MustNewConstMetric(e.bandwidthNode, prometheus.GaugeValue, snoRes.Bandwidth.Available, name, "available")

	ch <- prometheus.MustNewConstMetric(e.upToDate, prometheus.GaugeValue, boolFloat(snoRes.UpToDate), name)
	ch <- prometheus.MustNewConstMetric(e.quicOK, prometheus.GaugeValue, boolFloat(snoRes.QuicStatus == "OK"), name)
	ch <- prometheus.MustNewConstMetric(e.nodeInfo, prometheus.GaugeValue, 1, name, snoRes.NodeID, snoRes.Version, snoRes.AllowedVersion)

	if startedAt, err := time.Parse(time.RFC3339, snoRes.StartedAt); err == nil {
		ch <- prometheus.MustNewConstMetric(e.uptime, prometheus.GaugeValue, time.Since(startedAt).Seconds(), name)
	} else {
		e.lg.Error("failed to parse startedAt", zap.String("node", name), zap.Error(err))
	}
	if t, err := time.Parse(time.RFC3339, snoRes.LastPingedAt); err == nil {
		ch <- prometheus.MustNewConstMetric(e.lastPinged, prometheus.GaugeValue, float64(t.Unix()), name)
	}
	if t, err := time.Parse(time.RFC3339, snoRes.LastQuicPingedAt); err == nil {
		ch <- prometheus.MustNewConstMetric(e.lastQuicPinged, prometheus.GaugeValue, float64(t.Unix()), name)
	}

	for _, sat := range snoRes.Satellites {
		ch <- prometheus.MustNewConstMetric(e.satelliteStatus, prometheus.GaugeValue, boolFloat(sat.Disqualified != nil), name, sat.URL, "disqualified")
		ch <- prometheus.MustNewConstMetric(e.satelliteStatus, prometheus.GaugeValue, boolFloat(sat.Suspended != nil), name, sat.URL, "suspended")
		ch <- prometheus.MustNewConstMetric(e.satelliteStatus, prometheus.GaugeValue, boolFloat(sat.VettedAt != nil), name, sat.URL, "vetted")

		wg.Add(1)
		satID, satURL := sat.ID, sat.URL
		go func() {
			defer wg.Done()
			satRes, err := cl.GetSnoSattilite(ctx, satID)
			if err != nil {
				e.lg.Error("failed to scrape satellite", zap.String("node", name), zap.String("satellite", satURL), zap.Error(err))
				return
			}

			ch <- prometheus.MustNewConstMetric(e.bandwidthBytes, prometheus.GaugeValue, float64(satRes.IngressSummary), name, satURL, "ingress")
			ch <- prometheus.MustNewConstMetric(e.bandwidthBytes, prometheus.GaugeValue, float64(satRes.EgressSummary), name, satURL, "egress")
			ch <- prometheus.MustNewConstMetric(e.bandwidthBytes, prometheus.GaugeValue, float64(satRes.BandwidthSummary), name, satURL, "total")

			ch <- prometheus.MustNewConstMetric(e.storageSummary, prometheus.GaugeValue, satRes.StorageSummary, name, satURL)
			ch <- prometheus.MustNewConstMetric(e.storageAverage, prometheus.GaugeValue, satRes.AverageUsageBytes, name, satURL)

			ch <- prometheus.MustNewConstMetric(e.auditScore, prometheus.GaugeValue, satRes.Audits.AuditScore, name, satURL, "audit")
			ch <- prometheus.MustNewConstMetric(e.auditScore, prometheus.GaugeValue, satRes.Audits.OnlineScore, name, satURL, "online")
			ch <- prometheus.MustNewConstMetric(e.auditScore, prometheus.GaugeValue, satRes.Audits.SuspensionScore, name, satURL, "suspension")

			pm := satRes.PriceModel
			ch <- prometheus.MustNewConstMetric(e.satellitePrice, prometheus.GaugeValue, float64(pm.EgressBandwidth), name, satURL, "egress_bandwidth")
			ch <- prometheus.MustNewConstMetric(e.satellitePrice, prometheus.GaugeValue, float64(pm.RepairBandwidth), name, satURL, "repair_bandwidth")
			ch <- prometheus.MustNewConstMetric(e.satellitePrice, prometheus.GaugeValue, float64(pm.AuditBandwidth), name, satURL, "audit_bandwidth")
			ch <- prometheus.MustNewConstMetric(e.satellitePrice, prometheus.GaugeValue, float64(pm.DiskSpace), name, satURL, "disk_space")
		}()
	}
}

func (e *StorjExporter) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	wg := sync.WaitGroup{}
	for name, cl := range e.nodeClients {
		wg.Add(2)
		go func() {
			defer wg.Done()
			e.collectPayout(ctx, ch, name, cl)
		}()
		go func() {
			defer wg.Done()
			e.collectNode(ctx, ch, name, cl, &wg)
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
