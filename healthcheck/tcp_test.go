package healthcheck

import (
	"crypto/tls"
    "crypto/x509"
    "encoding/pem"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

const serverPEM = `
-----BEGIN CERTIFICATE-----
MIIEBDCCAuygAwIBAgIDAjppMA0GCSqGSIb3DQEBBQUAMEIxCzAJBgNVBAYTAlVT
MRYwFAYDVQQKEw1HZW9UcnVzdCBJbmMuMRswGQYDVQQDExJHZW9UcnVzdCBHbG9i
YWwgQ0EwHhcNMTMwNDA1MTUxNTU1WhcNMTUwNDA0MTUxNTU1WjBJMQswCQYDVQQG
EwJVUzETMBEGA1UEChMKR29vZ2xlIEluYzElMCMGA1UEAxMcR29vZ2xlIEludGVy
bmV0IEF1dGhvcml0eSBHMjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AJwqBHdc2FCROgajguDYUEi8iT/xGXAaiEZ+4I/F8YnOIe5a/mENtzJEiaB0C1NP
VaTOgmKV7utZX8bhBYASxF6UP7xbSDj0U/ck5vuR6RXEz/RTDfRK/J9U3n2+oGtv
h8DQUB8oMANA2ghzUWx//zo8pzcGjr1LEQTrfSTe5vn8MXH7lNVg8y5Kr0LSy+rE
ahqyzFPdFUuLH8gZYR/Nnag+YyuENWllhMgZxUYi+FOVvuOAShDGKuy6lyARxzmZ
EASg8GF6lSWMTlJ14rbtCMoU/M4iarNOz0YDl5cDfsCx3nuvRTPPuj5xt970JSXC
DTWJnZ37DhF5iR43xa+OcmkCAwEAAaOB+zCB+DAfBgNVHSMEGDAWgBTAephojYn7
qwVkDBF9qn1luMrMTjAdBgNVHQ4EFgQUSt0GFhu89mi1dvWBtrtiGrpagS8wEgYD
VR0TAQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAQYwOgYDVR0fBDMwMTAvoC2g
K4YpaHR0cDovL2NybC5nZW90cnVzdC5jb20vY3Jscy9ndGdsb2JhbC5jcmwwPQYI
KwYBBQUHAQEEMTAvMC0GCCsGAQUFBzABhiFodHRwOi8vZ3RnbG9iYWwtb2NzcC5n
ZW90cnVzdC5jb20wFwYDVR0gBBAwDjAMBgorBgEEAdZ5AgUBMA0GCSqGSIb3DQEB
BQUAA4IBAQA21waAESetKhSbOHezI6B1WLuxfoNCunLaHtiONgaX4PCVOzf9G0JY
/iLIa704XtE7JW4S615ndkZAkNoUyHgN7ZVm2o6Gb4ChulYylYbc3GrKBIxbf/a/
zG+FA1jDaFETzf3I93k9mTXwVqO94FntT0QJo544evZG0R0SnU++0ED8Vf4GXjza
HFa9llF7b1cq26KqltyMdMKVvvBulRP/F/A8rLIQjcxz++iPAsbw+zOzlTvjwsto
WHPbqCRiOwY1nQ2pM714A5AuTHhdUDqB1O6gyHA43LL5Z/qHQF1hwFGPa4NrzQU6
yuGnBXj8ytqU0CwIPX4WecigUCAkVDNx
-----END CERTIFICATE-----`

const serverKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA3Wkt22Q+iH1tPaF3Oo2zZQ+pRl3ae2ngyHqwmNN3TbzXqXjv
vavm0R7BrF8TIMNQpUeimhJWnSiaebgov8RkY8Y80H5+1rLzg666Q3NyPl9e8d2k
Ant/CnDO+LkoOO63WN6A+zvsgLYrNGmgREwQ2aWQo3WSiM6Q5nVtHsQXw+eZMLy/
3IDk0D9b/os8BNEs+fKhZ5PVDRsdqbpCKllfueL+lw48mgSW4JtxQScNloVtmHBp
yTToD57TiM+BsObrpuV0HwccovfV0KuHbJtQKkt6RFCzkCOjB6pK2GZI3VPOzk3G
glDUf3ph0Uzy9xezWILuRVghydnmbXtZ9P9KVwIDAQABAoIBAQCpvc/lKUXzl8ze
+eGRJz9IFCifBKbSBIrKx5yJnV0SYNspVsjdLWOIIL8z6bOdY395JqEW40Y5t/4t
oKzEz8hy4XCQGtocuRaNdq/ftCr6fF0qqdPp/0mW9IMIFdMrSn+Phw11DiyzRHV8
+hlfIr2sTgzD9AgCD/Lwa0fb1VOu4HqaF31qBVvf0J3+AuwnpOS8sEQvJGmJgFPp
A3Y62WjxdqbwqW/Mdb1RO42pWDOy0yGY2uPSIS60St/GMZNx0Jasq5Lu3lIVFfVl
HFb4xW+ziB78FYHC1IWMk61ogea9lrr62f8VXgLoie+uqtM+R3PRXMflfpAS5a5c
rFr1U1fZAoGBAPkm/o6jagxBFJIDiqQ+W/9ghwHiiT708AShuIj/SiI2sMnxI5aY
s5r9lTLB0/jz0Z5iu19wKxBAz3VsM0uujVvIrzZaKYaLMkOQxipg74mouLnh7zLY
LZ93//XYZfE++ot+cvc9s46UZuGur81sugGPlb3ivluhZWMcqfc2qJi1AoGBAON+
/8kpDkGCkwXyvk2Qt8/LAyf30uWhqfzKrwQknasjHLjaRfkf0E4K5TEeJ2oJZIUw
/IxzEet3y7Qs0P1BpdcdB6CB4d3WOfblPN0RZV31BvIc3n/8/rMJGqV7wgqh0jx7
ObqpOXNvsh/TuhAEhp5xvSZYnJa9ScsMu1ClmDpbAoGAHxUuTL02TbEQz+aBNVxS
Pdnc/e81EBWem/VRAEZZCUupYogi2HbUcVGRe3OS7kv8qrXGinGD7dMoDo4hGB/+
oqS2tyEobRCQhL1a+458U8Aoy4fUP5OYnXxrAlCs5xvkReLQlOetruv0qdMRO5+E
1Q0EsVvIQ8Yuz96TlbPL9MECgYAR3rwA9TSleLhL01GXjjKiI/RPg2wRla1gqhst
XCL2en+bFapBc3pNZxWx0giOj8ZRoBN2hON3d6WMtaiE/E8moqUiupEfd+B9wGwT
gXZQ9xpgklv3+cuYDLMHJL2NUEDPd26Fdx2IL9HyJhOLho98irqs9HD4dk4BoTJl
l1xp2QKBgQD1vqdIeokdVcHKUziQJ0G+X4EmvBjVDaH+4Ho1nFUfHgPT1EJhyu4j
f78Yac/8I4KwnmFq85ODBy6dSLoftVx7o4eU6tPrNNQ7w2r3gorwLI+oJU6/Nhyx
Oks6gY7X5f6uv5yl7P6gUl7744zEjj5ef9oD1v6+9JSlb7lTWpmp5g==
-----END RSA PRIVATE KEY-----`


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

func TestHealthcheckTcpTLSSkipVerify(t *testing.T) {
    block, _ := pem.Decode([]byte(serverPEM))
    ca, _ := x509.ParseCertificate(block.Bytes)
    block, _ = pem.Decode([]byte(serverKey))
    priv, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

    pool := x509.NewCertPool()
    pool.AddCert(ca)

    cert := tls.Certificate{
        Certificate: [][]byte{ []byte(serverPEM) },
        PrivateKey: priv,
    }

    config := &tls.Config{
        ClientAuth: tls.RequireAndVerifyClientCert,
        Certificates: []tls.Certificate{cert},
        ClientCAs: pool,
    }

	ln, err := tls.Listen("tcp", "127.0.0.1:0", config)
	defer ln.Close()
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
        c["serverName"] = "example.com"
        c["skipVerify"] = "true"

        h := Healthcheck{
            Type:          "tcp",
            Destination:   "127.0.0.1",
            Config:        c,
            TlsConnection: true,
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
    }
}
