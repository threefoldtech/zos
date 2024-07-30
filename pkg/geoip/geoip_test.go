package geoip

import (
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func Test_getLocation(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	t.Run("running correct response", func(t *testing.T) {
		for i := 0; i < len(geoipURLs); i++ {
			httpmock.RegisterResponder("GET", geoipURLs[i],
				func(req *http.Request) (*http.Response, error) {
					l := Location{
						Continent: "Africa",
						Country:   "Egypt",
						City:      "Cairo",
					}
					resp, err := httpmock.NewJsonResponse(200, l)
					return resp, err
				},
			)
			value, err := getLocation(geoipURLs[i])
			assert.Equal(t, nil, err)
			assert.Equal(t, "Egypt", value.Country)
			assert.Equal(t, "Africa", value.Continent)
			assert.Equal(t, "Cairo", value.City)
			if err != nil {
				t.Errorf("got %v", err)
			}
		}
	})

	t.Run("asserting wrong response code", func(t *testing.T) {
		for i := 0; i < len(geoipURLs); i++ {
			httpmock.RegisterResponder("GET", geoipURLs[i],
				func(req *http.Request) (*http.Response, error) {
					l := Location{
						Continent: "Africa",
						Country:   "Egypt",
						City:      "Cairo",
					}
					resp, err := httpmock.NewJsonResponse(404, l)
					return resp, err
				},
			)
			value, err := getLocation(geoipURLs[i])
			assert.NotEqual(t, err, nil)
			assert.Equal(t, "Unknown", value.Country)
			assert.Equal(t, "Unknown", value.Continent)
			assert.Equal(t, "Unknown", value.City)
		}
	})

	t.Run("asserting sending wrong response data", func(t *testing.T) {
		for i := 0; i < len(geoipURLs); i++ {
			httpmock.RegisterResponder("GET", geoipURLs[i],
				func(req *http.Request) (*http.Response, error) {
					l := Location{
						City: "Cairo",
					}
					resp, err := httpmock.NewJsonResponse(200, l.City)
					return resp, err
				},
			)
			value, err := getLocation(geoipURLs[i])
			assert.NotEqual(t, err, nil)
			assert.Equal(t, "Unknown", value.Country)
			assert.Equal(t, "Unknown", value.Continent)
			assert.Equal(t, "Unknown", value.City)
		}
	})

}
