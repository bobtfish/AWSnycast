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
	h, ok := c.Healthchecks["public"]
	if !ok {
		t.Fail()
	}
	if h.Type != "ping" {
		t.Fail()
	}
	if h.Destination != "8.8.8.8" {
		t.Fail()
	}
	if h.Rise != 2 {
		t.Fail()
	}
	if h.Fall != 10 {
		t.Fail()
	}
	if h.Every != 1 {
		t.Fail()
	}
	a, ok := c.RouteTables["a"]
	if !ok {
		t.Log("RouteTables a not ok")
		t.Fail()
	}
	if a.Find.Type != "by_tag" {
		t.Fail()
	}
	if v, ok := a.Find.Config["key"]; ok {
		if v != "Name" {
			t.Fail()
		}
	} else {
		t.Fail()
	}
	if v, ok := a.Find.Config["value"]; ok {
		if v != "private a" {
			t.Fail()
		}
	} else {
		t.Fail()
	}
	routes := a.UpsertRoutes
	if len(routes) != 2 {
		t.Fail()
	}
	for _, route := range routes {
		if route.Cidr == "0.0.0.0/0" || route.Cidr == "192.168.1.1/32" {
			if route.Instance != "SELF" {
				t.Fail()
			}
			if route.Cidr == "0.0.0.0/0" {
				if route.Healthcheck != "public" {
					t.Fail()
				}
			} else {
				if route.Healthcheck != "localservice" {
					t.Fail()
				}
			}
		} else {
			t.Fail()
		}
	}
	b, ok := c.RouteTables["b"]
	if !ok {
		t.Fail()
	}
	if b.Find.Type != "by_tag" {
		t.Fail()
	}
}
