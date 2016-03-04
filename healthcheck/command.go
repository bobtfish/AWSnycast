package healthcheck

import (
	log "github.com/Sirupsen/logrus"
	"os/exec"
)

func init() {
	RegisterHealthcheck("command", CommandConstructor)
}

type CommandHealthCheck struct {
	Destination string
	Command string
}

func (h CommandHealthCheck) Healthcheck() bool {
	args := []string{"-c", "1", h.Destination}
	contextLogger := log.WithFields(log.Fields{
		"destination": h.Destination,
		"command": h.Command,
	})
	contextLogger.Debug("Run command")
	if err := exec.Command(h.Command, args...).Run(); err != nil {
		contextLogger.WithFields(log.Fields{"err": err.Error()}).Debug("command healthcheck failed")
		return false
	}
	contextLogger.Debug("command OK")
	return true
}

func CommandConstructor(h Healthcheck) (HealthChecker, error) {
	command := "ping"
	return CommandHealthCheck{
		Destination: h.Destination,
		Command: command,
	}, nil
}
