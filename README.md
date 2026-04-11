# storj-exporter

Prometheus exporter for [Storj](https://www.storj.io/) storage node operators. It scrapes the node's local web dashboard API (`:14002`) and exposes node health, storage, bandwidth, per-satellite stats, audit scores and payout information as Prometheus metrics. Multiple nodes can be scraped from a single exporter instance.

## Metrics

All metrics are labeled with `node` (the name from config). Selected metrics:

- `storj_node_up` — 1 if the last scrape succeeded
- `storj_node_uptime_seconds`, `storj_node_up_to_date`, `storj_node_quic_ok`
- `storj_node_last_pinged_timestamp_seconds`, `storj_node_last_quic_pinged_timestamp_seconds`
- `storj_node_info{node_id,version,allowed_version}` — static info, always 1
- `storj_disk_space{type}` — `used`, `free`, `allocated`, `trash`, `overused`, `reclaimable`
- `storj_node_bandwidth_bytes{type}` — node-level bandwidth
- `storj_bandwidth_by_type_bytes_total{satellite,type}` — per-satellite ingress/egress/total (counter)
- `storj_satellite_storage_summary_byte_hours`, `storj_satellite_storage_average_bytes`
- `storj_audit_score{satellite,type}` — `audit`, `online`, `suspension`
- `storj_satellite_status{satellite,type}` — `disqualified`, `suspended`, `vetted`
- `storj_satellite_price_model{satellite,type}` — as reported by the node
- `storj_node_payout_current_month_dollars{type}` and `_previous_month_dollars{type}` — `total`, `disk_space`, `egress_bandwidth`, `egress_audit_bandwidth`, `held`
- `storj_node_payout_current_month_expected_dollars` — estimated total for the current month

The exporter listens on `:9100` and serves metrics at `/metrics`.

## Configuration

Configuration is loaded via [go-conf](https://github.com/ThomasObenaus/go-conf) and supports a YAML file, CLI flags and environment variables (prefix `STORJ_EXPORTER_`).

Example [config/dev.yml](config/dev.yml):

```yaml
nodes:
  - name: storj
    base_url: http://192.168.88.89:14002
  - name: storj-2
    base_url: http://192.168.88.90:14002
```

Run with a config file:

```sh
storj-exporter --config-file=config/dev.yml
```

## Running

### From source

```sh
go run ./cmd/exporter --config-file=config/dev.yml
```

### Docker

```sh
docker run -d \
  --name storj-exporter \
  -p 9100:9100 \
  -v $(pwd)/config/dev.yml:/storj-exporter/config.yml \
  alexnav/storj-exporter:latest \
  ./storj-exporter --config-file=/storj-exporter/config.yml
```

Building and pushing a multi-arch image:

```sh
make docker-push
```

## Prometheus scrape config

```yaml
scrape_configs:
  - job_name: storj
    static_configs:
      - targets: ['storj-exporter:9100']
```

## Grafana

A starter dashboard is available at [grafana/dashboard.json](grafana/dashboard.json).
