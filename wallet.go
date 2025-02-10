package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/metis-devops/metis-sequencer-exporter/internal/config"
	"github.com/metis-devops/metis-sequencer-exporter/internal/themis"
	"github.com/metis-devops/metis-sequencer-exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type WalletMetric struct {
	l1rpc   *ethclient.Client
	l2rpc   *ethclient.Client
	wallets map[string]common.Address

	balance *prometheus.GaugeVec
	nonce   *prometheus.CounterVec

	mutex    sync.Mutex
	nonceMap map[string]float64
	logger   *slog.Logger
}

func NewWalletMetric(basectx context.Context, reg prometheus.Registerer, conf *config.Config) (*WalletMetric, error) {
	if conf.Wallet == nil {
		return nil, nil
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil)).With("module", "wallet")
	ctx, cancel := context.WithTimeout(basectx, time.Minute)
	defer cancel()

	logger.Info("connect to l2geth", "url", conf.Wallet.L2Geth)
	l2rpc, err := ethclient.DialContext(ctx, conf.Wallet.L2Geth)
	if err != nil {
		return nil, fmt.Errorf("connect to l2geth %s", conf.Wallet.L2Geth)
	}

	logger.Info("connect to l1geth", "url", conf.Wallet.L1Geth)
	l1rpc, err := ethclient.DialContext(ctx, conf.Wallet.L1Geth)
	if err != nil {
		return nil, fmt.Errorf("connect to l1geth %s", conf.Wallet.L1Geth)
	}

	wallets := make(map[string]common.Address)
	for name, wallet := range conf.Wallet.Wallets {
		wallets[name] = wallet
		logger.Info("Add custom wallet", "name", name, "wallet", wallet)
	}

	if conf.Wallet.Themis == "" {
		logger.Warn("mpc wallet metric is disabled")
		if len(conf.Wallet.Wallets) == 0 {
			return nil, nil
		}
	} else {
		logger.Info("connect to themis", "url", conf.Wallet.Themis)
		pos, err := themis.NewClient(conf.Wallet.Themis)
		if err != nil {
			return nil, fmt.Errorf("connect to themis %s", conf.Wallet.L1Geth)
		}
		for i := themis.CommonMpcAddr; i <= themis.BlobSubmitMpcAddr; i++ {
			res, err := pos.LatestMpcInfo(ctx, i)
			if err != nil {
				slog.Error("failed to get mpc address", "addr", i.String(), "err", err)
				continue
			}

			if _, ok := wallets[i.String()]; ok {
				return nil, fmt.Errorf("custom wallet is duplicated with mpc address %s", i)
			}

			wallets[i.String()] = res.Address
		}
	}

	balance := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "metis:sequencer:wallet:balance",
		Help: "Balance of mpc and custom addresses from config",
	}, []string{"chain", "addr", "alias"})

	nonce := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "metis:sequencer:wallet:nonce",
		Help: "Nonce of mpc and custom addresses from config",
	}, []string{"chain", "addr", "alias"})

	reg.MustRegister(balance, nonce)

	return &WalletMetric{
		l1rpc:    l1rpc,
		l2rpc:    l2rpc,
		wallets:  wallets,
		balance:  balance,
		nonce:    nonce,
		nonceMap: make(map[string]float64),
		logger:   logger,
	}, nil
}

func (m *WalletMetric) Scrape(basectx context.Context, failureCounter *prometheus.CounterVec, scrapeInterval time.Duration) {
	if m == nil {
		slog.Warn("wallet metric is disabled")
		return
	}
	go m.scrapeL2(basectx, failureCounter, scrapeInterval)
	go m.scrapeL1(basectx, failureCounter, scrapeInterval)
}

func (m *WalletMetric) scrapeL2(basectx context.Context, failureCounter *prometheus.CounterVec, scrapeInterval time.Duration) {
	ticker := time.NewTimer(0)
	defer ticker.Stop()

	scrape := func(name string, addr common.Address) error {
		labels := prometheus.Labels{"chain": "metis", "addr": addr.Hex(), "alias": name}
		nonceKey := fmt.Sprintf("metis:%s", name)

		newctx, cancel := context.WithTimeout(basectx, time.Minute)
		defer cancel()

		wei, err := m.l2rpc.BalanceAt(newctx, addr, nil)
		if err != nil {
			return fmt.Errorf("failed to get balance: %s", err)
		}
		balance := utils.ToEther(wei)

		nonce, err := m.l2rpc.NonceAt(newctx, addr, nil)
		if err != nil {
			return fmt.Errorf("failed to get nonce: %s", err)
		}

		m.logger.Info("wallet", "chain", "metis", "alias", name, "addr", addr, "balance", balance, "nonce", nonce)

		m.balance.With(labels).Set(balance)

		m.mutex.Lock()
		defer m.mutex.Unlock()
		if v, ok := m.nonceMap[nonceKey]; !ok && nonce == 0 {
			m.nonce.With(labels).Add(0)
			m.nonceMap[nonceKey] = 0
		} else if t := float64(nonce) - v; t > 0 {
			m.nonce.With(labels).Add(t)
			m.nonceMap[nonceKey] += t
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
			for name, addr := range m.wallets {
				wg.Add(1)
				name, addr := name, addr
				go func() {
					if err := scrape(name, addr); err != nil {
						failureCounter.With(prometheus.Labels{"svc_name": "metis_balance"}).Inc()
						m.logger.Error("scrape metis wallet metrics", "addr", name, "err", err)
					}
					wg.Done()
				}()
			}
			wg.Wait()
			m.logger.Info("Done", "target", "metis_wallet", "duration", time.Since(start))
			ticker.Reset(scrapeInterval)
		}
	}
}

func (m *WalletMetric) scrapeL1(basectx context.Context, failureCounter *prometheus.CounterVec, scrapeInterval time.Duration) {
	ticker := time.NewTimer(0)
	defer ticker.Stop()

	scrape := func(name string, addr common.Address) error {
		labels := prometheus.Labels{"chain": "eth", "addr": addr.Hex(), "alias": name}
		nonceKey := fmt.Sprintf("eth:%s", name)

		newctx, cancel := context.WithTimeout(basectx, time.Minute)
		defer cancel()

		wei, err := m.l1rpc.BalanceAt(newctx, addr, nil)
		if err != nil {
			return fmt.Errorf("failed to get balance: %s", err)
		}
		balance := utils.ToEther(wei)

		nonce, err := m.l1rpc.NonceAt(newctx, addr, nil)
		if err != nil {
			return fmt.Errorf("failed to get nonce: %s", err)
		}

		m.logger.Info("wallet", "chain", "eth", "alias", name, "addr", addr, "balance", balance, "nonce", nonce)

		m.balance.With(labels).Set(balance)

		m.mutex.Lock()
		defer m.mutex.Unlock()
		if v, ok := m.nonceMap[nonceKey]; !ok && nonce == 0 {
			m.nonce.With(labels).Add(0)
			m.nonceMap[nonceKey] = 0
		} else if t := float64(nonce) - v; t > 0 {
			m.nonce.With(labels).Add(t)
			m.nonceMap[nonceKey] += t
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
			for alias, addr := range m.wallets {
				wg.Add(1)
				alias, addr := alias, addr
				go func() {
					if err := scrape(alias, addr); err != nil {
						failureCounter.With(prometheus.Labels{"svc_name": "eth_balance"}).Inc()
						m.logger.Error("scrape eth wallet metrics", "alias", alias, "err", err)
					}
					wg.Done()
				}()
			}
			wg.Wait()
			m.logger.Info("Done", "target", "eth_wallet", "duration", time.Since(start))
			ticker.Reset(scrapeInterval)
		}
	}
}
