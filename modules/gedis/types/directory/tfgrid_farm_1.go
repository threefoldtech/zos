package directory

import (
	"encoding/json"
	schema "github.com/threefoldtech/zosv2/modules/schema"
)

type TfgridFarm1 struct {
	ThreebotId      string                     `json:"threebot_id"`
	IyoOrganization string                     `json:"iyo_organization"`
	Name            string                     `json:"name"`
	WalletAddresses []string                   `json:"wallet_addresses"`
	Location        TfgridLocation1            `json:"location"`
	Email           string                     `json:"email"`
	ResourcePrices  []TfgridNodeResourcePrice1 `json:"resource_prices"`
	PrefixZero      schema.IPRange             `json:"prefix_zero"`
}

func NewTfgridFarm1() (TfgridFarm1, error) {
	const value = "{}"
	var object TfgridFarm1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridNodeResourcePrice1 struct {
	Currency TfgridNodeResourcePrice1CurrencyEnum `json:"currency"`
	Cru      float64                              `json:"cru"`
	Mru      float64                              `json:"mru"`
	Hru      float64                              `json:"hru"`
	Sru      float64                              `json:"sru"`
	Nru      float64                              `json:"nru"`
}

func NewTfgridNodeResourcePrice1() (TfgridNodeResourcePrice1, error) {
	const value = "{}"
	var object TfgridNodeResourcePrice1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}

type TfgridNodeResourcePrice1CurrencyEnum uint8

const (
	TfgridNodeResourcePrice1CurrencyEUR TfgridNodeResourcePrice1CurrencyEnum = iota
	TfgridNodeResourcePrice1CurrencyUSD
	TfgridNodeResourcePrice1CurrencyTFT
	TfgridNodeResourcePrice1CurrencyAED
	TfgridNodeResourcePrice1CurrencyGBP
)

func (e TfgridNodeResourcePrice1CurrencyEnum) String() string {
	switch e {
	case TfgridNodeResourcePrice1CurrencyEUR:
		return "EUR"
	case TfgridNodeResourcePrice1CurrencyUSD:
		return "USD"
	case TfgridNodeResourcePrice1CurrencyTFT:
		return "TFT"
	case TfgridNodeResourcePrice1CurrencyAED:
		return "AED"
	case TfgridNodeResourcePrice1CurrencyGBP:
		return "GBP"
	}
	return "UNKNOWN"
}
