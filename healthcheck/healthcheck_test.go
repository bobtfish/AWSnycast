package healthcheck

import (
	"errors"
	"fmt"
	"testing"
)

func TestHealthcheckDefault(t *testing.T) {
	h := Healthcheck{
		Type: "ping",
	}
	h.Default()
	if h.Rise != 2 {
		t.Fail()
	}
	if h.Fall != 3 {
		t.Fail()
	}
	if h.Config == nil {
		t.Fail()
	}
	if h.listeners == nil {
		t.Fail()
	}
}

func TestHealthcheckDefaultLengthRise(t *testing.T) {
	h := Healthcheck{
		Type: "ping",
		Rise: 20,
	}
	h.Default()
	if len(h.History) != 21 {
		t.Fail()
	}
}

func TestHealthcheckDefaultLengthFall(t *testing.T) {
	h := Healthcheck{
		Type: "ping",
		Fall: 20,
	}
	h.Default()
	if len(h.History) != 21 {
		t.Fail()
	}
}

func TestHealthcheckValidateNoType(t *testing.T) {
	h := Healthcheck{
		Destination: "127.0.0.1",
	}
	h.Default()
	err := h.Validate("foo", false)
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "No healthcheck type set" {
			t.Log(err)
			t.Fail()
		}
	}
}

func TestHealthcheckValidateRemoteWithDestFails(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	h.Default()
	err := h.Validate("foo", true)
	if err == nil {
		t.Fail()
	} else {
		if err.Error() != "Remote healthcheck foo cannot have destination set" {
			t.Log(err)
			t.Fail()
		}
	}
}

func TestHealthcheckValidate(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	h.Default()
	err := h.Validate("foo", false)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestHealthcheckValidateFailNoDestination(t *testing.T) {
	h := Healthcheck{
		Type: "notping",
	}
	err := h.Validate("foo", false)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Healthcheck foo has no destination set" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestHealthcheckValidateFailDestination(t *testing.T) {
	h := Healthcheck{
		Type:        "notping",
		Destination: "www.google.com",
	}
	err := h.Validate("foo", false)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Healthcheck foo destination 'www.google.com' does not parse as an IP address" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestHealthcheckValidateFailType(t *testing.T) {
	h := Healthcheck{
		Type:        "notping",
		Destination: "127.0.0.1",
	}
	err := h.Validate("foo", false)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Unknown healthcheck type 'notping' in foo" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestHealthcheckValidateFailRise(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Fall:        1,
		Destination: "127.0.0.1",
	}
	err := h.Validate("foo", false)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "rise must be > 0 in foo" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestHealthcheckValidateFailFall(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Rise:        1,
		Destination: "127.0.0.1",
	}
	err := h.Validate("foo", false)
	if err == nil {
		t.Fail()
	}
	if err.Error() != "fall must be > 0 in foo" {
		t.Log(err.Error())
		t.Fail()
	}
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
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Test" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestHealthcheckGetHealthcheckNotExist(t *testing.T) {
	h := Healthcheck{
		Type:        "test_this_healthcheck_does_not_exist",
		Destination: "127.0.0.1",
	}
	_, err := h.GetHealthChecker()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Healthcheck type 'test_this_healthcheck_does_not_exist' not found in the healthcheck registry" {
		t.Log(err.Error())
		t.Fail()
	}
}

func TestHealthcheckGetHealthcheckNotExistSetup(t *testing.T) {
	h := Healthcheck{
		Type:        "test_this_healthcheck_does_not_exist",
		Destination: "127.0.0.1",
	}
	err := h.Setup()
	if err == nil {
		t.Fail()
	}
	if err.Error() != "Healthcheck type 'test_this_healthcheck_does_not_exist' not found in the healthcheck registry" {
		t.Log(err.Error())
		t.Fail()
	}
}

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

func TestPerformHealthcheckNotSetup(t *testing.T) {
	h := Healthcheck{Type: "test_ok", Destination: "127.0.0.1", Rise: 1}
	defer func() {
		// recover from panic if one occured. Set err to nil otherwise.
		err := recover()
		if err == nil {
			t.Fail()
		}
		if err.(string) != "Setup() never called for healthcheck before Run" {
			t.Fail()
		}
	}()
	h.PerformHealthcheck()
}

func TestHealthcheckRunSimple(t *testing.T) {
	RegisterHealthcheck("test_ok", MyFakeHealthConstructorOk)
	RegisterHealthcheck("test_fail", MyFakeHealthConstructorFail)
	h_ok := Healthcheck{Type: "test_ok", Destination: "127.0.0.1", Rise: 1}
	ok, err := h_ok.GetHealthChecker()
	if err != nil {
		t.Fail()
	}
	h_fail := Healthcheck{Type: "test_fail", Destination: "127.0.0.1"}
	fail, err := h_fail.GetHealthChecker()
	if err != nil {
		t.Fail()
	}
	if !ok.Healthcheck() {
		t.Fail()
	}
	if fail.Healthcheck() {
		t.Fail()
	}
	h_ok.Default()
	h_ok.Setup()
	if h_ok.IsHealthy() {
		t.Fail()
	}
	if h_ok.CanPassYet() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	if !h_ok.CanPassYet() {
		t.Fail()
	}
}

func TestHealthcheckRise(t *testing.T) {
	RegisterHealthcheck("test_ok", MyFakeHealthConstructorOk)
	h_ok := Healthcheck{Type: "test_ok", Destination: "127.0.0.1", Rise: 2}
	h_ok.Default()
	h_ok.Setup()
	if h_ok.IsHealthy() {
		t.Log("Started healthy")
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if h_ok.IsHealthy() {
		t.Log("Became healthy after 1")
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Log("Never became healthy")
		t.Fail()
	}
	h_ok.PerformHealthcheck() // 3
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck() // 10
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	for i, v := range h_ok.History {
		if !v {
			t.Log(fmt.Printf("Index %d was unhealthy", i))
			t.Fail()
		}
	}
}

func TestHealthcheckFall(t *testing.T) {
	RegisterHealthcheck("test_fail", MyFakeHealthConstructorFail)
	h_ok := Healthcheck{Type: "test_fail", Destination: "127.0.0.1", Fall: 2}
	h_ok.Default()
	h_ok.Setup()
	h_ok.History = []bool{true, true, true, true, true, true, true, true, true, true}
	h_ok.isHealthy = true
	if !h_ok.IsHealthy() {
		t.Log("Started unhealthy")
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Log("Became unhealthy after 1 (expected 2)")
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if h_ok.IsHealthy() {
		t.Log("Never became unhealthy")
		t.Fail()
	}
	h_ok.PerformHealthcheck() // 3
	if h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck() // 10
	if h_ok.IsHealthy() {
		t.Fail()
	}
	for i, v := range h_ok.History {
		if v {
			t.Log(fmt.Printf("Index %d was healthy", i))
			t.Fail()
		}
	}
}

func TestHealthcheckFallTen(t *testing.T) {
	RegisterHealthcheck("test_fail", MyFakeHealthConstructorFail)
	h_ok := Healthcheck{Type: "test_fail", Destination: "127.0.0.1", Fall: 10}
	h_ok.Default()
	h_ok.Setup()
	h_ok.History = []bool{true, true, true, true, true, true, true, true, true, true, true}
	h_ok.isHealthy = true
	if !h_ok.IsHealthy() {
		t.Log("Started unhealthy")
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Log("Became unhealthy after 1 (expected 2)")
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Log("Never unhealthy")
		t.Fail()
	}
	h_ok.PerformHealthcheck() // 3
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck()
	if !h_ok.IsHealthy() {
		t.Fail()
	}
	h_ok.PerformHealthcheck() // 10
	if h_ok.IsHealthy() {
		t.Log("Didn't become unhealthy after 10")
		t.Fail()
	}
	h_ok.PerformHealthcheck() // 11 to get false all through history
	for i, v := range h_ok.History {
		if v {
			t.Log(fmt.Printf("Index %d was healthy", i))
			t.Fail()
		}
	}
}

func TestHealthcheckRun(t *testing.T) {
	RegisterHealthcheck("test_ok", MyFakeHealthConstructorOk)
	h_ok := Healthcheck{Type: "test_ok", Destination: "127.0.0.1", Rise: 2}
	h_ok.Default()
	h_ok.Setup()
	h_ok.Run(true)
	if !h_ok.IsRunning() {
		t.Fail()
	}
	h_ok.Run(true)
	if !h_ok.IsRunning() {
		t.Fail()
	}
	h_ok.Stop()
}

func TestHealthcheckStop(t *testing.T) {
	RegisterHealthcheck("test_ok", MyFakeHealthConstructorOk)
	h_ok := Healthcheck{Type: "test_ok", Destination: "127.0.0.1", Rise: 2}
	h_ok.Default()
	h_ok.Setup()
	if h_ok.IsRunning() {
		t.Fail()
	}
	h_ok.Stop()
	if h_ok.IsRunning() {
		t.Fail()
	}
	h_ok.Run(false)
	if !h_ok.IsRunning() {
		t.Fail()
	}
	h_ok.Stop()
	if h_ok.IsRunning() {
		t.Fail()
	}
}

func TestHealthcheckListener(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	h.Default()
	err := h.Validate("foo", false)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = h.Setup()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	c := h.GetListener()
	h.PerformHealthcheck()
	h.PerformHealthcheck()
	val := <-c
	if val != true {
		t.Fail()
	}
}

func TestHealthcheckListenerUnhealthy(t *testing.T) {
	pingCmd = "false"
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	h.Default()
	err := h.Validate("foo", false)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = h.Setup()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	c := h.GetListener()
	h.PerformHealthcheck()
	h.PerformHealthcheck()
	val := <-c
	if val != false {
		t.Fail()
	}
	pingCmd = "ping"
}

func TestChangeDestination(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	h.Default()
	err := h.Validate("foo", false)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = h.Setup()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	h.PerformHealthcheck()
	h.PerformHealthcheck()
	pingCmd = "false"
	if h.canPassYet == false {
		t.Fail()
	}
	if h.runCount != 2 {
		t.Fail()
	}
	n, err := h.NewWithDestination("127.0.0.2")
	if err != nil {
		t.Fail()
	}
	if n.canPassYet == true {
		t.Fail()
	}
	if n.runCount != 0 {
		t.Fail()
	}
	pingCmd = "ping"
}

func TestChangeDestinationFail(t *testing.T) {
	h := Healthcheck{
		Type:        "ping",
		Destination: "127.0.0.1",
	}
	h.Default()
	err := h.Validate("foo", false)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	err = h.Setup()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	h.PerformHealthcheck()
	h.PerformHealthcheck()
	if h.canPassYet == false {
		t.Fail()
	}
	if h.runCount != 2 {
		t.Fail()
	}
	h.Type = "test_this_healthcheck_does_not_exist"
	n, err := h.NewWithDestination("127.0.0.2")
	if err == nil {
		t.Fail()
	}
	if n.canPassYet == true {
		t.Fail()
	}
}
