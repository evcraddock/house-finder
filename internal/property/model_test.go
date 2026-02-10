package property

import (
	"encoding/json"
	"testing"
)

func TestExtractPhotoURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "house_view tag preferred",
			raw: `{"data":{"photos":[
				{"href":"https://example.com/garage.jpg","tags":[{"label":"garage"}]},
				{"href":"https://example.com/house.jpg","tags":[{"label":"house_view"}]},
				{"href":"https://example.com/kitchen.jpg","tags":[{"label":"kitchen"}]}
			]}}`,
			want: "https://example.com/house.jpg",
		},
		{
			name: "fallback to first photo",
			raw: `{"data":{"photos":[
				{"href":"https://example.com/first.jpg","tags":[{"label":"garage"}]},
				{"href":"https://example.com/second.jpg","tags":[{"label":"yard"}]}
			]}}`,
			want: "https://example.com/first.jpg",
		},
		{
			name: "no photos",
			raw:  `{"data":{"photo_count":0}}`,
			want: "",
		},
		{
			name: "empty photos array",
			raw:  `{"data":{"photos":[]}}`,
			want: "",
		},
		{
			name: "invalid json",
			raw:  `not json`,
			want: "",
		},
		{
			name: "photos at top level",
			raw: `{"photos":[
				{"href":"https://example.com/top.jpg","tags":[{"label":"house_view"}]}
			]}`,
			want: "https://example.com/top.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPhotoURL(json.RawMessage(tt.raw))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
