// Package mls fetches property data from realtor.com and RapidAPI.
package mls

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	defaultSuggestURL  = "https://parser-external.geo.moveaws.com/suggest"
	defaultHulkURL     = "https://www.realtor.com/api/v1/hulk_main_srp?client_id=rdc-x&schema=vesta"
	defaultRapidAPIURL = "https://us-real-estate-listings.p.rapidapi.com/v2/property"
	userAgent          = "Mozilla/5.0"
)

// Result holds the data returned from a property lookup.
type Result struct {
	MprID      string          `json:"mpr_id"`
	RealtorURL string          `json:"realtor_url"`
	RawJSON    json.RawMessage `json:"raw_json"`
}

// Client fetches property data from external APIs.
type Client struct {
	httpClient  *http.Client
	rapidAPIKey string

	// Overridable URLs for testing.
	suggestURL  string
	hulkURL     string
	rapidAPIURL string
}

// NewClient creates an MLS client with the given RapidAPI key.
func NewClient(rapidAPIKey string) (*Client, error) {
	if rapidAPIKey == "" {
		return nil, fmt.Errorf("RAPIDAPI_KEY is required")
	}
	return &Client{
		httpClient:  &http.Client{},
		rapidAPIKey: rapidAPIKey,
		suggestURL:  defaultSuggestURL,
		hulkURL:     defaultHulkURL,
		rapidAPIURL: defaultRapidAPIURL,
	}, nil
}

// Lookup fetches property data for the given address.
// This makes API calls: 2 free (realtor.com) + 1 RapidAPI call.
func (c *Client) Lookup(address string) (*Result, error) {
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}

	mprID, err := c.lookupMprID(address)
	if err != nil {
		return nil, fmt.Errorf("geocoder lookup: %w", err)
	}

	href, err := c.lookupRealtorURL(mprID)
	if err != nil {
		return nil, fmt.Errorf("realtor URL lookup: %w", err)
	}

	rawJSON, err := c.fetchPropertyDetail(href)
	if err != nil {
		return nil, fmt.Errorf("property detail fetch: %w", err)
	}

	return &Result{
		MprID:      mprID,
		RealtorURL: href,
		RawJSON:    rawJSON,
	}, nil
}

// suggestResponse is the response from the realtor.com suggest API.
type suggestResponse struct {
	Autocomplete []struct {
		MprID string `json:"mpr_id"`
	} `json:"autocomplete"`
}

// lookupMprID resolves an address to a realtor.com property ID.
func (c *Client) lookupMprID(address string) (string, error) {
	params := url.Values{
		"input":     {address},
		"client_id": {"rdc-home"},
		"limit":     {"1"},
	}

	req, err := http.NewRequest("GET", c.suggestURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = fmt.Errorf("%w (also failed to close body: %v)", err, closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result suggestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	if len(result.Autocomplete) == 0 || result.Autocomplete[0].MprID == "" {
		return "", fmt.Errorf("no property found for address: %s", address)
	}

	return result.Autocomplete[0].MprID, nil
}

// hulkRequest is the GraphQL request body for the hulk API.
type hulkRequest struct {
	Query string `json:"query"`
}

// hulkResponse is the response from the hulk API.
type hulkResponse struct {
	Data struct {
		Home struct {
			Href       string `json:"href"`
			PropertyID string `json:"property_id"`
		} `json:"home"`
	} `json:"data"`
}

// lookupRealtorURL resolves a property ID to a realtor.com URL.
func (c *Client) lookupRealtorURL(mprID string) (string, error) {
	query := fmt.Sprintf(`query { home(property_id: "%s") { href property_id } }`, mprID)
	body, err := json.Marshal(hulkRequest{Query: query})
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", c.hulkURL, strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = fmt.Errorf("%w (also failed to close body: %v)", err, closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result hulkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	if result.Data.Home.Href == "" {
		return "", fmt.Errorf("no realtor.com URL found for property ID: %s", mprID)
	}

	return result.Data.Home.Href, nil
}

// fetchPropertyDetail fetches full property details from RapidAPI.
func (c *Client) fetchPropertyDetail(realtorURL string) (json.RawMessage, error) {
	params := url.Values{
		"property_url": {realtorURL},
	}

	req, err := http.NewRequest("GET", c.rapidAPIURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("x-rapidapi-host", "us-real-estate-listings.p.rapidapi.com")
	req.Header.Set("x-rapidapi-key", c.rapidAPIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = fmt.Errorf("%w (also failed to close body: %v)", err, closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if !json.Valid(raw) {
		return nil, fmt.Errorf("response is not valid JSON")
	}

	return json.RawMessage(raw), nil
}
