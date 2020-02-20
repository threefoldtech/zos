package directory

import (
	"encoding/json"
	schema "github.com/threefoldtech/zos/pkg/schema"
)

type TfgridDirectoryFarm1 struct {
	ThreebotId      int64                               `bson:"threebot_id" json:"threebot_id"`
	IyoOrganization string                              `bson:"iyo_organization" json:"iyo_organization"`
	Name            string                              `bson:"name" json:"name"`
	WalletAddresses []string                            `bson:"wallet_addresses" json:"wallet_addresses"`
	Location        TfgridDirectoryLocation1            `bson:"location" json:"location"`
	Email           string                              `bson:"email" json:"email"`
	ResourcePrices  []TfgridDirectoryNodeResourcePrice1 `bson:"resource_prices" json:"resource_prices"`
	PrefixZero      schema.IPRange                      `bson:"prefix_zero" json:"prefix_zero"`
}

func NewTfgridDirectoryFarm1() (TfgridDirectoryFarm1, error) {
	const value = "{}"
	var object TfgridDirectoryFarm1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodeResourcePrice1 struct {
	Currency TfgridDirectoryNodeResourcePrice1CurrencyEnum `bson:"currency" json:"currency"`
	Cru      float64                                       `bson:"cru" json:"cru"`
	Mru      float64                                       `bson:"mru" json:"mru"`
	Hru      float64                                       `bson:"hru" json:"hru"`
	Sru      float64                                       `bson:"sru" json:"sru"`
	Nru      float64                                       `bson:"nru" json:"nru"`
}

func NewTfgridDirectoryNodeResourcePrice1() (TfgridDirectoryNodeResourcePrice1, error) {
	const value = "{}"
	var object TfgridDirectoryNodeResourcePrice1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridDirectoryNodeResourcePrice1CurrencyEnum uint8

const (
	TfgridDirectoryNodeResourcePrice1CurrencyEUR TfgridDirectoryNodeResourcePrice1CurrencyEnum = iota
	TfgridDirectoryNodeResourcePrice1CurrencyUSD
	TfgridDirectoryNodeResourcePrice1CurrencyTFT
	TfgridDirectoryNodeResourcePrice1CurrencyAED
	TfgridDirectoryNodeResourcePrice1CurrencyGBP
)

func (e TfgridDirectoryNodeResourcePrice1CurrencyEnum) String() string {
	switch e {
	case TfgridDirectoryNodeResourcePrice1CurrencyEUR:
		return "EUR"
	case TfgridDirectoryNodeResourcePrice1CurrencyUSD:
		return "USD"
	case TfgridDirectoryNodeResourcePrice1CurrencyTFT:
		return "TFT"
	case TfgridDirectoryNodeResourcePrice1CurrencyAED:
		return "AED"
	case TfgridDirectoryNodeResourcePrice1CurrencyGBP:
		return "GBP"
	}
	return "UNKNOWN"
}
