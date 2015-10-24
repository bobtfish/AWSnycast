package healthcheck

import (
	"errors"
	"fmt"
	"log"
	"net"
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
	canPassYet    bool   `yaml:"-"`
	runCount      uint64
	Type          string `yaml:"type"`
	Destination   string `yaml:"destination"`
	isHealthy     bool   `yaml:"-"`
	Rise          uint   `yaml:"rise"`
	Fall          uint   `yaml:"fall"`
	Every         uint   `yaml:"every"`
	History       []bool `yaml:"-"`
	healthchecker HealthChecker
	isRunning     bool
	quitChan      chan<- bool
	hasQuitChan   <-chan bool
	Config        map[string]string
	listeners     []chan<- bool
}

func (h *Healthcheck) GetListener() <-chan bool {
	c := make(chan bool, 5)
	h.listeners = append(h.listeners, c)
	return c
}

func (h *Healthcheck) stateChange() {
	h.canPassYet = true
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

func (h *Healthcheck) Default() {
	if h.Config == nil {
		h.Config = make(map[string]string)
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
	if h.isHealthy {
		downTo := maxIdx - h.Fall + 1
		for i := maxIdx; i >= downTo; i-- {
			if h.History[i] {
				return
			}
		}
		log.Printf("Healthcheck %s to %s is unhealthy", h.Type, h.Destination)
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
		log.Printf("Healthcheck %s to %s is healthy", h.Type, h.Destination)
		h.stateChange()
	}
}

func (h Healthcheck) Validate(name string) error {
	if h.Destination == "" {
		return errors.New(fmt.Sprintf("Healthcheck %s has no destination set", name))
	}
	if net.ParseIP(h.Destination) == nil {
		return errors.New(fmt.Sprintf("Healthcheck %s destination '%s' does not parse as an IP address", name, h.Destination))
	}
	if h.Type == "" {
		return errors.New("No healthcheck type set")
	}
	if _, found := healthCheckTypes[h.Type]; !found {
		return errors.New(fmt.Sprintf("Unknown healthcheck type '%s' in %s", h.Type, name))
	}
	if h.Rise == 0 {
		return errors.New(fmt.Sprintf("rise must be > 0 in %s", name))
	}
	if h.Fall == 0 {
		return errors.New(fmt.Sprintf("fall must be > 0 in %s", name))
	}
	return nil
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
				log.Println("healthcheck is exiting")
				break Loop
			case <-run:
				if debug {
					log.Println("healthcheck is running")
				}
				h.PerformHealthcheck()
				if debug {
					log.Println("healthcheck has run")
				}
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
