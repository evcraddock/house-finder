// Package client provides an HTTP client for the house-finder REST API.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/evcraddock/house-finder/internal/comment"
	"github.com/evcraddock/house-finder/internal/property"
	"github.com/evcraddock/house-finder/internal/visit"
)

// Client is an HTTP client for the house-finder API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New creates a new API client.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ShowResponse is the response from GET /api/properties/{id}.
type ShowResponse struct {
	Property *property.Property `json:"property"`
	Comments []*comment.Comment `json:"comments"`
	Visits   []*visit.Visit     `json:"visits"`
}

// ListOptions controls filtering for ListProperties.
type ListOptions struct {
	MinRating   int
	VisitStatus string // not_visited, want_to_visit, visited (empty = all)
}

// ListProperties returns all properties, optionally filtered.
func (c *Client) ListProperties(opts ListOptions) ([]*property.Property, error) {
	path := "/api/properties"
	var params []string
	if opts.MinRating > 0 {
		params = append(params, fmt.Sprintf("min_rating=%d", opts.MinRating))
	}
	if opts.VisitStatus != "" {
		params = append(params, fmt.Sprintf("visit_status=%s", opts.VisitStatus))
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	var props []*property.Property
	if err := c.get(path, &props); err != nil {
		return nil, err
	}
	return props, nil
}

// GetProperty returns a property with its comments.
func (c *Client) GetProperty(id int64) (*ShowResponse, error) {
	var resp ShowResponse
	if err := c.get(fmt.Sprintf("/api/properties/%d", id), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AddProperty adds a property by address (server does MLS lookup).
func (c *Client) AddProperty(address string) (*property.Property, error) {
	body := map[string]string{"address": address}
	var p property.Property
	if err := c.post("/api/properties", body, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// DeleteProperty removes a property.
func (c *Client) DeleteProperty(id int64) error {
	return c.doDelete(fmt.Sprintf("/api/properties/%d", id))
}

// RateProperty sets a rating on a property.
func (c *Client) RateProperty(id int64, rating int) error {
	body := map[string]int{"rating": rating}
	return c.post(fmt.Sprintf("/api/properties/%d/rate", id), body, nil)
}

// AddComment adds a comment to a property.
func (c *Client) AddComment(id int64, text string) (*comment.Comment, error) {
	body := map[string]string{"text": text}
	var comm comment.Comment
	if err := c.post(fmt.Sprintf("/api/properties/%d/comments", id), body, &comm); err != nil {
		return nil, err
	}
	return &comm, nil
}

// ListComments returns comments for a property.
func (c *Client) ListComments(id int64) ([]*comment.Comment, error) {
	var comments []*comment.Comment
	if err := c.get(fmt.Sprintf("/api/properties/%d/comments", id), &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// AddVisit records a visit to a property.
func (c *Client) AddVisit(id int64, visitDate, visitType, notes string) (*visit.Visit, error) {
	body := map[string]string{
		"visit_date": visitDate,
		"visit_type": visitType,
		"notes":      notes,
	}
	var v visit.Visit
	if err := c.post(fmt.Sprintf("/api/properties/%d/visits", id), body, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVisits returns visits for a property.
func (c *Client) ListVisits(id int64) ([]*visit.Visit, error) {
	var visits []*visit.Visit
	if err := c.get(fmt.Sprintf("/api/properties/%d/visits", id), &visits); err != nil {
		return nil, err
	}
	return visits, nil
}

// get performs a GET request and decodes the response.
func (c *Client) get(path string, result interface{}) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	return c.do(req, result)
}

// post performs a POST request with a JSON body and decodes the response.
func (c *Client) post(path string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.do(req, result)
}

// doDelete performs a DELETE request.
func (c *Client) doDelete(path string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	return c.do(req, nil)
}

// do executes an HTTP request with auth header and handles errors.
func (c *Client) do(req *http.Request, result interface{}) error {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Printf("warning: closing response body: %v\n", cerr)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("%s", errResp.Error)
		}
		return fmt.Errorf("server error: %s", http.StatusText(resp.StatusCode))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

// EmailRequest specifies which properties to email.
type EmailRequest struct {
	PropertyIDs []int64 `json:"property_ids,omitempty"`
	MinRating   *int    `json:"min_rating,omitempty"`
	VisitStatus string  `json:"visit_status,omitempty"`
	DryRun      bool    `json:"dry_run"`
}

// EmailResponse is the response from POST /api/email.
type EmailResponse struct {
	Sent    bool     `json:"sent"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

// SendEmail sends an email to realtors with the selected properties.
func (c *Client) SendEmail(req EmailRequest) (*EmailResponse, error) {
	var resp EmailResponse
	if err := c.post("/api/email", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
