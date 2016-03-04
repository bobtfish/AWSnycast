package healthcheck

import (
	log "github.com/Sirupsen/logrus"
	"os/exec"
	"strings"
)

func init() {
	RegisterHealthcheck("command", CommandConstructor)
}

type CommandHealthCheck struct {
	Destination string
	Command     string
	Arguments   []string
}

func (h CommandHealthCheck) Healthcheck() bool {
	contextLogger := log.WithFields(log.Fields{
		"destination": h.Destination,
		"command":     h.Command,
		"arguments":   strings.Join(h.Arguments, ", "),
	})
	contextLogger.Debug("Run command")
	if err := exec.Command(h.Command, h.Arguments...).Run(); err != nil {
		contextLogger.WithFields(log.Fields{"err": err.Error()}).Debug("command healthcheck failed")
		return false
	}
	contextLogger.Debug("command OK")
	return true
}

func CommandConstructor(h Healthcheck) (HealthChecker, error) {
	command := "ping"
	args := []string{"-c", "1", h.Destination}
	return CommandHealthCheck{
		Destination: h.Destination,
		Command:     command,
		Arguments:   args,
	}, nil
}
