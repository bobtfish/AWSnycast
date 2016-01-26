package healthcheck

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net"
	"os"
	"testing"
)

func TestHealthcheckTcpNoPort(t *testing.T) {
	c := make(map[string]interface{})
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
		c := make(map[string]interface{})
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
	c := make(map[string]interface{})
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
		c := make(map[string]interface{})
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
		c := make(map[string]interface{})
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
		c := make(map[string]interface{})
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
	c := make(map[string]interface{})
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
		c := make(map[string]interface{})
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

const serverPEM = `
-----BEGIN CERTIFICATE-----
MIIC2zCCAcOgAwIBAgIQXwEw45ywTG3NCPgR7mWtHDANBgkqhkiG9w0BAQsFADAW
MRQwEgYDVQQKEwtCb290MkRvY2tlcjAgFw0xNjAxMjQwMjMwNDBaGA8xODQ2MDQx
NzAzMjEzMlowFjEUMBIGA1UEChMLQm9vdDJEb2NrZXIwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQC441RhFGz4q6yTRA4qynFhwZ5XJPtZ8D6urPTucQM8
eZOv6ycgJBudERhKYuv2eOkwzQLOOn+4Xv7MXVGwBHYivWFmKuoeDPhqVAvA2Qaq
ZsSY7jrGrpLF0vwTcOE2M5mXO9Hh2hIHTv0y7A2hIZJiqsHUzlNxVpf4au7xdz1S
l1FtBXqYtd/aWYZ2v9oPb/NxMnNBHYZhjPZfX+hOInPTvI2V6GW6qQvjzp6EnpiB
CWGFhdelt59DeR7YaU3esXHiOd18nkeFqMPPH2IDnsloSWFm032Ja4iw3O/cv1gc
3tKFqwiQ023DnHx23MImFNU3MclAvbRf0gz2miq01njZAgMBAAGjIzAhMA4GA1Ud
DwEB/wQEAwICpDAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQAF
sR8zfb5Scai1t0Xiso5aNY+d7Is5QVNwobyC1NLX22igqQAlFSIw0K9SX7GtSaOX
xx+NY30g6U7xSP3lPHYJcRtgfjosdLXGvsSx4wIMF32dadObhdkcRIE9NC86xXSE
l9paIHAoFPfp2l/GCNGe1fsRIx55cExW0Vdi6jClpup/E9IYAOwUxLcSowXLRTni
3Zv/pFjfcocxM2XLr7565GBD0hYazePzIeIUT3lrh19zqpD3pJoIrQwmhcwK15j5
whzIBpBQbSsPVNXzfU55vFnL+ae8C6b/BWvQRCeCQVDD8su00EG2gj2VGqcZ3K1V
LAAPCCgfGVd6oC9U7mUc
-----END CERTIFICATE-----`

const serverKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAuONUYRRs+Kusk0QOKspxYcGeVyT7WfA+rqz07nEDPHmTr+sn
ICQbnREYSmLr9njpMM0Czjp/uF7+zF1RsAR2Ir1hZirqHgz4alQLwNkGqmbEmO46
xq6SxdL8E3DhNjOZlzvR4doSB079MuwNoSGSYqrB1M5TcVaX+Gru8Xc9UpdRbQV6
mLXf2lmGdr/aD2/zcTJzQR2GYYz2X1/oTiJz07yNlehluqkL486ehJ6YgQlhhYXX
pbefQ3ke2GlN3rFx4jndfJ5HhajDzx9iA57JaElhZtN9iWuIsNzv3L9YHN7ShasI
kNNtw5x8dtzCJhTVNzHJQL20X9IM9poqtNZ42QIDAQABAoIBAQCeob32BXZx7aoG
OLAHGucpPtzCXGKkijLd4FvOcXybWJzUpWhOdWyT2OUEfdeDA77gNiYgF6UZ5bes
VK0P3uQVbnOxG0UAP5SsdiwXbJ4BemdX557adVJNFXdens08mg0/6v1rUJWDW8x2
8n3HMGzO5z+jfNuvNOLzK5yT5QDiaKZ5xsJBOi2t6/sAk5ZapwQvXGFG7r3v2PVL
LX/7RUCmgpWHwoLlKRkwcHqkvl6VBUuPtjGSOe80bHiaYcWoTA+M0dZxBfPcKT19
DtzANDG4EgF4L0G6QAGrbyhRq4TFj4Uq02JdFNHVpyPs6SX53vTsC/zil+A0ZIpH
eZjtRh5hAoGBAMGDQwq3zI/D6tc9t7m2KRg3VO+JCeyNFrLnpxLPkmv/RZjt8Bv4
KKa/v6I/+ct0ETRgdJQ3k++uWxB3G+quMR4GK/xZbr8CDdWf3WMyAgUDH4dPl3n2
8nAZP1ObKeWJgLWehLm/y0iJ/CMCuLp+yBlO0SWZVx9demn0PR8jBVg1AoGBAPSX
Gog+rWwPI/Ol1IfUyT4jDbTyQFKtW+qPs8HHDf7xsAT4t4bw24MG7oKrE1gmADtm
PH/X0dI+zugmx6DJnraTtHxDY9ASyJY3JDsCmG7uHGDAIyy2ftU24nU3ods172fI
3xkHSS55XocpsIqThw0GEPZWg3FXguvKAqT0ydqVAoGBAJ1Wwp3mT5bc7wbPEaEX
8VXVN2QDgmQpWzlfjMKIrz7MMaRkYgP7w+HAqmmbpti7qHlzq5YPkmMg2r4KelJY
C2ukDQODG76GRwVYlELhGC9HGM2F812hYgGvJYQu9uPA5zvEhZoZzYlPWAiHX/eS
udOJ+BegE+xWrv+TLFcyvFe5AoGBALFJIUsmGy/bHZUKWy2Fd8TZRaMlgKgszhYL
ySCo9qUXbB1+ZhCiXonvqUv/UosvKDXl2e5Ucdqx+eldyo7p9WejUkxL0HpOUyRG
nbVEIVcuslUSj6xmLzK+kJCkHWa2Bmy0tbj/hfTwtirEdhlL67Tt87eKZ8Xsx5G/
IAGPCQytAoGAeMAa0e6eIk1Pgad2OvLYVC+VVrNNe6EGsCAlSVFsoXFPq4NIFhUL
O/T8Ix3u3678ihTT2MSMJm2ve5Z0YEC/DvuBxTCH/kXjmprpoXTT785KbCHR1TIw
nnK1NVPr4qP5lM6RBh9zodBcU7ZAnERTAORRR/u7ZSWnod475LaPQ1o=
-----END RSA PRIVATE KEY-----`

var (
	tmpTestFakeFile string
)

func TestMain(m *testing.M) {
	if f, err := ioutil.TempFile("/tmp", "ca.pem"); err == nil {
		f.WriteString(serverPEM)
		tmpTestFakeFile = f.Name()
		f.Close()
		defer os.Remove(tmpTestFakeFile)
	}
	os.Exit(m.Run())
}

func TestHealthcheckTcpTLS(t *testing.T) {

	cert, _ := tls.X509KeyPair([]byte(serverPEM), []byte(serverKey))
	clients := x509.NewCertPool()
	clients.AppendCertsFromPEM([]byte(serverPEM))
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    clients,
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
				}(conn)
			}
		}()
		<-ready
		c := make(map[string]interface{})
		c["port"] = fmt.Sprintf("%d", port)
		c["send"] = "HEAD / HTTP/1.0\r\n\r\n"
		c["expect"] = "200 OK"
		c["cert"] = string(serverPEM)
		c["serverName"] = "127.0.0.1"
		c["skipVerify"] = "true"
		c["ssl"] = "true"

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
			assert.Equal(t, true, res, "h.healthchecker.Healthcheck() returned false")
		}
		quit = true
	}
}

func TestHealthcheckTcpTLSSkipVerify(t *testing.T) {

	cert, _ := tls.X509KeyPair([]byte(serverPEM), []byte(serverKey))
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
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
				}(conn)
			}
		}()
		<-ready
		c := make(map[string]interface{})
		c["port"] = fmt.Sprintf("%d", port)
		c["send"] = "HEAD / HTTP/1.0\r\n\r\n"
		c["expect"] = "200 OK"
		c["skipVerify"] = "true"
		c["ssl"] = "true"

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
			assert.Equal(t, true, res, "h.healthchecker.Healthcheck() returned false")
		}
		quit = true
	}
}

func TestHealthcheckTcpTLSEmptyExpect(t *testing.T) {

	cert, _ := tls.X509KeyPair([]byte(serverPEM), []byte(serverKey))
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
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
				}(conn)
			}
		}()
		<-ready
		c := make(map[string]interface{})
		c["port"] = fmt.Sprintf("%d", port)
		c["send"] = "HEAD / HTTP/1.0\r\n\r\n"
		c["expect"] = ""
		c["skipVerify"] = "true"
		c["ssl"] = "true"

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
			assert.Equal(t, true, res, "h.healthchecker.Healthcheck() returned false")
		}
		quit = true
	}
}

func TestHealthcheckTcpTLSFailedParse(t *testing.T) {
	c := make(map[string]interface{})
	c["port"] = "0"
	c["send"] = "HEAD / HTTP/1.0\r\n\r\n"
	c["expect"] = "200 OK"
	c["cert"] = string("Hello")
	c["ssl"] = "true"

	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	err := h.Validate("foo", false)
	assert.Nil(t, err)
	err = h.Setup()
	if assert.Nil(t, err) {
		log.Printf("%+v", h)
		res := h.healthchecker.Healthcheck()
		assert.Equal(t, false, res, "h.healthchecker.Healthcheck() returned false")
	}
}

func TestHealthcheckTcpTLSFailedread(t *testing.T) {

	cert, _ := tls.X509KeyPair([]byte(serverPEM), []byte(serverKey))
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
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
					conn.Close()
				}(conn)
			}
		}()
		<-ready
		c := make(map[string]interface{})
		c["port"] = fmt.Sprintf("%d", port)
		c["send"] = "HEAD / HTTP/1.0\r\n\r\n"
		c["expect"] = "200 OK"
		c["skipVerify"] = "true"
		c["ssl"] = "true"

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
			assert.Equal(t, false, res, "h.healthchecker.Healthcheck() returned false")
		}
		quit = true
	}
}

func TestHealthcheckTcpTLSFailedConnet(t *testing.T) {
	c := make(map[string]interface{})
	c["port"] = "hello"
	c["ssl"] = "true"

	h := Healthcheck{
		Type:        "tcp",
		Destination: "rollover",
		Config:      c,
	}
	err := h.Setup()
	if assert.Nil(t, err) {
		log.Printf("%+v", h)
		res := h.healthchecker.Healthcheck()
		assert.Equal(t, false, res, "h.healthchecker.Healthcheck() returned false")
	}
}

func TestHealthcheckTcpTLSFailedParseSkipVerify(t *testing.T) {
	c := make(map[string]interface{})
	c["port"] = "0"
	c["skipVerify"] = "bye"
	c["ssl"] = "true"

	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	err := h.Setup()
	assert.NotNil(t, err)
}

func TestHealthcheckTcpTLSFailedParseSSL(t *testing.T) {
	c := make(map[string]interface{})
	c["port"] = "0"
	c["ssl"] = "bye"

	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	err := h.Setup()
	assert.NotNil(t, err)
}

func TestHealthcheckTcpTLSFailedCertPath(t *testing.T) {
	c := make(map[string]interface{})
	c["port"] = "0"
	c["certPath"] = "fakeFile"
	c["ssl"] = "true"

	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	err := h.Setup()
	assert.NotNil(t, err)
}

func TestHealthcheckTcpTLSCertPath(t *testing.T) {
	c := make(map[string]interface{})
	c["port"] = "0"
	c["certPath"] = tmpTestFakeFile
	c["ssl"] = "true"

	h := Healthcheck{
		Type:        "tcp",
		Destination: "127.0.0.1",
		Config:      c,
	}
	err := h.Setup()
	assert.Nil(t, err)
}
