package healthcheck

import (
	"testing"
)

func TestHealthcheckDefault(t *testing.T) {
	h := Healthcheck{
		Type: "ping",
	}
	h.Default()
	if h.Rise != 2 {
		t.Fail()
	}
	if h.Fall != 3 {
		t.Fail()
	}
}

func TestHealthcheckValidate(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	h.Default()
	err := h.Validate("foo")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestHealthcheckValidateFailNoDestination(t *testing.T) {
	h := Healthcheck{
		Type: "notping",
	}
	err := h.Validate("foo")
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Healthcheck foo has no destination set" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestHealthcheckValidateFailDestination(t *testing.T) {
	h := Healthcheck{
		Type:        "notping",
		Destination: "www.google.com",
	}
	err := h.Validate("foo")
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Healthcheck foo destination 'www.google.com' does not parse as an IP address" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestHealthcheckValidateFailType(t *testing.T) {
	h := Healthcheck{
		Type:        "notping",
		Destination: "127.0.0.1",
	}
	err := h.Validate("foo")
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Unknown healthcheck type 'notping' in foo" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestHealthcheckValidateFailRise(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Fall:        1,
		Destination: "127.0.0.1",
	}
	err := h.Validate("foo")
	if err == nil {
		t.Fail()
	}
	if err.Error() != "rise must be > 0 in foo" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestHealthcheckValidateFailFall(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Rise:        1,
		Destination: "127.0.0.1",
	}
	err := h.Validate("foo")
	if err == nil {
		t.Fail()
	}
	if err.Error() != "fall must be > 0 in foo" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestHealthcheckGetRunner(t *testing.T) {

}
