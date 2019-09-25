package directory

import "encoding/json"

type TfgridFarmer1 struct {
	OwnerThreebotId string   `json:"owner_threebot_id"`
	IyoOrganization string   `json:"iyo_organization"`
	Name            string   `json:"name"`
	WalletAddresses []string `json:"wallet_addresses"`
	Email           string   `json:"email"`
}

func NewTfgridFarmer1() (TfgridFarmer1, error) {
	const value = "{}"
	var object TfgridFarmer1
	if err := json.Unmarshal([]byte(value), &object); err != nil {
		return object, err
	}
	return object, nil
}
