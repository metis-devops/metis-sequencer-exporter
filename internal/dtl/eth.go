package dtl

import (
	"context"
	"fmt"
)

type L1HighestSynced struct {
	BlockNumber uint64 `json:"blockNumber"`
}

func (c *Client) GetL1HighestSynced(ctx context.Context) (uint64, error) {
	var res L1HighestSynced
	if err := c.Get(ctx, "/highest/l1", &res); err != nil {
		return 0, fmt.Errorf("DTL: GetL1HighestSynced: %w", err)
	}
	return res.BlockNumber, nil
}
