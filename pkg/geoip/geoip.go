package geoip

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
)

// Location holds the result of a geoip request
type Location struct {
	Longitute   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
	Continent   string  `json:"continent"`
	Country     string  `json:"country_name"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city_name"`
}

// Fetch retrieves the location of the system calling this function
func Fetch() (Location, error) {
	geoipURLs := []string{"https://geoip.grid.tf/", "https://02.geoip.grid.tf/", "https://03.geoip.grid.tf/"}

	for _, url := range geoipURLs {
		l, err := getLocation(url)
		if err != nil {
			log.Err(err).Msg("failed to fetch location. retrying...")
			continue
		}

		return l, nil
	}

	return Location{}, errors.New("failed to fetch location information")
}

func getLocation(url string) (Location, error) {
	resp, err := http.Get(url)
	if err != nil {
		return Location{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return Location{}, errors.New("error fetching location")
	}

	l := Location{}
	if err := json.NewDecoder(resp.Body).Decode(&l); err != nil {
		return Location{}, err
	}

	return l, nil
}
