package storj

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		httpClient: &http.Client{},
		cfg:        cfg,
	}
}

func (c *Client) GetSno(ctx context.Context) (*SNOResponse, error) {
	resp, err := c.httpClient.Get(c.cfg.BaseURL + "/api/sno/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	var res SNOResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, %s", err, body)
	}

	return &res, nil
}

func (c *Client) GetSnoSattilite(ctx context.Context, satID string) (*SNOSatteliteResponse, error) {
	resp, err := c.httpClient.Get(
		c.cfg.BaseURL + "/api/sno/satellite/" + satID,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	var res SNOSatteliteResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, %s", err, body)
	}

	return &res, nil
}

func (c *Client) GetSnoPayout(ctx context.Context) (*SNOPayoutResponse, error) {
	resp, err := c.httpClient.Get(
		c.cfg.BaseURL + "/api/sno/estimated-payout",
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, body)
	}

	var res SNOPayoutResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, %s", err, body)
	}

	// Convert cents to dollars.
	res.CurrentMonth.Payout /= 100
	res.CurrentMonth.DiskSpacePayout /= 100
	res.CurrentMonth.EgressBandwidthPayout /= 100
	res.CurrentMonth.EgressRepairAuditPayout /= 100
	res.CurrentMonth.Held /= 100
	res.CurrentMonthExpectations /= 100

	return &res, nil
}
