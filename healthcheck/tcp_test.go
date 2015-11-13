package healthcheck

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/bobtfish/AWSnycast/instancemetadata"
	"net"
	"testing"
)

func TestHealthcheckTcpNoPort(t *testing.T) {
	c := make(map[string]string)
	c["send"] = "HEAD / HTTP/1.0\r\n\r\n"
	c["expect"] = "200 OK"
	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	h.Default(instancemetadata.InstanceMetadata{})
	h.Validate("foo", false)
	err := h.Setup()
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "'port' not defined in tcp healthcheck config to 127.0.0.1" {
			t.Log(err.Error())
			t.Fail()
		}
	}
}

func TestHealthcheckTcp(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ready := make(chan bool, 1)
	quit := false
	go func() {
		for {
			ready <- true
			conn, err := ln.Accept()
			if err != nil {
				if quit {
					return
				}
				t.Fatal(fmt.Printf("Error accepting: %s", err.Error()))
			}
			go func(conn net.Conn) {
				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil {
					t.Log(fmt.Printf("Error reading: %s", err.Error()))
					t.Fail()
				}
				if string(buf[:n]) != "HEAD / HTTP/1.0\r\n\r\n" {
					t.Log(string(buf[:n]))
					t.Fail()
				}
				conn.Write([]byte("200 OK"))
				conn.Close()
			}(conn)
		}
	}()
	<-ready
	c := make(map[string]string)
	c["port"] = fmt.Sprintf("%d", port)
	c["send"] = "HEAD / HTTP/1.0\r\n\r\n"
	c["expect"] = "200 OK"
	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	h.Default(instancemetadata.InstanceMetadata{})
	err = h.Validate("foo", false)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = h.Setup()
	if err != nil {
		t.Log("Setup failed: %s", err.Error())
		t.Fail()
	} else {
		log.Printf("%+v", h)
		res := h.healthchecker.Healthcheck()
		if !res {
			t.Log("h.healthchecker.Healthcheck() returned false")
			t.Fail()
		}
	}
	quit = true
	ln.Close()
}

func TestHealthcheckTcpFail(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ready := make(chan bool, 1)
	quit := false
	go func() {
		for {
			ready <- true
			conn, err := ln.Accept()
			if err != nil {
				if quit {
					return
				}
				t.Fatal(fmt.Printf("Error accepting: %s", err.Error()))
			}
			go func(conn net.Conn) {
				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil {
					t.Log(fmt.Printf("Error reading: %s", err.Error()))
					t.Fail()
				}
				if string(buf[:n]) != "HEAD / HTTP/1.0\r\n\r\n" {
					t.Log(string(buf[:n]))
					t.Fail()
				}
				conn.Write([]byte("500 OOPS"))
				conn.Close()
			}(conn)
		}
	}()
	<-ready
	c := make(map[string]string)
	c["port"] = fmt.Sprintf("%d", port)
	c["send"] = "HEAD / HTTP/1.0\r\n\r\n"
	c["expect"] = "200 OK"
	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	h.Default(instancemetadata.InstanceMetadata{})
	err = h.Validate("foo", false)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = h.Setup()
	if err != nil {
		t.Log("Setup failed: %s", err.Error())
		t.Fail()
	} else {
		log.Printf("%+v", h)
		res := h.healthchecker.Healthcheck()
		if res {
			t.Log("h.healthchecker.Healthcheck() returned OK for a 500")
			t.Fail()
		}
	}
	quit = true
	ln.Close()
}

func TestHealthcheckTcpClosed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // Close the port again before running healthcheck
	c := make(map[string]string)
	c["port"] = fmt.Sprintf("%d", port)
	c["send"] = "HEAD / HTTP/1.0\r\n\r\n"
	c["expect"] = "200 OK"
	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	h.Default(instancemetadata.InstanceMetadata{})
	err = h.Validate("foo", false)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = h.Setup()
	if err != nil {
		t.Log("Setup failed: %s", err.Error())
		t.Fail()
	} else {
		log.Printf("%+v", h)
		res := h.healthchecker.Healthcheck()
		if res {
			t.Log("h.healthchecker.Healthcheck() returned OK for closed port")
			t.Fail()
		}
	}
}

func TestHealthcheckTcpFailClientClose(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ready := make(chan bool, 1)
	quit := false
	go func() {
		for {
			ready <- true
			conn, err := ln.Accept()
			if err != nil {
				if quit {
					return
				}
				t.Fatal(fmt.Printf("Error accepting: %s", err.Error()))
			}
			conn.Close() // Client closes connection straight away, before reading anything
		}
	}()
	<-ready
	c := make(map[string]string)
	c["port"] = fmt.Sprintf("%d", port)
	c["send"] = "HEAD / HTTP/1.0\r\n\r\n"
	c["expect"] = "200 OK"
	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	h.Default(instancemetadata.InstanceMetadata{})
	err = h.Validate("foo", false)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = h.Setup()
	if err != nil {
		t.Log("Setup failed: %s", err.Error())
		t.Fail()
	} else {
		log.Printf("%+v", h)
		res := h.healthchecker.Healthcheck()
		if res {
			t.Log("h.healthchecker.Healthcheck() returned OK for client close before send")
			t.Fail()
		}
	}
	quit = true
	ln.Close()
}

func TestHealthcheckTcpNoExpect(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ready := make(chan bool, 1)
	quit := false
	go func() {
		for {
			ready <- true
			conn, err := ln.Accept()
			if err != nil {
				if quit {
					return
				}
				t.Fatal(fmt.Printf("Error accepting: %s", err.Error()))
			}
			go func(conn net.Conn) {
				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil {
					t.Log(fmt.Printf("Error reading: %s", err.Error()))
					t.Fail()
				}
				if string(buf[:n]) != "HEAD / HTTP/1.0\r\n\r\n" {
					t.Log(string(buf[:n]))
					t.Fail()
				}
				conn.Close()
			}(conn)
		}
	}()
	<-ready
	c := make(map[string]string)
	c["port"] = fmt.Sprintf("%d", port)
	c["send"] = "HEAD / HTTP/1.0\r\n\r\n"
	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	h.Default(instancemetadata.InstanceMetadata{})
	err = h.Validate("foo", false)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = h.Setup()
	if err != nil {
		t.Log("Setup failed: %s", err.Error())
		t.Fail()
	} else {
		log.Printf("%+v", h)
		res := h.healthchecker.Healthcheck()
		if !res {
			t.Log("h.healthchecker.Healthcheck() returned false")
			t.Fail()
		}
	}
	quit = true
	ln.Close()
}

func TestHealthcheckTcpNoSendOrExpect(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ready := make(chan bool, 1)
	quit := false
	go func() {
		for {
			ready <- true
			conn, err := ln.Accept()
			if err != nil {
				if quit {
					return
				}
				t.Fatal(fmt.Printf("Error accepting: %s", err.Error()))
			}
			conn.Close() // Client closes connection straight away, before reading anything
		}
	}()
	<-ready
	c := make(map[string]string)
	c["port"] = fmt.Sprintf("%d", port)
	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	h.Default(instancemetadata.InstanceMetadata{})
	err = h.Validate("foo", false)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = h.Setup()
	if err != nil {
		t.Log("Setup failed: %s", err.Error())
		t.Fail()
	} else {
		log.Printf("%+v", h)
		res := h.healthchecker.Healthcheck()
		if !res {
			t.Log("h.healthchecker.Healthcheck() returned FAIL for client close no send")
			t.Fail()
		}
	}
	quit = true
	ln.Close()
}

func TestHealthcheckTcpNoSend(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ready := make(chan bool, 1)
	quit := false
	go func() {
		for {
			ready <- true
			conn, err := ln.Accept()
			if err != nil {
				if quit {
					return
				}
				t.Fatal(fmt.Printf("Error accepting: %s", err.Error()))
			}
			conn.Write([]byte("200 OK"))
			conn.Close()
		}
	}()
	<-ready
	c := make(map[string]string)
	c["port"] = fmt.Sprintf("%d", port)
	c["expect"] = "200 OK"
	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	h.Default(instancemetadata.InstanceMetadata{})
	err = h.Validate("foo", false)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = h.Setup()
	if err != nil {
		t.Log("Setup failed: %s", err.Error())
		t.Fail()
	} else {
		log.Printf("%+v", h)
		res := h.healthchecker.Healthcheck()
		if !res {
			t.Log("h.healthchecker.Healthcheck() returned false")
			t.Fail()
		}
	}
	quit = true
	ln.Close()
}
