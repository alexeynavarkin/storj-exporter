package storj

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Config struct {
	BaseURL string
}

type Client struct {
	httpClient *http.Client
	cfg        Config
}

func NewClient(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		cfg:        cfg,
	}
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w, %s", err, body)
	}
	return nil
}

func (c *Client) GetSno(ctx context.Context) (*SNOResponse, error) {
	var res SNOResponse
	if err := c.get(ctx, "/api/sno/", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) GetSnoSattilite(ctx context.Context, satID string) (*SNOSatteliteResponse, error) {
	var res SNOSatteliteResponse
	if err := c.get(ctx, "/api/sno/satellite/"+satID, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) GetSnoPayout(ctx context.Context) (*SNOPayoutResponse, error) {
	var res SNOPayoutResponse
	if err := c.get(ctx, "/api/sno/estimated-payout", &res); err != nil {
		return nil, err
	}

	// API reports payouts in cents.
	for _, m := range []*payoutMonth{&res.CurrentMonth, &res.PreviousMonth} {
		m.Payout /= 100
		m.DiskSpacePayout /= 100
		m.EgressBandwidthPayout /= 100
		m.EgressRepairAuditPayout /= 100
		m.Held /= 100
	}
	return &res, nil
}
