// Command technocore-monitor is a Prometheus exporter for a technocore.chat
// instance. It scrapes public endpoints and exposes room/activity metrics so you
// can alert on liveness, growth and per-room throughput.
//
// Usage:
//
//	TECHNOCORE_URL=https://technocore.chat \
//	TECHNOCORE_ROOMS=lobby,announcements \
//	technocore-monitor            # serves /metrics on :9184
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type collector struct {
	baseURL string
	rooms   []string
	http    *http.Client

	up          *prometheus.Desc
	roomLastSeq *prometheus.Desc
	roomCount   *prometheus.Desc
	roomsTotal  *prometheus.Desc
	scrapeDur   *prometheus.Desc
}

func newCollector(baseURL string, rooms []string) *collector {
	return &collector{
		baseURL: strings.TrimRight(baseURL, "/"),
		rooms:   rooms,
		http:    &http.Client{Timeout: 10 * time.Second},
		up:          prometheus.NewDesc("technocore_up", "1 if the instance answered this scrape.", nil, nil),
		roomLastSeq: prometheus.NewDesc("technocore_room_last_seq", "Highest message sequence seen in a room.", []string{"room"}, nil),
		roomCount:   prometheus.NewDesc("technocore_room_messages_window", "Messages returned in the latest read window for a room.", []string{"room"}, nil),
		roomsTotal:  prometheus.NewDesc("technocore_rooms_total", "Number of rooms listed by /rooms.", nil, nil),
		scrapeDur:   prometheus.NewDesc("technocore_scrape_duration_seconds", "Duration of the last scrape.", nil, nil),
	}
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up
	ch <- c.roomLastSeq
	ch <- c.roomCount
	ch <- c.roomsTotal
	ch <- c.scrapeDur
}

type roomResponse struct {
	Count   int `json:"count"`
	LastSeq int `json:"last_seq"`
}

func (c *collector) getJSON(path string, v any) error {
	req, _ := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "technocore-monitor/1.0")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	up := 1.0

	for _, room := range c.rooms {
		var r roomResponse
		if err := c.getJSON("/r/"+room+"?format=json", &r); err != nil {
			up = 0
			continue
		}
		ch <- prometheus.MustNewConstMetric(c.roomLastSeq, prometheus.GaugeValue, float64(r.LastSeq), room)
		ch <- prometheus.MustNewConstMetric(c.roomCount, prometheus.GaugeValue, float64(r.Count), room)
	}

	// /rooms may be an array or an object with a list; count defensively.
	var raw json.RawMessage
	if err := c.getJSON("/rooms?format=json", &raw); err == nil {
		if n := countRooms(raw); n >= 0 {
			ch <- prometheus.MustNewConstMetric(c.roomsTotal, prometheus.GaugeValue, float64(n))
		}
	}

	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, up)
	ch <- prometheus.MustNewConstMetric(c.scrapeDur, prometheus.GaugeValue, time.Since(start).Seconds())
}

func countRooms(raw json.RawMessage) int {
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil {
		return len(arr)
	}
	var obj struct {
		Rooms []json.RawMessage `json:"rooms"`
		Count int               `json:"count"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		if len(obj.Rooms) > 0 {
			return len(obj.Rooms)
		}
		return obj.Count
	}
	return -1
}

func main() {
	base := getenv("TECHNOCORE_URL", "https://technocore.chat")
	rooms := strings.Split(getenv("TECHNOCORE_ROOMS", "lobby"), ",")
	for i := range rooms {
		rooms[i] = strings.TrimSpace(rooms[i])
	}
	addr := getenv("LISTEN_ADDR", ":9184")

	prometheus.MustRegister(newCollector(base, rooms))
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`technocore-monitor — <a href="/metrics">/metrics</a>`))
	})
	log.Printf("technocore-monitor scraping %s rooms=%v, serving %s/metrics", base, rooms, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
