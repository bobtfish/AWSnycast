package healthcheck

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
	h.Validate("foo", false)
	err := h.Setup()
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "'port' not defined in tcp healthcheck config to 127.0.0.1")
	}
}

func TestHealthcheckTcp(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if assert.Nil(t, err) {
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
					assert.Nil(t, err)
					assert.Equal(t, string(buf[:n]), "HEAD / HTTP/1.0\r\n\r\n")
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
		err = h.Validate("foo", false)
		assert.Nil(t, err)
		err = h.Setup()
		if assert.Nil(t, err) {
			log.Printf("%+v", h)
			res := h.healthchecker.Healthcheck()
			assert.Equal(t, res, true, "h.healthchecker.Healthcheck() returned false")
		}
		quit = true
		ln.Close()
	}
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
				assert.Nil(t, err, "Error reading")
				assert.Equal(t, string(buf[:n]), "HEAD / HTTP/1.0\r\n\r\n")
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
	err = h.Validate("foo", false)
	assert.Nil(t, err)
	err = h.Setup()
	if assert.Nil(t, err) {
		log.Printf("%+v", h)
		res := h.healthchecker.Healthcheck()
		assert.Equal(t, res, false, "h.healthchecker.Healthcheck() returned OK for a 500")
	}
	quit = true
	ln.Close()
}

func TestHealthcheckTcpClosed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if assert.Nil(t, err) {
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
		err = h.Validate("foo", false)
		assert.Nil(t, err)
		err = h.Setup()
		if assert.Nil(t, err) {
			log.Printf("%+v", h)
			assert.Equal(t, h.healthchecker.Healthcheck(), false, "h.healthchecker.Healthcheck() returned OK for closed port")
		}
	}
}

func TestHealthcheckTcpFailClientClose(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if assert.Nil(t, err) {
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
		err = h.Validate("foo", false)
		assert.Nil(t, err)
		err = h.Setup()
		if assert.Nil(t, err) {
			log.Printf("%+v", h)
			res := h.healthchecker.Healthcheck()
			assert.Equal(t, res, false, "h.healthchecker.Healthcheck() returned OK for client close before send")
		}
		quit = true
		ln.Close()
	}
}

func TestHealthcheckTcpNoExpect(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if assert.Nil(t, err) {
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
					assert.Nil(t, err)
					assert.Equal(t, string(buf[:n]), "HEAD / HTTP/1.0\r\n\r\n")
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
		err = h.Validate("foo", false)
		assert.Nil(t, err)
		err = h.Setup()
		if assert.Nil(t, err) {
			log.Printf("%+v", h)
			assert.Equal(t, h.healthchecker.Healthcheck(), true)
		}
		quit = true
		ln.Close()
	}
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
	err = h.Validate("foo", false)
	assert.Nil(t, err)
	err = h.Setup()

	if assert.Nil(t, err) {
		log.Printf("%+v", h)
		assert.Equal(t, h.healthchecker.Healthcheck(), true, "h.healthchecker.Healthcheck() returned FAIL for client close no send")
	}
	quit = true
	ln.Close()
}

func TestHealthcheckTcpNoSend(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if assert.Nil(t, err) {
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
		err = h.Validate("foo", false)
		assert.Nil(t, err)
		err = h.Setup()

		if assert.Nil(t, err) {
			log.Printf("%+v", h)
			assert.Equal(t, h.healthchecker.Healthcheck(), true, "h.healthchecker.Healthcheck() returned false")
		}
		quit = true
		ln.Close()
	}
}
