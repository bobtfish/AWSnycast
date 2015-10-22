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
		log.Println("net.Dial: %s", err.Error())
		return false
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(time.Second * 10))

	_, err = fmt.Fprintf(c, h.Send)
	if err != nil {
		log.Println("TCP send: %s", err.Error())
	}

	if h.Expect == "" {
		return true
	}

	b := make([]byte, 1024)
	n, err := c.Read(b)
	if err != nil {
		log.Println(err)
		return false
	}
	answer := string(b[:n])
	if strings.Contains(answer, h.Expect) {
		return true
	}
	return false
}

func TcpConstructor(h Healthcheck) (HealthChecker, error) {
	if _, ok := h.Config["port"]; !ok {
		return TcpHealthCheck{}, errors.New("'port' not defined in tcp healthcheck config to " + h.Destination)
	}
	hc := TcpHealthCheck{
		Destination: h.Destination,
		Send:        "HEAD / HTTP/1.0\r\n\r\n",
		Port:        h.Config["port"],
	}
	if v, ok := h.Config["expect"]; ok {
		hc.Expect = v
	}
	return hc, nil
}
