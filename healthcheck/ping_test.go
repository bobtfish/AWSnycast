package healthcheck

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHealthcheckPing(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	h.Validate("foo", false)
	err := h.Validate("foo", false)
	assert.Nil(t, err)
	h.Setup()
	assert.Equal(t, h.healthchecker.Healthcheck(), true)
}

func TestHealthcheckPingFail(t *testing.T) {
	pingCmd = "false"
	h := Healthcheck{
		Type:        "ping",
		Destination: "169.254.255.45", // Hopefully you can't talk to this :)
	}
	h.Validate("foo", false)
	err := h.Validate("foo", false)
	assert.Nil(t, err)
	h.Setup()
	assert.Equal(t, h.healthchecker.Healthcheck(), false)
	pingCmd = "ping"
}
