package healthcheck

import (
	"AWSnycast/testhelpers"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

type MyFakeHealthCheck struct {
	Healthy bool
}

func (h MyFakeHealthCheck) Healthcheck() bool {
	return h.Healthy
}

func MyFakeHealthConstructorOk(h Healthcheck) (HealthChecker, error) {
	return MyFakeHealthCheck{Healthy: true}, nil
}

func MyFakeHealthConstructorFail(h Healthcheck) (HealthChecker, error) {
	return MyFakeHealthCheck{Healthy: false}, nil
}

func TestHealthcheckDefault(t *testing.T) {
	h := Healthcheck{
		Type: "ping",
	}
	h.Validate("foo", false)
	assert.Equal(t, h.Rise, uint(2))
	assert.Equal(t, h.Fall, uint(3))
	assert.NotNil(t, h.Config)
	assert.NotNil(t, h.listeners)
}

func TestHealthcheckDefaultLengthRise(t *testing.T) {
	h := Healthcheck{
		Type: "ping",
		Rise: 20,
	}
	h.Validate("foo", false)
	assert.Equal(t, len(h.History), 21)
}

func TestHealthcheckDefaultLengthFall(t *testing.T) {
	h := Healthcheck{
		Type: "ping",
		Fall: 20,
	}
	h.Validate("foo", false)
	assert.Equal(t, len(h.History), 21)
}

func TestHealthcheckValidateNoType(t *testing.T) {
	h := Healthcheck{
		Destination: "127.0.0.1",
	}
	err := h.Validate("foo", false)
	testhelpers.CheckOneMultiError(t, err, "No healthcheck type set")
}

func TestHealthcheckValidateRemoteWithDestFails(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	err := h.Validate("foo", true)
	testhelpers.CheckOneMultiError(t, err, "Remote healthcheck foo cannot have destination set")
}

func TestHealthcheckValidate(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	err := h.Validate("foo", false)
	assert.Nil(t, err)
}

func TestHealthcheckValidateFailNoDestination(t *testing.T) {
	h := Healthcheck{
		Type: "ping",
	}
	err := h.Validate("foo", false)
	testhelpers.CheckOneMultiError(t, err, "Healthcheck foo has no destination set")
}

func TestHealthcheckValidateFailDestination(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "www.google.com",
	}
	err := h.Validate("foo", false)
	testhelpers.CheckOneMultiError(t, err, "Healthcheck foo destination 'www.google.com' does not parse as an IP address")
}

func TestHealthcheckValidateFailType(t *testing.T) {
	h := Healthcheck{
		Type:        "notping",
		Destination: "127.0.0.1",
	}
	err := h.Validate("foo", false)
	testhelpers.CheckOneMultiError(t, err, "Unknown healthcheck type 'notping' in foo")
}

func myHealthCheckConstructorFail(h Healthcheck) (HealthChecker, error) {
	return nil, errors.New("Test")
}

func TestHealthcheckRegisterNew(t *testing.T) {
	RegisterHealthcheck("testconstructorfail", myHealthCheckConstructorFail)
	h := Healthcheck{
		Type:        "testconstructorfail",
		Destination: "127.0.0.1",
	}
	_, err := h.GetHealthChecker()
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Test")
	}
}

func TestHealthcheckGetHealthcheckNotExist(t *testing.T) {
	h := Healthcheck{
		Type:        "test_this_healthcheck_does_not_exist",
		Destination: "127.0.0.1",
	}
	_, err := h.GetHealthChecker()
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Healthcheck type 'test_this_healthcheck_does_not_exist' not found in the healthcheck registry")
	}
}

func TestHealthcheckGetHealthcheckNotExistSetup(t *testing.T) {
	h := Healthcheck{
		Type:        "test_this_healthcheck_does_not_exist",
		Destination: "127.0.0.1",
	}
	err := h.Setup()
	if assert.NotNil(t, err) {
		assert.Equal(t, err.Error(), "Healthcheck type 'test_this_healthcheck_does_not_exist' not found in the healthcheck registry")
	}
}

func TestPerformHealthcheckNotSetup(t *testing.T) {
	h := Healthcheck{Type: "test_ok", Destination: "127.0.0.1", Rise: 1}
	defer func() {
		// recover from panic if one occured. Set err to nil otherwise.
		err := recover()
		if assert.NotNil(t, err) {
			assert.Equal(t, err.(string), "Setup() never called for healthcheck before Run")
		}
	}()
	h.PerformHealthcheck()
}

func TestHealthcheckRunSimple(t *testing.T) {
	RegisterHealthcheck("test_ok", MyFakeHealthConstructorOk)
	RegisterHealthcheck("test_fail", MyFakeHealthConstructorFail)
	h_ok := Healthcheck{Type: "test_ok", Destination: "127.0.0.1", Rise: 1}
	ok, err := h_ok.GetHealthChecker()
	assert.Nil(t, err)
	h_fail := Healthcheck{Type: "test_fail", Destination: "127.0.0.1"}
	fail, err := h_fail.GetHealthChecker()
	assert.Nil(t, err)
	assert.Equal(t, ok.Healthcheck(), true)
	assert.Equal(t, fail.Healthcheck(), false)
	h_ok.Validate("foo", false)
	h_ok.Setup()
	assert.Equal(t, h_ok.IsHealthy(), false)
	assert.Equal(t, h_ok.CanPassYet(), false)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	assert.Equal(t, h_ok.CanPassYet(), true)
}

func TestHealthcheckRise(t *testing.T) {
	RegisterHealthcheck("test_ok", MyFakeHealthConstructorOk)
	h_ok := Healthcheck{Type: "test_ok", Destination: "127.0.0.1", Rise: 2}
	h_ok.Validate("foo", false)
	h_ok.Setup()
	assert.Equal(t, h_ok.IsHealthy(), false, "Started healthy")
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), false, "Became healthy after 1")
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true, "Never became healthy")
	h_ok.PerformHealthcheck() // 3
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck() // 10
	assert.Equal(t, h_ok.IsHealthy(), true)
	for i, v := range h_ok.History {
		assert.Equal(t, v, true, fmt.Sprintf("Index %d was unhealthy", i))
	}
}

func TestHealthcheckFall(t *testing.T) {
	RegisterHealthcheck("test_fail", MyFakeHealthConstructorFail)
	h_ok := Healthcheck{Type: "test_fail", Destination: "127.0.0.1", Fall: 2}
	h_ok.Validate("foo", false)
	h_ok.Setup()
	h_ok.History = []bool{true, true, true, true, true, true, true, true, true, true}
	h_ok.isHealthy = true
	assert.Equal(t, h_ok.IsHealthy(), true, "Started unhealthy")
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true, "Became unhealthy after 1 (expected 2)")
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), false, "Never became unhealthy")
	h_ok.PerformHealthcheck() // 3
	assert.Equal(t, h_ok.IsHealthy(), false)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), false)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), false)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), false)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), false)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), false)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), false)
	h_ok.PerformHealthcheck() // 10
	assert.Equal(t, h_ok.IsHealthy(), false)
	for i, v := range h_ok.History {
		assert.Equal(t, v, false, fmt.Sprintf("Index %d was healthy", i))
	}
}

func TestHealthcheckFallTen(t *testing.T) {
	RegisterHealthcheck("test_fail", MyFakeHealthConstructorFail)
	h_ok := Healthcheck{Type: "test_fail", Destination: "127.0.0.1", Fall: 10}
	h_ok.Validate("foo", false)
	h_ok.Setup()
	h_ok.History = []bool{true, true, true, true, true, true, true, true, true, true, true}
	h_ok.isHealthy = true
	assert.Equal(t, h_ok.IsHealthy(), true, "Started unhealthy")
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true, "Became unhealthy after 1 (expected 2)")
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true, "Never unhealthy")
	h_ok.PerformHealthcheck() // 3
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck()
	assert.Equal(t, h_ok.IsHealthy(), true)
	h_ok.PerformHealthcheck() // 10
	assert.Equal(t, h_ok.IsHealthy(), false, "Didn't become unhealthy after 10")
	h_ok.PerformHealthcheck() // 11 to get false all through history
	for i, v := range h_ok.History {
		assert.Equal(t, v, false, fmt.Sprintf("Index %d was healthy", i))
	}
}

func TestHealthcheckRun(t *testing.T) {
	RegisterHealthcheck("test_ok", MyFakeHealthConstructorOk)
	h_ok := Healthcheck{Type: "test_ok", Destination: "127.0.0.1", Rise: 2}
	assert.Nil(t, h_ok.Validate("foo", false))
	h_ok.Setup()
	h_ok.Run(true)
	assert.Equal(t, h_ok.IsRunning(), true)
	h_ok.Run(true)
	assert.Equal(t, h_ok.IsRunning(), true)
	h_ok.Stop()
}

func TestHealthcheckStop(t *testing.T) {
	RegisterHealthcheck("test_ok", MyFakeHealthConstructorOk)
	h_ok := Healthcheck{Type: "test_ok", Destination: "127.0.0.1", Rise: 2}
	assert.Nil(t, h_ok.Validate("foo", false))
	h_ok.Setup()
	assert.Equal(t, h_ok.IsRunning(), false)
	h_ok.Stop()
	assert.Equal(t, h_ok.IsRunning(), false)
	h_ok.Run(false)
	assert.Equal(t, h_ok.IsRunning(), true)
	h_ok.Stop()
	assert.Equal(t, h_ok.IsRunning(), false)
}

func TestHealthcheckListener(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	assert.Nil(t, h.Validate("foo", false))
	assert.Nil(t, h.Setup())
	c := h.GetListener()
	h.PerformHealthcheck()
	h.PerformHealthcheck()
	assert.Equal(t, <-c, true)
}

func TestHealthcheckListenerUnhealthy(t *testing.T) {
	pingCmd = "false"
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	assert.Nil(t, h.Validate("foo", false))
	assert.Nil(t, h.Setup())
	c := h.GetListener()
	h.PerformHealthcheck()
	h.PerformHealthcheck()
	assert.Equal(t, <-c, false)
	pingCmd = "ping"
}

func TestChangeDestination(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	assert.Nil(t, h.Validate("foo", false))
	assert.Nil(t, h.Setup())
	h.PerformHealthcheck()
	h.PerformHealthcheck()
	pingCmd = "false"
	assert.Equal(t, h.canPassYet, true)
	assert.Equal(t, h.runCount, uint64(2))
	n, err := h.NewWithDestination("127.0.0.2")
	assert.Nil(t, err)
	assert.Equal(t, n.canPassYet, false)
	assert.Equal(t, n.runCount, uint64(0))
	pingCmd = "ping"
}

func TestChangeDestinationFail(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	assert.Nil(t, h.Validate("foo", false))
	assert.Nil(t, h.Setup())
	h.PerformHealthcheck()
	h.PerformHealthcheck()
	assert.Equal(t, h.canPassYet, true)
	assert.Equal(t, h.runCount, uint64(2))
	h.Type = "test_this_healthcheck_does_not_exist"
	n, err := h.NewWithDestination("127.0.0.2")
	assert.NotNil(t, err)
	assert.Equal(t, n.canPassYet, false)
}

func TestHealthcheckRunOnHealthy(t *testing.T) {
	dir, err := ioutil.TempDir("", "awsnycast")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up
	flagFile := dir + "/run_on_healthy"
	h := Healthcheck{
		Type:         "ping",
		Destination:  "127.0.0.1",
		RunOnHealthy: []string{"/usr/bin/touch", flagFile},
	}
	assert.Nil(t, h.Validate("foo", false))
	assert.Nil(t, h.Setup())
	if _, err := os.Stat(flagFile); err == nil {
		t.Log(flagFile + " exists already")
		t.Fail()
	}
	c := h.GetListener()
	h.PerformHealthcheck()
	h.PerformHealthcheck()
	assert.Equal(t, <-c, true)
	if _, err := os.Stat(flagFile); os.IsNotExist(err) {
		t.Log(flagFile + " does not exist")
		t.Fail()
	}
}

func TestHealthcheckRunOnUnhealthy(t *testing.T) {
	pingCmd = "false"
	dir, err := ioutil.TempDir("", "awsnycast")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up
	flagFile := dir + "/run_on_unhealthy"
	h := Healthcheck{
		Type:           "ping",
		Destination:    "127.0.0.1",
		RunOnUnhealthy: []string{"/usr/bin/touch", flagFile},
	}
	assert.Nil(t, h.Validate("foo", false))
	assert.Nil(t, h.Setup())
	if _, err := os.Stat(flagFile); err == nil {
		t.Log(flagFile + " exists already")
		t.Fail()
	}
	c := h.GetListener()
	h.PerformHealthcheck()
	h.PerformHealthcheck()
	assert.Equal(t, <-c, false)
	if _, err := os.Stat(flagFile); os.IsNotExist(err) {
		t.Log(flagFile + " does not exist")
		t.Fail()
	}
	pingCmd = "ping"
}
