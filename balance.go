package main

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metis-devops/metis-sequencer-exporter/internal/config"
	"github.com/metis-devops/metis-sequencer-exporter/internal/themis"
	"github.com/metis-devops/metis-sequencer-exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type BalanceMetric struct {
	l1rpc   *ethclient.Client
	l2rpc   *ethclient.Client
	themis  *themis.Client
	wallets map[string]common.Address

	metric *prometheus.GaugeVec
}

func NewBalanceMetric(basectx context.Context, reg prometheus.Registerer, conf *config.Config) (*BalanceMetric, error) {
	if conf.Balance == nil {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(basectx, time.Minute)
	defer cancel()

	l2rpc, err := ethclient.DialContext(ctx, conf.Balance.L2Geth)
	if err != nil {
		return nil, err
	}

	l1rpc, err := ethclient.DialContext(ctx, conf.Balance.L1Geth)
	if err != nil {
		return nil, err
	}

	pos, err := themis.NewClient(conf.Balance.Themis)
	if err != nil {
		return nil, err
	}

	wallets := make(map[string]common.Address)
	maps.Copy(conf.Balance.Wallets, wallets)

	for i := themis.CommonMpcAddr; i <= themis.RewardSubmitMpcAddr; i++ {
		res, err := pos.LatestMpcInfo(ctx, i)
		if err != nil {
			return nil, fmt.Errorf("get mpc address %s: %w", i, err)
		}

		if _, ok := wallets[i.String()]; ok {
			return nil, fmt.Errorf("custom wallet is duplicated with mpc address %s", i)
		}

		wallets[i.String()] = res.Address
	}

	metric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "metis:sequencer:balance",
		Help: "Balance of mpc and custom addresses from config",
	}, []string{"chain", "addr", "addr_alias"})
	reg.MustRegister(metric)

	return &BalanceMetric{
		l1rpc:   l1rpc,
		l2rpc:   l2rpc,
		themis:  pos,
		wallets: wallets,
		metric:  metric,
	}, nil
}

func (m *BalanceMetric) Scrape(basectx context.Context, failureCounter *prometheus.CounterVec, scrapeInterval time.Duration) {
	if m == nil {
		slog.Warn("balance metric is disabled")
		return
	}
	m.scrapeL2Balance(basectx, failureCounter, scrapeInterval)
	m.scrapeL1Balance(basectx, failureCounter, scrapeInterval)
}

func (m *BalanceMetric) scrapeL2Balance(basectx context.Context, failureCounter *prometheus.CounterVec, scrapeInterval time.Duration) {
	ticker := time.NewTimer(0)
	defer ticker.Stop()

	scrape := func(name string, addr common.Address) error {
		newctx, cancel := context.WithTimeout(basectx, time.Minute)
		defer cancel()

		wei, err := m.l2rpc.BalanceAt(newctx, addr, nil)
		if err != nil {
			return fmt.Errorf("failed to get l1 balance: %s", err)
		}

		balance := utils.ToEther(wei)

		slog.Info("balance", "chain", "metis", "alias", name, "addr", addr, "balance", balance)
		m.metric.With(prometheus.Labels{"chain": "metis", "addr": addr.Hex(), "addr_alias": name}).Set(balance)
		return nil
	}

	for {
		select {
		case <-basectx.Done():
			return
		case <-ticker.C:
			var wg sync.WaitGroup
			var start = time.Now()
			for name, addr := range m.wallets {
				wg.Add(1)
				name, addr := name, addr
				go func() {
					if err := scrape(name, addr); err != nil {
						failureCounter.With(prometheus.Labels{"svc_name": "metis_balance"}).Inc()
						slog.Error("scrape metis balance metrics", "addr", name, "err", err)
					}
					wg.Done()
				}()
			}
			wg.Wait()
			slog.Info("Done", "target", "metis_balance", "duration", time.Since(start))
			ticker.Reset(scrapeInterval)
		}
	}
}

func (m *BalanceMetric) scrapeL1Balance(basectx context.Context, failureCounter *prometheus.CounterVec, scrapeInterval time.Duration) {
	ticker := time.NewTimer(0)
	defer ticker.Stop()

	scrape := func(name string, addr common.Address) error {
		newctx, cancel := context.WithTimeout(basectx, time.Minute)
		defer cancel()

		wei, err := m.l1rpc.BalanceAt(newctx, addr, nil)
		if err != nil {
			return fmt.Errorf("failed to get l1 balance: %s", err)
		}

		balance := utils.ToEther(wei)
		slog.Info("balance", "chain", "eth", "alias", name, "addr", addr, "balance", balance)
		m.metric.With(prometheus.Labels{"chain": "eth", "addr": addr.Hex(), "addr_alias": name}).Set(balance)
		return nil
	}

	for {
		select {
		case <-basectx.Done():
			return
		case <-ticker.C:
			var wg sync.WaitGroup
			var start = time.Now()
			for name, addr := range m.wallets {
				wg.Add(1)
				name, addr := name, addr
				go func() {
					if err := scrape(name, addr); err != nil {
						failureCounter.With(prometheus.Labels{"svc_name": "eth_balance"}).Inc()
						slog.Error("scrape eth balance metrics", "addr", name, "err", err)
					}
					wg.Done()
				}()
			}
			wg.Wait()
			slog.Info("Done", "target", "eth_balance", "duration", time.Since(start))
			ticker.Reset(scrapeInterval)
		}
	}
}
