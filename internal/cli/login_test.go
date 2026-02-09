package cli

import "testing"

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid key", "hf_abc123def456", false},
		{"empty key", "", true},
		{"missing prefix", "abc123def456", true},
		{"wrong prefix", "xx_abc123", true},
		{"just prefix", "hf_", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPIKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAPIKey(%q) err = %v, wantErr = %v", tt.key, err, tt.wantErr)
			}
		})
	}
}
