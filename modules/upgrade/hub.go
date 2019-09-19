package upgrade

import (
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

const (
	// HubBaseURL base hub url
	HubBaseURL = "https://hub.grid.tf/"
	// HubStorage default hub db
	HubStorage = "zdb://hub.grid.tf:9900"
)

// Hub API for f-list
type Hub struct{}

// URL returns the full url of given flist.
func (h *Hub) URL(flist string) string {
	url, err := url.Parse(HubBaseURL)
	if err != nil {
		panic("invalid base url")
	}
	url.Path = flist
	return url.String()
}

// Storage return hub storage url
func (h *Hub) Storage() string {
	return HubStorage
}

// Hash gets flist has from hub. flist is formatted as 'org/flist-name.flist'
func (h *Hub) Hash(flist string) (string, error) {
	response, err := http.Get(h.URL(flist) + ".md5")
	if err != nil {
		return "", errors.Wrap(err, "failed to get hash")
	}

	defer response.Body.Close()
	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read hub response")
	}

	if response.StatusCode != http.StatusOK {
		return "", errors.Errorf("invalid response for flist hash: %s", response.Status)
	}

	return string(content), nil
}
