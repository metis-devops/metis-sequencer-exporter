package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v3"
)

type Sequencer struct {
	L1DTL  string `json:"l1dtl,omitempty" yaml:"l1dtl,omitempty"`
	Themis string `json:"themis,omitempty" yaml:"themis,omitempty"`
	L2Geth string `json:"l2geth" yaml:"l2geth"`
}

type Wallet struct {
	Themis  string                    `json:"themis,omitempty" yaml:"themis,omitempty"`
	L2Geth  string                    `json:"l2geth" yaml:"l2geth"`
	L1Geth  string                    `json:"l1geth" yaml:"l1geth"`
	Wallets map[string]common.Address `json:"wallets" yaml:"wallets"`
}

type Config struct {
	Sequencers map[string]*Sequencer `json:"sequencer" yaml:"sequencer"`
	Wallet     *Wallet               `json:"wallet,omitempty" yaml:"wallet,omitempty"`
}

func Parse(p string) (*Config, error) {
	file, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	conf := new(Config)
	switch ext := path.Ext(p); ext {
	case ".json":
		err = json.Unmarshal(file, &conf)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(file, &conf)
	default:
		err = fmt.Errorf("not supported file extension %s", ext)
	}
	if err != nil {
		return nil, err
	}
	return conf, nil
}
