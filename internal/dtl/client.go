package dtl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Client struct {
	restHost   string
	httpClient *http.Client
}

func NewClient(rest string) (*Client, error) {
	parsed, err := url.Parse(rest)
	if err != nil {
		return nil, fmt.Errorf("invalid rest server base url: %s", rest)
	}
	if parsed.Path != "/" {
		parsed.Path = ""
	}
	return &Client{
		restHost:   parsed.String(),
		httpClient: &http.Client{},
	}, nil
}

// ErrorResponse defines the attributes of a JSON error response
type ErrorResponse struct {
	Error string `json:"error"`
}

func (c *Client) Get(ctx context.Context, path string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.restHost+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	var data ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	return fmt.Errorf("rest client error: path %s msg %s", path, data.Error)
}
