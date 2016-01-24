package healthcheck

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net"
	"strconv"
	"strings"
	"time"

	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
)

func init() {
	RegisterHealthcheck("tcp", TcpConstructor)
}

type TcpHealthCheck struct {
	Destination string
	Port        string
	Send        string
	Expect      string
	TLS         bool
	x509        []byte
	SkipVerify  bool
	ServerName  string
}

func (h TcpHealthCheck) VerifyResponse(answer string, contextLogger *log.Entry) bool {

	ansLogger := contextLogger.WithFields(log.Fields{
		"answer": answer,
		"expect": h.Expect,
	})

	if strings.Contains(answer, h.Expect) {
		ansLogger.Info("Healthy response")
		return true
	}
	ansLogger.Info("Unhealthy response")
	return false
}

func TLSHealthCheck(h TcpHealthCheck) bool {
	contextLogger := log.WithFields(log.Fields{
		"destination": h.Destination,
		"port":        h.Port,
		"tls":         h.TLS,
	})
	contextLogger.Info("Probing TCP port")

	config := &tls.Config{
		InsecureSkipVerify: h.SkipVerify,
	}
	if !h.SkipVerify {
		roots := x509.NewCertPool()
		if len(h.x509) > 0 {
			ok := roots.AppendCertsFromPEM(h.x509)
			if !ok {
				contextLogger.Info("Failed to parse PEM file")
				return false
			}
		}
		config = &tls.Config{
			RootCAs:    roots,
			ServerName: h.ServerName,
		}
	}

	c, err := tls.Dial(
		"tcp",
		h.Destination+":"+h.Port,
		config,
	)

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
	return h.VerifyResponse(answer, contextLogger)
}

func (h TcpHealthCheck) Healthcheck() bool {
	if h.TLS {
		return TLSHealthCheck(h)
	}

	contextLogger := log.WithFields(log.Fields{
		"destination": h.Destination,
		"port":        h.Port,
	})
	contextLogger.Info("Probing TCP port")

	c, err := net.Dial(
		"tcp",
		h.Destination+":"+h.Port,
	)

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
	return h.VerifyResponse(answer, contextLogger)
}

func TcpConstructor(h Healthcheck) (HealthChecker, error) {
	if _, ok := h.Config["port"]; !ok {
		return TcpHealthCheck{}, errors.New("'port' not defined in tcp healthcheck config to " + h.Destination)
	}

	hc := TcpHealthCheck{
		Destination: h.Destination,
		Port:        h.Config["port"],
		TLS:         h.TlsConnection,
	}
	if v, ok := h.Config["expect"]; ok {
		hc.Expect = v
	}
	if v, ok := h.Config["send"]; ok {
		hc.Send = v
	}
	if val, exists := h.Config["cert"]; h.TlsConnection && exists {
		x509, err := ioutil.ReadFile(val)
		if err != nil {
			return TcpHealthCheck{}, errors.New("'cert' refers to a file that can not be parsed" + val)
		}
		hc.x509 = x509
	}

	if val, exists := h.Config["skipVerify"]; exists {
		skipVerify, err := strconv.ParseBool(val)
		if err != nil {
			return TcpHealthCheck{}, errors.New("'skipVerify' has to be true or false, input: " + val)
		}
		hc.SkipVerify = skipVerify
	}

	if val, exists := h.Config["serverName"]; exists {
		hc.ServerName = string(val)
	}
	return hc, nil
}
