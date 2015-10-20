package healthcheck

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

func init() {
	registerHealthcheck("tcp", TcpConstructor)
}

type TcpHealthCheck struct {
	Destination string
	Port        string
	Send        string
	Expect      string
}

func (h TcpHealthCheck) Healthcheck() bool {
	//func (TcpPlugin) Perform(req *api.StatusRequest, ch chan bool) {
	log.Println("Probing TCP port " + h.Destination + ":" + h.Port)
	// Connect to the remote TCP port
	//
	c, err := net.Dial("tcp", h.Destination+":"+h.Port)
	if err != nil {
		log.Println("net.Dial: %s", err)
		return false
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(time.Second * 10))

	fmt.Fprintf(c, h.Send)
	// Read 1 byte from the TCP connection.
	//
	b := make([]byte, 1024)
	n, err := c.Read(b)
	if err != nil {
		log.Println(err)
		return false
	}
	answer := string(b[:n])
	if h.Expect == "" {
		return true
	} else {
		if strings.Contains(answer, h.Expect) {
			return true
		}
	}
	return false
}

func TcpConstructor(h Healthcheck) (HealthChecker, error) {
	if _, ok := h.Config["port"]; !ok {
		return TcpHealthCheck{}, errors.New("'port' not defined in tcp healthcheck config to " + h.Destination)
	}
	return TcpHealthCheck{
		Destination: h.Destination,
		Send:        "HEAD / HTTP/1.0\r\n\r\n",
		Expect:      "200 OK",
		Port:        h.Config["port"],
	}, nil
}
