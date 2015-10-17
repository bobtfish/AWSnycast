package healthcheck

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

var healthCheckTypes map[string]func(Healthcheck) (HealthChecker, error)

func registerHealthcheck(name string, f func(Healthcheck) (HealthChecker, error)) {
	if healthCheckTypes == nil {
		healthCheckTypes = make(map[string]func(Healthcheck) (HealthChecker, error))
	}
	healthCheckTypes[name] = f
}

type HealthChecker interface {
	Healthcheck() bool
}

type Healthcheck struct {
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
}

func (h Healthcheck) GetHealthChecker() (HealthChecker, error) {
	if constructor, found := healthCheckTypes[h.Type]; found {
		return constructor(h)
	}
	return nil, errors.New(fmt.Sprintf("Healthcheck type '%s' not found in the healthcheck registry", h.Type))
}

func (h *Healthcheck) Default() {
	if h.Rise == 0 {
		h.Rise = 2
	}
	if h.Fall == 0 {
		h.Fall = 3
	}
	max := h.Rise
	if h.Fall > h.Rise {
		max = h.Rise
	}
	if max < 10 {
		max = 10
	}
	h.History = make([]bool, max)
}

func (h Healthcheck) IsHealthy() bool {
	return h.isHealthy
}

func (h *Healthcheck) PerformHealthcheck() {
	if h.healthchecker == nil {
		panic("Setup() never called for healthcheck before Run")
	}
	result := h.healthchecker.Healthcheck()
	maxIdx := uint(len(h.History) - 1)
	h.History = append(h.History[:1], h.History[2:]...)
	h.History = append(h.History, result)
	log.Printf("History: %s", strings.Join(h.History, ", "))
	if h.isHealthy {
		for i := maxIdx; i > (maxIdx - h.Fall); i-- {
			if h.History[i] {
				return
			}
		}
		log.Printf("Healthcheck %s to %s is unhealthy", h.Type, h.Destination)
		h.isHealthy = false
	} else {
		for i := maxIdx; i > (maxIdx - h.Rise); i-- {
			if !h.History[i] {
				return
			}
		}
		h.isHealthy = true
		log.Printf("Healthcheck %s to %s is healthy", h.Type, h.Destination)
	}
}

func (h Healthcheck) Validate(name string) error {
	if h.Destination == "" {
		return errors.New(fmt.Sprintf("Healthcheck %s has no destination set", name))
	}
	if net.ParseIP(h.Destination) == nil {
		return errors.New(fmt.Sprintf("Healthcheck %s destination '%s' does not parse as an IP address", name, h.Destination))
	}
	if h.Type != "ping" {
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

func (h *Healthcheck) Run() {
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
				log.Println("healthcheck is running")
				h.PerformHealthcheck()
				log.Println("healthcheck has run")
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
