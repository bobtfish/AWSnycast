package healthcheck

import (
	"fmt"
	"log"
	"net"
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
	b := make([]byte, 1)
	if _, err := c.Read(b); err != nil {
		log.Println(err)
		return false
	}
	return true
}

func TcpConstructor(h Healthcheck) (HealthChecker, error) {
	return TcpHealthCheck{
		Destination: h.Destination,
		Send:        "HEAD / HTTP/1.0\r\n\r\n",
		Expect:      "200 OK",
	}, nil
}
