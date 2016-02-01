package healthcheck

/*
import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHealthcheckCommand(t *testing.T) {
	h := Healthcheck{
		Type:        "command",
		Destination: "127.0.0.1",
	}
	err := h.Validate("foo", false)
	assert.Nil(t, err)
	h.Setup()
	assert.Equal(t, h.healthchecker.Healthcheck(), true)
}
func TestHealthcheckCommandFail(t *testing.T) {
	h := Healthcheck{
		Type:        "command",
		Destination: "169.254.255.45", // Hopefully you can't talk to this :)
	}
	err := h.Validate("foo", false)
	assert.Nil(t, err)
	h.Setup()
	assert.Equal(t, h.healthchecker.Healthcheck(), false)
}
*/
