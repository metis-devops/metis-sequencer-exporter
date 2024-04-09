package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metis-devops/metis-sequencer-exporter/internal/config"
	"github.com/metis-devops/metis-sequencer-exporter/internal/dtl"
	"github.com/metis-devops/metis-sequencer-exporter/internal/themis"
	"github.com/prometheus/client_golang/prometheus"
)

type SequencerClient struct {
	l2rpc          *ethclient.Client
	dtl            *dtl.Client
	themis         *themis.Client
	lastHeights    map[string]float64
	lastTimestamps map[string]float64
	mutex          sync.Mutex
}

type SequencerMetric struct {
	clients    map[string]*SequencerClient
	timestamps *prometheus.CounterVec
	heights    *prometheus.CounterVec
}

func NewSeqMetric(basectx context.Context, reg prometheus.Registerer, conf *config.Config) (*SequencerMetric, error) {
	ctx, cancel := context.WithTimeout(basectx, time.Minute)
	defer cancel()

	var clients = make(map[string]*SequencerClient)
	for name, ep := range conf.Sequencers {
		client := &SequencerClient{
			lastHeights:    make(map[string]float64),
			lastTimestamps: make(map[string]float64),
		}

		var err error
		client.l2rpc, err = ethclient.DialContext(ctx, ep.L2Geth)
		if err != nil {
			return nil, err
		}

		if ep.L1DTL != "" {
			client.dtl, err = dtl.NewClient(ep.L1DTL)
			if err != nil {
				return nil, err
			}
		}

		if ep.Themis != "" {
			client.themis, err = themis.NewClient(ep.Themis)
			if err != nil {
				return nil, err
			}
		}

		clients[name] = client
	}

	m := &SequencerMetric{
		clients: clients,
		timestamps: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "metis:sequencer:timestamp",
				Help: "Current Unix timestamp of the service.",
			},
			[]string{"svc_name", "seq_name"},
		),
		heights: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "metis:sequencer:height",
				Help: "Current block number of the service.",
			},
			[]string{"svc_name", "seq_name"},
		),
	}

	reg.MustRegister(m.timestamps)
	reg.MustRegister(m.heights)
	return m, nil
}

func (m *SequencerMetric) Scrape(basectx context.Context, failureCounter *prometheus.CounterVec, scrapeInterval time.Duration) {
	go m.scrapeL2gethMetrics(basectx, failureCounter, scrapeInterval)
	go m.scrapeThemisMetrics(basectx, failureCounter, scrapeInterval)
	go m.scrapeL1DTLMetrics(basectx, failureCounter, scrapeInterval)
}

func (m *SequencerMetric) scrapeL2gethMetrics(basectx context.Context, failureCounter *prometheus.CounterVec, scrapeInterval time.Duration) {
	ticker := time.NewTimer(0)
	defer ticker.Stop()

	scrape := func(name string, client *SequencerClient) error {
		newctx, cancel := context.WithTimeout(basectx, time.Minute)
		defer cancel()

		header, err := client.l2rpc.HeaderByNumber(newctx, nil)
		if err != nil {
			return fmt.Errorf("failed to get l2 height: %s", err)
		}

		slog.Info("l2geth", "name", name, "height", header.Number, "timestamp", header.Time)

		client.mutex.Lock()
		defer client.mutex.Unlock()

		if t := float64(header.Time) - client.lastTimestamps["l2geth"]; t > 0 {
			m.timestamps.With(prometheus.Labels{"svc_name": "l2geth", "seq_name": name}).Add(t)
			client.lastTimestamps["l2geth"] += t
		}

		if t := float64(header.Number.Int64()) - client.lastHeights["l2geth"]; t > 0 {
			m.heights.With(prometheus.Labels{"svc_name": "l2geth", "seq_name": name}).Add(t)
			client.lastHeights["l2geth"] += t
		}

		return nil
	}

	for {
		select {
		case <-basectx.Done():
			return
		case <-ticker.C:
			var wg sync.WaitGroup
			var start = time.Now()
			for name, client := range m.clients {
				wg.Add(1)
				name, client := name, client
				go func() {
					if err := scrape(name, client); err != nil {
						failureCounter.With(prometheus.Labels{"svc_name": fmt.Sprintf("seq-%s-l2geth", name)}).Inc()
						slog.Error("scrape l2geth metrics", "seq", name, "err", err)
					}
					wg.Done()
				}()
			}
			wg.Wait()
			slog.Info("Done", "target", "l2geth", "duration", time.Since(start))
			ticker.Reset(scrapeInterval)
		}
	}
}

func (m *SequencerMetric) scrapeThemisMetrics(basectx context.Context, failureCounter *prometheus.CounterVec, scrapeInterval time.Duration) {
	if enabled := func() bool {
		for _, i := range m.clients {
			if i.themis != nil {
				return true
			}
		}
		return false
	}(); !enabled {
		slog.Warn("PoS metric is disabled")
		return
	}

	ticker := time.NewTimer(0)
	defer ticker.Stop()

	scrape := func(name string, client *SequencerClient) error {
		if client.themis == nil {
			return nil
		}

		newctx, cancel := context.WithTimeout(basectx, time.Minute)
		defer cancel()

		height, epoch, err := client.themis.LatestEpoch(newctx)
		if err != nil {
			return fmt.Errorf("failed to get epoch info: %s", err)
		}

		slog.Info("themis", "name", name, "height", height, "span", epoch.ID)

		client.mutex.Lock()
		defer client.mutex.Unlock()

		if t := float64(height) - client.lastHeights["themis"]; t > 0 {
			m.heights.With(prometheus.Labels{"svc_name": "themis", "seq_name": name}).Add(t)
			client.lastHeights["themis"] += t
		}

		return nil
	}

	for {
		select {
		case <-basectx.Done():
			return
		case <-ticker.C:
			var wg sync.WaitGroup
			var start = time.Now()
			for name, client := range m.clients {
				wg.Add(1)
				name, client := name, client
				go func() {
					if err := scrape(name, client); err != nil {
						failureCounter.With(prometheus.Labels{"svc_name": fmt.Sprintf("seq-%s-themis", name)}).Inc()
						slog.Error("scrape themis metrics", "seq", name, "err", err)
					}
					wg.Done()
				}()
			}
			wg.Wait()
			slog.Info("Done", "target", "themis", "duration", time.Since(start))
			ticker.Reset(scrapeInterval)
		}
	}
}

func (m *SequencerMetric) scrapeL1DTLMetrics(basectx context.Context, failureCounter *prometheus.CounterVec, scrapeInterval time.Duration) {
	if enabled := func() bool {
		for _, i := range m.clients {
			if i.dtl != nil {
				return true
			}
		}
		return false
	}(); !enabled {
		slog.Warn("DTL metric is disabled")
		return
	}

	ticker := time.NewTimer(0)
	defer ticker.Stop()

	scrape := func(name string, client *SequencerClient) error {
		if client.dtl == nil {
			return nil
		}

		newctx, cancel := context.WithTimeout(basectx, time.Minute)
		defer cancel()

		height, err := client.dtl.GetL1HighestSynced(newctx)
		if err != nil {
			return fmt.Errorf("failed to l1dtl status: %s", err)
		}

		slog.Info("l1dtl", "name", name, "height", height)

		client.mutex.Lock()
		defer client.mutex.Unlock()

		if t := float64(height) - client.lastHeights["l1dtl"]; t > 0 {
			m.heights.With(prometheus.Labels{"svc_name": "l1dtl", "seq_name": name}).Add(t)
			client.lastHeights["l1dtl"] += t
		}

		return nil
	}

	for {
		select {
		case <-basectx.Done():
			return
		case <-ticker.C:
			var wg sync.WaitGroup
			var start = time.Now()
			for name, client := range m.clients {
				wg.Add(1)
				name, client := name, client
				go func() {
					if err := scrape(name, client); err != nil {
						failureCounter.With(prometheus.Labels{"svc_name": fmt.Sprintf("seq-%s-l1dtl", name)}).Inc()
						slog.Error("scrape l1dtl metrics", "seq", name, "err", err)
					}
					wg.Done()
				}()
			}
			wg.Wait()
			slog.Info("Done", "target", "l1dtl", "duration", time.Since(start))
			ticker.Reset(scrapeInterval)
		}
	}
}
