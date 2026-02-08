package property

import (
	"fmt"

	"github.com/evcraddock/house-finder/internal/mls"
)

// Service provides property business logic.
type Service struct {
	repo   *Repository
	client *mls.Client
}

// NewService creates a property service.
func NewService(repo *Repository, client *mls.Client) *Service {
	return &Service{repo: repo, client: client}
}

// Add looks up a property by address, fetches its data, and stores it.
// This is the only operation that hits external APIs.
func (s *Service) Add(address string) (*Property, error) {
	result, err := s.client.Lookup(address)
	if err != nil {
		return nil, fmt.Errorf("looking up property: %w", err)
	}

	fields := parseRawJSON(result.RawJSON)

	p := &Property{
		Address:      address,
		MprID:        result.MprID,
		RealtorURL:   result.RealtorURL,
		Price:        fields.Price,
		Bedrooms:     fields.Bedrooms,
		Bathrooms:    fields.Bathrooms,
		Sqft:         fields.Sqft,
		LotSize:      fields.LotSize,
		YearBuilt:    fields.YearBuilt,
		PropertyType: fields.PropertyType,
		Status:       fields.Status,
		RawJSON:      result.RawJSON,
	}

	saved, err := s.repo.Insert(p)
	if err != nil {
		return nil, fmt.Errorf("saving property: %w", err)
	}

	return saved, nil
}
