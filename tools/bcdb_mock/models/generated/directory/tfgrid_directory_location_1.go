package directory

import "encoding/json"

type TfgridDirectoryLocation1 struct {
	City      string  `bson:"city" json:"city"`
	Country   string  `bson:"country" json:"country"`
	Continent string  `bson:"continent" json:"continent"`
	Latitude  float64 `bson:"latitude" json:"latitude"`
	Longitude float64 `bson:"longitude" json:"longitude"`
}

func NewTfgridDirectoryLocation1() (TfgridDirectoryLocation1, error) {
	const value = "{}"
	var object TfgridDirectoryLocation1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}
