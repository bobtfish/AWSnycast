package healthcheck

import (
	"errors"
	log "github.com/sirupsen/logrus"
	utils "github.com/bobtfish/AWSnycast/utils"
	"github.com/hashicorp/go-multierror"
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
	var result *multierror.Error
	hc := CommandHealthCheck{
		Destination: h.Destination,
	}
	if val, ok := h.Config["command"]; ok {
		hc.Command = utils.GetAsString(val)
	} else {
		result = multierror.Append(result, errors.New("'command' not defined in command healthcheck config to "+h.Destination))
	}
	if val, ok := h.Config["arguments"]; ok {
		args, err := utils.GetAsSlice(val)
		if err != nil {
			result = multierror.Append(result, err)
		} else {
			for i, val := range args {
				args[i] = strings.Replace(val, "%DESTINATION%", h.Destination, -1)
			}
			hc.Arguments = args
		}
	}
	return hc, result.ErrorOrNil()
}
