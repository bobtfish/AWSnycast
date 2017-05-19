package healthcheck

import (
	"errors"
	"fmt"
	log "github.com/bobtfish/logrus"
	"github.com/hashicorp/go-multierror"
	"net"
	"os/exec"
	"time"
)

var healthCheckTypes map[string]func(Healthcheck) (HealthChecker, error)

func RegisterHealthcheck(name string, f func(Healthcheck) (HealthChecker, error)) {
	if healthCheckTypes == nil {
		healthCheckTypes = make(map[string]func(Healthcheck) (HealthChecker, error))
	}
	healthCheckTypes[name] = f
}

type HealthChecker interface {
	Healthcheck() bool
}

type CanBeHealthy interface {
	IsHealthy() bool
	GetListener() <-chan bool
	CanPassYet() bool
}

type Healthcheck struct {
	canPassYet     bool                   `yaml:"-"`
	runCount       uint64                 `yaml:"-"`
	Type           string                 `yaml:"type"`
	Destination    string                 `yaml:"destination"`
	isHealthy      bool                   `yaml:"-"`
	Rise           uint                   `yaml:"rise"`
	Fall           uint                   `yaml:"fall"`
	Every          uint                   `yaml:"every"`
	History        []bool                 `yaml:"-"`
	Config         map[string]interface{} `yaml:"config"`
	RunOnHealthy   []string               `yaml:"run_on_healthy"`
	RunOnUnhealthy []string               `yaml:"run_on_unhealthy"`
	healthchecker  HealthChecker          `yaml:"-"`
	isRunning      bool                   `yaml:"-"`
	quitChan       chan<- bool            `yaml:"-"`
	hasQuitChan    <-chan bool            `yaml:"-"`
	listeners      []chan<- bool          `yaml:"-"`
}

func (h *Healthcheck) NewWithDestination(destination string) (*Healthcheck, error) {
	n := &Healthcheck{
		Destination:    destination,
		Type:           h.Type,
		Rise:           h.Rise,
		Fall:           h.Fall,
		Every:          h.Every,
		Config:         h.Config,
		RunOnHealthy:   h.RunOnHealthy,
		RunOnUnhealthy: h.RunOnUnhealthy,
	}
	err := n.Validate(destination, false)
	if err == nil {
		err = n.Setup()
	}
	log.WithFields(log.Fields{
		"destination": n.Destination,
		"type":        n.Type,
		"err":         err,
	}).Info("Made new remote healthcheck")
	return n, err
}

func (h *Healthcheck) GetListener() <-chan bool {
	c := make(chan bool, 5)
	h.listeners = append(h.listeners, c)
	return c
}

func (h *Healthcheck) stateChange() {
	contextLogger := log.WithFields(log.Fields{
		"destination": h.Destination,
		"type":        h.Type,
	})
	h.canPassYet = true
	if h.isHealthy {
		if len(h.RunOnHealthy) > 0 {
			cmd := h.RunOnHealthy[0]
			if err := exec.Command(cmd, h.RunOnHealthy[1:]...).Run(); err != nil {
				contextLogger.WithFields(log.Fields{"err": err.Error()}).Debug("healthcheck RunOnHealthy failed")
			}
		}
	} else {
		if len(h.RunOnUnhealthy) > 0 {
			cmd := h.RunOnUnhealthy[0]
			if err := exec.Command(cmd, h.RunOnUnhealthy[1:]...).Run(); err != nil {
				contextLogger.WithFields(log.Fields{"err": err.Error()}).Debug("healthcheck RunOnUnhealthy failed")
			}
		}
	}
	for _, l := range h.listeners {
		l <- h.isHealthy
	}
}

func (h *Healthcheck) CanPassYet() bool {
	return h.canPassYet
}

func (h Healthcheck) GetHealthChecker() (HealthChecker, error) {
	if constructor, found := healthCheckTypes[h.Type]; found {
		return constructor(h)
	}
	return nil, errors.New(fmt.Sprintf("Healthcheck type '%s' not found in the healthcheck registry", h.Type))
}

func (h Healthcheck) IsHealthy() bool {
	return h.isHealthy
}

func (h *Healthcheck) PerformHealthcheck() {
	if h.healthchecker == nil {
		panic("Setup() never called for healthcheck before Run")
	}
	h.runCount = h.runCount + 1
	result := h.healthchecker.Healthcheck()
	maxIdx := uint(len(h.History) - 1)
	h.History = append(h.History[:0], h.History[1:]...)
	h.History = append(h.History, result)
	contextLogger := log.WithFields(log.Fields{
		"destination": h.Destination,
		"type":        h.Type,
	})
	if h.isHealthy {
		downTo := maxIdx - h.Fall + 1
		for i := maxIdx; i >= downTo; i-- {
			if h.History[i] {
				return
			}
		}
		contextLogger.Info("Healthcheck is unhealthy")
		h.isHealthy = false
		h.stateChange()
	} else { // Currently unhealthy
		downTo := maxIdx - h.Rise + 1
		for i := maxIdx; i >= downTo; i-- {
			if !h.History[i] { // Still unhealthy
				if h.runCount == uint64(h.Rise) { // We just started running, and *could* have come healthy, but didn't,
					h.stateChange() // so lets inform anyone listening, in case they want to take action
				}
				return
			}
		}
		h.isHealthy = true
		contextLogger.Info("Healthcheck is healthy")
		h.stateChange()
	}
}

func (h *Healthcheck) Validate(name string, remote bool) error {
	if h.Config == nil {
		h.Config = make(map[string]interface{})
	}
	if h.Rise == 0 {
		h.Rise = 2
	}
	if h.Fall == 0 {
		h.Fall = 3
	}
	max := h.Rise
	if h.Fall > h.Rise {
		max = h.Fall
	}
	max = max + 1 // Avoid integer overflow in the loop counting down by keeping 1 more check than we need.
	if max < 10 {
		max = 10
	}
	h.History = make([]bool, max)
	h.listeners = make([]chan<- bool, 0)
	var result *multierror.Error
	if !remote {
		if h.Destination == "" {
			result = multierror.Append(result, errors.New(fmt.Sprintf("Healthcheck %s has no destination set", name)))
		} else {
			if net.ParseIP(h.Destination) == nil {
				result = multierror.Append(result, errors.New(fmt.Sprintf("Healthcheck %s destination '%s' does not parse as an IP address", name, h.Destination)))
			}
		}
	} else {
		if h.Destination != "" {
			result = multierror.Append(result, errors.New(fmt.Sprintf("Remote healthcheck %s cannot have destination set", name)))
		}
	}
	if h.Type == "" {
		result = multierror.Append(result, errors.New("No healthcheck type set"))
	} else {
		if _, found := healthCheckTypes[h.Type]; !found {
			result = multierror.Append(result, errors.New(fmt.Sprintf("Unknown healthcheck type '%s' in %s", h.Type, name)))
		}
	}
	return result.ErrorOrNil()
}

func (h *Healthcheck) Setup() error {
	hc, err := h.GetHealthChecker()
	if err != nil {
		return err
	}
	h.healthchecker = hc
	return nil
}

func sleepAndSend(t uint, send chan<- bool) {
	go func() {
		time.Sleep(time.Duration(t) * time.Second)
		send <- true
	}()
}

func (h *Healthcheck) Run(debug bool) {
	if h.isRunning {
		return
	}
	hasquit := make(chan bool)
	quit := make(chan bool)
	run := make(chan bool)
	go func() { // Simple and dumb runner. Runs healthcheck and then sleeps the 'Every' time.
	Loop: // Healthchecks are expected to complete much faster than the Every time!
		for {
			select {
			case <-quit:
				log.Debug("Healthcheck is exiting")
				break Loop
			case <-run:
				log.Debug("Healthcheck is running")
				h.PerformHealthcheck()
				log.Debug("Healthcheck has run")
				sleepAndSend(h.Every, run) // Queue the next run up
			}
		}
		hasquit <- true
		close(hasquit)
	}()
	h.hasQuitChan = hasquit
	h.quitChan = quit
	h.isRunning = true
	run <- true // Fire straight away once set running
}

func (h Healthcheck) IsRunning() bool {
	return h.isRunning
}

func (h *Healthcheck) Stop() {
	if !h.IsRunning() {
		return
	}
	h.quitChan <- true
	close(h.quitChan)
	<-h.hasQuitChan // Block till finished
	h.quitChan = nil
	h.hasQuitChan = nil
	h.isRunning = false
}
