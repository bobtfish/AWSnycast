package healthcheck

import (
	"github.com/bobtfish/AWSnycast/instancemetadata"
	"testing"
)

func TestHealthcheckPing(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	h.Default(instancemetadata.InstanceMetadata{})
	err := h.Validate("foo")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	h.Setup()
	res := h.healthchecker.Healthcheck()
	if !res {
		t.Fail()
	}
}

func TestHealthcheckPingFail(t *testing.T) {
	pingCmd = "false"
	h := Healthcheck{
		Type:        "ping",
		Destination: "169.254.255.45", // Hopefully you can't talk to this :)
	}
	h.Default(instancemetadata.InstanceMetadata{})
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
	pingCmd = "ping"
}
