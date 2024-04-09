package themis

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestClient_Get(t *testing.T) {
	type args struct {
		path   string
		result any
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok-0",
			args: args{path: "/ok", result: float64(100)},
		},
		{
			name: "ok-1",
			args: args{path: "/ok", result: "ok"},
		},
		{
			name:    "err",
			args:    args{path: "/err"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("method should be GET, but got %s", r.Method)
					return
				}

				if r.URL.Path != tt.args.path {
					t.Errorf("expected url path %s got url path %s", tt.args.path, r.URL.Path)
					return
				}

				if tt.wantErr {
					w.WriteHeader(http.StatusBadRequest)
					w.Header().Add("content-type", "application/json")
					_ = json.NewEncoder(w).Encode(&ErrorResponse{Error: "error"})
				} else {
					result, _ := json.Marshal(tt.args.result)
					_ = json.NewEncoder(w).Encode(&ResponseWithHeight{Result: result})
				}
			}))
			defer server.Close()

			c := &Client{
				restHost:   server.URL,
				httpClient: http.DefaultClient,
			}

			var res any
			if _, err := c.Get(context.Background(), tt.args.path, &res); (err != nil) != tt.wantErr {
				t.Errorf("Client.Get() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(res, tt.args.result) {
				t.Errorf("Client.Get() = %v, want %v", res, tt.args.result)
			}
		})
	}
}

func TestClient_Post(t *testing.T) {
	type ReqData struct {
		Any string `json:"any"`
	}

	type args struct {
		path   string
		req    ReqData
		result any
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok-0",
			args: args{path: "/ok", req: ReqData{"test1"}, result: float64(100)},
		},
		{
			name: "ok-1",
			args: args{path: "/ok", req: ReqData{"test2"}, result: "ok"},
		},
		{
			name:    "err",
			args:    args{path: "/err", req: ReqData{"test2"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("method should be POST, but got %s", r.Method)
					return
				}

				if r.URL.Path != tt.args.path {
					t.Errorf("expected url path %s got url path %s", tt.args.path, r.URL.Path)
					return
				}

				if header := r.Header.Get("content-type"); header != "application/json" {
					t.Errorf("expected contene-type header application/json but got %q", header)
					return
				}

				var req ReqData
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("couldn't decode the request to ReqData %s", err)
					return
				}

				if !reflect.DeepEqual(req, tt.args.req) {
					t.Errorf("expected body = %v got = %v", tt.args.req, req)
					return
				}

				if tt.wantErr {
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(&ErrorResponse{Error: "error"})
				} else {
					_ = json.NewEncoder(w).Encode(tt.args.result)
				}
			}))

			defer server.Close()

			c := &Client{
				restHost:   server.URL,
				httpClient: http.DefaultClient,
			}

			var res any
			if err := c.Post(context.Background(), tt.args.path, tt.args.req, &res); (err != nil) != tt.wantErr {
				t.Errorf("Client.Post() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(res, tt.args.result) {
				t.Errorf("Client.Post() = %v, want %v", res, tt.args.result)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	type args struct {
		rest string
	}
	tests := []struct {
		name    string
		args    args
		want    *Client
		wantErr bool
	}{
		{
			name: "test-1",
			args: args{"http://test.com/hello"},
			want: &Client{
				restHost:   "http://test.com",
				httpClient: &http.Client{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewClient(tt.args.rest)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClient() = %v, want %v", got, tt.want)
			}
		})
	}
}
