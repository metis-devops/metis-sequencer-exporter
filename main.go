package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/metis-devops/metis-sequencer-exporter/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		ScrapeInterval time.Duration
		ConfPath       string
	)

	flag.DurationVar(&ScrapeInterval, "scrape.interval", time.Second*15, "scrape interval")
	flag.StringVar(&ConfPath, "config", "config.yaml", "config path")
	flag.Parse()

	conf, err := config.Parse(ConfPath)
	if err != nil {
		slog.Error("config", "path", ConfPath, "err", err)
		os.Exit(1)
	}

	basectx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	reg := prometheus.NewRegistry()

	seq, err := NewSeqMetric(basectx, reg, conf)
	if err != nil {
		slog.Error("NewSeqMetrics", "err", err)
		os.Exit(1)
	}

	bal, err := NewBalanceMetric(basectx, reg, conf)
	if err != nil {
		slog.Error("NewBalanceMetric", "err", err)
		os.Exit(1)
	}

	scrapeFailuresMetric := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "metis_sequencer_exporter_failures",
			Help: "Number of scrape errors.",
		},
		[]string{"svc_name"},
	)
	reg.MustRegister(scrapeFailuresMetric)

	go seq.Scrape(basectx, scrapeFailuresMetric, ScrapeInterval)
	go bal.Scrape(basectx, scrapeFailuresMetric, ScrapeInterval)

	server := &http.Server{Addr: ":9090"}
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "pong") })

	go func() {
		slog.Info("serving")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cancel()
			slog.Error("ListenAndServe", "err", err)
		}
	}()

	<-basectx.Done()
	slog.Info("graceful stopping")
	_ = server.Shutdown(context.Background())
}
