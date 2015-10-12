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

func TestLoadConfigFails(t *testing.T) {
	_, err := New("../tests/doesnotexist.yaml")
	if err == nil {
		t.Fail()
	}
}

func TestLoadConfigHealthchecks(t *testing.T) {
	c, _ := New("../tests/awsnycast.yaml")
	if c.Healthchecks == nil {
		t.Fail()
	}
}
