package healthcheck

import (
	log "github.com/Sirupsen/logrus"
	"os/exec"
)

var pingCmd string

func init() {
	pingCmd = "ping"
	RegisterHealthcheck("ping", PingConstructor)
}

type PingHealthCheck struct {
	Destination string
}

func (h PingHealthCheck) Healthcheck() bool {
	args := []string{"-c", "1", h.Destination}
	if err := exec.Command(pingCmd, args...).Run(); err != nil {
		log.Printf("ping healthcheck to %s failed: %s", h.Destination, err.Error())
		return false
	}
	return true
}

func PingConstructor(h Healthcheck) (HealthChecker, error) {
	return PingHealthCheck{Destination: h.Destination}, nil
}
