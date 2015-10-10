package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	c := New("config_file_example.yaml")
	if c == nil {
		t.Fail()
	}
}
