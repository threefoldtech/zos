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

	l := Location{
		Longitute: 0.0,
		Latitude:  0.0,
		Continent: "Unknown",
		Country:   "Unknown",
		City:      "Unknown",
	}

	for i := 0; i < len(geoipURLs); i++ {
		resp, err := http.Get(geoipURLs[i])
		if err != nil {
			log.Err(err).Msgf("failed to make http call to geoip service %s. retrying...", geoipURLs[i])
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Err(err).Msgf("geoip service %s responded with status %d. retrying...", geoipURLs[i], resp.StatusCode)
			continue
		}

		if err := json.NewDecoder(resp.Body).Decode(&l); err != nil {
			log.Err(err).Msgf("failed to decode location data from %s", geoipURLs[i])
			continue
		}

		return l, nil
	}

	return l, errors.New("failed to fetch location information")
}
