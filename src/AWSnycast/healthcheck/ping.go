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
	contextLogger := log.WithFields(log.Fields{
		"destination": h.Destination,
	})
	contextLogger.Debug("Pinging")
	if err := exec.Command(pingCmd, args...).Run(); err != nil {
		contextLogger.WithFields(log.Fields{"err": err.Error()}).Debug("ping healthcheck failed")
		return false
	}
	contextLogger.Debug("Ping OK")
	return true
}

func PingConstructor(h Healthcheck) (HealthChecker, error) {
	return PingHealthCheck{Destination: h.Destination}, nil
}
