package healthcheck

import (
	"fmt"
	"log"
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
	h.Default()
	h.Validate("foo")
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
	h.Default()
	err = h.Validate("foo")
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
	h.Default()
	err = h.Validate("foo")
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
			t.Log(fmt.Sprintf("%+v", res))
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
	h.Default()
	err = h.Validate("foo")
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
			t.Log(fmt.Sprintf("%+v", res))
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
	h.Default()
	err = h.Validate("foo")
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
			t.Log(fmt.Sprintf("%+v", res))
			t.Fail()
		}
	}
	quit = true
	ln.Close()
}
