package healthcheck

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net"
	"strings"
	"time"
)

func init() {
	RegisterHealthcheck("tcp", TcpConstructor)
}

type TcpHealthCheck struct {
	Destination string
	Port        string
	Send        string
	Expect      string
}

func (h TcpHealthCheck) Healthcheck() bool {
	contextLogger := log.WithFields(log.Fields{
		"destination": h.Destination,
		"port":        h.Port,
	})
	contextLogger.Info("Probing TCP port")
	c, err := net.Dial("tcp", h.Destination+":"+h.Port)
	if err != nil {
		contextLogger.WithFields(log.Fields{"err": err.Error()}).Info("Failed connecting")
		return false
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(time.Second * 10))

	if h.Send != "" {
		fmt.Fprintf(c, h.Send)
	}

	if h.Expect == "" {
		return true
	}

	b := make([]byte, 1024)
	n, err := c.Read(b)
	if err != nil {
		contextLogger.WithFields(log.Fields{"err": err.Error()}).Debug("Could not read response")
		return false
	}
	answer := string(b[:n])
	ansLogger := contextLogger.WithFields(log.Fields{
		"answer": answer,
		"expect": h.Expect,
	})
	if strings.Contains(answer, h.Expect) {
		ansLogger.Debug("Healthy response")
		return true
	}
	ansLogger.Debug("Unhealthy response")
	return false
}

func TcpConstructor(h Healthcheck) (HealthChecker, error) {
	if _, ok := h.Config["port"]; !ok {
		return TcpHealthCheck{}, errors.New("'port' not defined in tcp healthcheck config to " + h.Destination)
	}
	hc := TcpHealthCheck{
		Destination: h.Destination,
		Port:        h.Config["port"],
	}
	if v, ok := h.Config["expect"]; ok {
		hc.Expect = v
	}
	if v, ok := h.Config["send"]; ok {
		hc.Send = v
	}
	return hc, nil
}
