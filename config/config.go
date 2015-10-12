package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
}

func New(filename string) (*Config, error) {
	c := new(Config)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return c, err
	}
	err = yaml.Unmarshal(data, &c)
	return c, err
}
