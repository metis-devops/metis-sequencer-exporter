package themis

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type Validator struct {
	ID               uint64         `json:"ID"`
	StartBatch       uint64         `json:"startBatch"`
	EndBatch         uint64         `json:"endBatch"`
	Nonce            uint64         `json:"nonce"`
	VotingPower      int64          `json:"power"`
	PubKey           string         `json:"pubKey"`
	Signer           common.Address `json:"signer"`
	LastUpdated      string         `json:"last_updated"`
	Jailed           bool           `json:"jailed"`
	ProposerPriority int64          `json:"accum"`
}

type ValidatorSet struct {
	Validators []*Validator `json:"validators"`
	Proposer   *Validator   `json:"proposer"`
}

type SpanResp struct {
	ID           uint64       `json:"span_id" yaml:"span_id"`
	StartBlock   uint64       `json:"start_block" yaml:"start_block"`
	EndBlock     uint64       `json:"end_block" yaml:"end_block"`
	ValidatorSet ValidatorSet `json:"validator_set" yaml:"validator_set"`
	Producers    []*Validator `json:"selected_producers" yaml:"selected_producers"`
	ChainID      string       `json:"metis_chain_id" yaml:"metis_chain_id"`
}

func (bs *Client) LatestEpoch(ctx context.Context) (int64, *SpanResp, error) {
	var result SpanResp
	height, err := bs.Get(ctx, "/metis/latest-span", &result)
	if err != nil {
		return 0, nil, fmt.Errorf("LatestEpoch: %w", err)
	}
	return height, &result, nil
}

func (bs *Client) GetEpochByID(ctx context.Context, id int64) (int64, *SpanResp, error) {
	var result SpanResp
	height, err := bs.Get(ctx, fmt.Sprintf("/metis/span/%d", id), &result)
	if err != nil {
		return 0, nil, fmt.Errorf("GetEpochByID: %w", err)
	}
	return height, &result, nil
}
