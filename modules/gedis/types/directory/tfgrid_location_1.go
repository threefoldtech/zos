package directory

import "encoding/json"

type TfgridLocation1 struct {
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Continent string  `json:"continent"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func NewTfgridLocation1() (TfgridLocation1, error) {
	const value = "{}"
	var object TfgridLocation1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}
