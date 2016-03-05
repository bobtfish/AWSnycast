package healthcheck

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHealthcheckCommand(t *testing.T) {
	c := make(map[string]interface{})
	c["command"] = "/bin/true"
	h := Healthcheck{
		Type:        "command",
		Destination: "127.0.0.1",
		Config: c,
	}
	err := h.Validate("foo", false)
	if assert.Nil(t, err) {
		h.Setup()
		assert.Equal(t, h.healthchecker.Healthcheck(), true)
	}
}

func TestHealthcheckCommandFail(t *testing.T) {
	c := make(map[string]interface{})
        c["command"] = "/bin/false"
	h := Healthcheck{
		Type:        "command",
		Destination: "169.254.255.45", // Hopefully you can't talk to this :)
		Config: c,
	}
	err := h.Validate("foo", false)
	if assert.Nil(t, err) {
		h.Setup()
		assert.Equal(t, h.healthchecker.Healthcheck(), false)
	}
}
