package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/threefoldtech/zos/pkg/gedis/types/directory"
)

type farmStore struct {
	Farms []*directory.TfgridFarm1 `json:"farms"`
}

func LoadfarmStore() (farmStore, error) {
	store := farmStore{
		Farms: []*directory.TfgridFarm1{},
	}
	f, err := os.OpenFile("farms.json", os.O_RDONLY, 0660)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return store, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&store); err != nil {
		return store, err
	}
	return store, nil
}

func (s *farmStore) Save() error {
	f, err := os.OpenFile("farms.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(s); err != nil {
		return err
	}
	return nil
}

func (s *farmStore) List() []*directory.TfgridFarm1 {
	return s.Farms
}

func (s *farmStore) Get(name string) (*directory.TfgridFarm1, error) {
	for _, f := range s.Farms {
		if f.Name == name {
			return f, nil
		}
	}
	return nil, fmt.Errorf("farm %s not found", name)
}

func (s *farmStore) Add(farm directory.TfgridFarm1) error {
	for _, f := range s.Farms {
		if f.Name == farm.Name {
			f.WalletAddresses = farm.WalletAddresses
			f.Location = farm.Location
			f.Email = farm.Email
			f.ResourcePrices = farm.ResourcePrices
			return nil
		}
	}

	s.Farms = append(s.Farms, &farm)
	return nil
}
