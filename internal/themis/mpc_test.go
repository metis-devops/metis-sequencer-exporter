package themis

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestClient_LatestMpcInfo(t *testing.T) {
	base64Decode := func(s string) []byte {
		data, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			t.Fatal("invalid base64 string")
			return nil
		}
		return data
	}

	tests := []struct {
		name     string
		testdata string
		args     MpcAddrType
		want     *MpcInfoResponse
		wantErr  bool
	}{
		{
			name:     "ok",
			testdata: "mpc-info.json",
			args:     CommonMpcAddr,
			want: &MpcInfoResponse{
				Id:        "4e83e51b-f019-4c5f-a68e-175a6afe1075",
				Threshold: 1,
				Address:   common.HexToAddress("0x48120daed4f33ad803b19e4e237c4180a4043045"),
				Pubkey:    base64Decode("AhvBzH3102WlwfXmg1HspPJJqre22ZUAZbNpAYjzWelo"),
				AddrType:  CommonMpcAddr,
				Participants: []*MpcParticipant{
					{
						Id:      "16Uiu2HAmAHXrEoYHBVcPcW8MkMDoVVjTvcPCPNUdzvAwUWpwL23r",
						Moniker: "0xabb8407d97fd40410f91ae9ef80df3aa45e9affa",
						Pubkey:  base64Decode("AtzJdup4whlC5LDym/pgVFf8DjBW/ShIJ7nnEG4SSsVB"),
					},
					{
						Id:      "16Uiu2HAmDHsXhskjkUSq1NUrW3S7CxFwuz7cMeo3HEe7BG8aGQs9",
						Moniker: "0x87fb79e80599392b0069b9b472e53fd32d53a9fd",
						Pubkey:  base64Decode("Awlyxy8lQtjp/VqshUigshwOukrWHq5l+WR6LiAzGesg"),
					},
					{
						Id:      "16Uiu2HAmPTogjoRusC42T7zjzVKwj4PEkbgQR2oX1m4xuu5k3oqT",
						Moniker: "0xd0b814096cb9ca5e140a7616a2885c5abfb71bc1",
						Pubkey:  base64Decode("A6CTG9QJY0owCcHWICA/k4CcuJW1si8Srb/24U195L/y"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("method should be GET, but got %s", r.Method)
					return
				}

				if path := fmt.Sprintf("/mpc/latest/%d", tt.args); r.URL.Path != path {
					t.Errorf("expected url path %s got url path %s", path, r.URL.Path)
					return
				}

				if tt.wantErr {
					w.WriteHeader(http.StatusBadRequest)
					w.Header().Add("content-type", "application/json")
					_ = json.NewEncoder(w).Encode(&ErrorResponse{Error: "error"})
				} else {
					jsonResp, err := os.Open(fmt.Sprintf("testdata/%s", tt.testdata))
					if err != nil {
						t.Errorf("can't read test file: %s", err)
						return
					}
					defer jsonResp.Close() //nolint:errcheck
					_, _ = io.Copy(w, jsonResp)
				}
			}))
			defer server.Close()

			bs := &Client{
				restHost:   server.URL,
				httpClient: http.DefaultClient,
			}

			got, err := bs.LatestMpcInfo(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.LatestMpcInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.LatestMpcInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetMpcSign(t *testing.T) {
	base64Decode := func(s string) []byte {
		data, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			t.Fatal("invalid base64 string")
			return nil
		}
		return data
	}

	tests := []struct {
		name     string
		testdata string
		args     string
		want     *MpcSignResp
		wantErr  bool
	}{
		{
			name:     "ok",
			testdata: "mpc-sign.json",
			args:     "2b92db63-0e6e-47f5-aaab-1ee76dd7668e",
			want: &MpcSignResp{
				SignID:    "2b92db63-0e6e-47f5-aaab-1ee76dd7668e",
				MpcID:     "5ca928c2-85c5-4d1a-8005-8a217eb47ab5",
				SignType:  BatchRewardSignType,
				SignData:  base64Decode("dGVzdG9ubHk="),
				SignMsg:   base64Decode("yChgW0EpOWNFhxkZRsEyyxVJYHBHEkBE1yldidQIJhw="),
				Proposer:  "0x87fb79e80599392b0069b9b472e53fd32d53a9fd",
				Signature: base64Decode("cmVkYWN0ZWQ="),
				SignedTx:  base64Decode("dGVzdG9ubHk="),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("method should be GET, but got %s", r.Method)
					return
				}

				if path := fmt.Sprintf("/mpc/sign/%s", tt.args); r.URL.Path != path {
					t.Errorf("expected url path %s got url path %s", path, r.URL.Path)
					return
				}

				if tt.wantErr {
					w.WriteHeader(http.StatusBadRequest)
					w.Header().Add("content-type", "application/json")
					_ = json.NewEncoder(w).Encode(&ErrorResponse{Error: "error"})
				} else {
					jsonResp, err := os.Open(fmt.Sprintf("testdata/%s", tt.testdata))
					if err != nil {
						t.Errorf("can't read test file: %s", err)
						return
					}
					defer jsonResp.Close() //nolint:errcheck
					_, _ = io.Copy(w, jsonResp)
				}
			}))
			defer server.Close()

			bs := &Client{
				restHost:   server.URL,
				httpClient: http.DefaultClient,
			}

			got, err := bs.GetMpcSign(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetMpcSign() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.GetMpcSign() = %v, want %v", got, tt.want)
			}
		})
	}
}
