package mls

// SetTestURLs overrides the API URLs on a client for testing.
// This should only be used in tests.
func SetTestURLs(c *Client, suggestURL, hulkURL, rapidAPIURL string) {
	if suggestURL != "" {
		c.suggestURL = suggestURL
	}
	if hulkURL != "" {
		c.hulkURL = hulkURL
	}
	if rapidAPIURL != "" {
		c.rapidAPIURL = rapidAPIURL
	}
}
