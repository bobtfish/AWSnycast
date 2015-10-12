package config

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	c, err := New("../tests/awsnycast.yaml")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if c == nil {
		t.Fail()
	}
}
