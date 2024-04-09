package themis

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type MpcParticipant struct {
	Id      string `json:"id"`
	Moniker string `json:"moniker"`
	Pubkey  []byte `json:"key"`
}

type MpcInfoResponse struct {
	Id           string            `json:"mpc_id"`
	Threshold    int               `json:"threshold"`
	Address      common.Address    `json:"mpc_address"`
	Pubkey       []byte            `json:"mpc_pubkey"`
	AddrType     MpcAddrType       `json:"mpc_type"`
	Participants []*MpcParticipant `json:"participants"`
}

func (bs *Client) LatestMpcInfo(ctx context.Context, addrType MpcAddrType) (*MpcInfoResponse, error) {
	var result MpcInfoResponse
	if _, err := bs.Get(ctx, fmt.Sprintf("/mpc/latest/%d", addrType), &result); err != nil {
		return nil, fmt.Errorf("LatestMpcInfo: %w", err)
	}
	return &result, nil
}

type SignByMpcReq struct {
	SignID   string      `json:"sign_id"`
	MpcID    string      `json:"mpc_id"`
	SignType MpcSignType `json:"sign_type"`
	SignData string      `json:"sign_data"` // binary encoded raw transaction in hex string
	SignMsg  string      `json:"sign_msg"`  // the hashed data of SignData for signing in hex string
}

type SignByMpcRes struct {
	Tx string `json:"tx"`
}

func (bs *Client) InitSign(ctx context.Context, req *SignByMpcReq) error {
	var res SignByMpcRes
	if err := bs.Post(ctx, "/mpc/propose-mpc-sign", req, &res); err != nil {
		return fmt.Errorf("SignByMpc: %w", err)
	}
	return nil
}

type MpcSignResp struct {
	SignID    string      `json:"sign_id"`
	MpcID     string      `json:"mpc_id"`
	SignType  MpcSignType `json:"sign_type"`
	SignData  []byte      `json:"sign_data"`
	SignMsg   []byte      `json:"sign_msg"`
	Signature []byte      `json:"signature"`
	SignedTx  []byte      `json:"signed_tx"`
	Proposer  string      `json:"proposer"`
}

func (bs *Client) GetMpcSign(ctx context.Context, id string) (*MpcSignResp, error) {
	var result MpcSignResp
	if _, err := bs.Get(ctx, fmt.Sprintf("/mpc/sign/%s", id), &result); err != nil {
		return nil, fmt.Errorf("GetMpcSign: %w", err)
	}
	return &result, nil
}
