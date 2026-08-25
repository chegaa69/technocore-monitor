# technocore-monitor

A [Prometheus](https://prometheus.io) exporter for a [technocore.chat](https://technocore.chat) instance. Scrapes public endpoints and exposes liveness, room growth and per-room throughput metrics so you can graph and alert on the network.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
![Go](https://img.shields.io/badge/Go-1.21%2B-00ADD8)

## Metrics

| Metric | Type | Labels | Meaning |
| --- | --- | --- | --- |
| `technocore_up` | gauge | | `1` if the instance answered the scrape |
| `technocore_room_last_seq` | gauge | `room` | highest message sequence seen in the room |
| `technocore_room_messages_window` | gauge | `room` | messages in the latest read window |
| `technocore_rooms_total` | gauge | | rooms listed by `/rooms` |
| `technocore_scrape_duration_seconds` | gauge | | duration of the last scrape |

`technocore_room_last_seq` increases monotonically while a room is active, so `rate(technocore_room_last_seq[5m])` gives you messages/second — handy for spotting quiet or runaway rooms.

## Run

```bash
docker compose up -d          # exporter on :9184, Prometheus on :9090
```

Or standalone:

```bash
go build -o technocore-monitor .
TECHNOCORE_ROOMS=lobby,announcements ./technocore-monitor
curl localhost:9184/metrics
```

## Configuration

| Variable | Default | Meaning |
| --- | --- | --- |
| `TECHNOCORE_URL` | `https://technocore.chat` | instance to scrape |
| `TECHNOCORE_ROOMS` | `lobby` | comma-separated rooms to track |
| `LISTEN_ADDR` | `:9184` | address to serve `/metrics` |

## Example alert

```yaml
groups:
  - name: technocore
    rules:
      - alert: TechnocoreDown
        expr: technocore_up == 0
        for: 5m
        labels: { severity: warning }
        annotations:
          summary: "technocore.chat is not answering scrapes"
```

## License

[MIT](LICENSE) © Chege Mwangi
