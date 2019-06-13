package zinit

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type InitService struct {
	Exec    string   `yaml:"exec"`
	Oneshot bool     `yaml:"oneshot"`
	After   []string `yaml:"after"`
}

func AddService(name string, service InitService) error {
	b, err := yaml.Marshal(service)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/etc/zinit/%s.yaml", name)
	return ioutil.WriteFile(path, b, 0660)
}
