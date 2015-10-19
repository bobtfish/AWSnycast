package healthcheck

import (
	"testing"
)

func TestHealthcheckTcpNoPort(t *testing.T) {
	c := make(map[string]string)
	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	h.Default()
	// FIXME h.Validate("foo")
	err := h.Setup()
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "'port' not defined in tcp healthcheck config to 127.0.0.1" {
			t.Log(err.Error())
			t.Fail()
		}
	}
}

/*
func TestHealthcheckTcp(t *testing.T) {
	c := make(map[string]string)
	c["port"] = "80"
	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	h.Default()
	err := h.Validate("foo")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err := h.Setup()
	res := h.healthchecker.Healthcheck()
	if !res {
		t.Fail()
	}
}

func TestHealthcheckTcpFail(t *testing.T) {
	h := Healthcheck{
		Type:        "tcp",
		Destination: "169.254.255.45", // Hopefully you can't talk to this :)
	}
	h.Default()
	err := h.Validate("foo")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	h.Setup()
	res := h.healthchecker.Healthcheck()
	if res {
		t.Fail()
	}
}
*/
