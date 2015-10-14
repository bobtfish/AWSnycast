package healthcheck

func init() {
	registerHealthcheck("ping", PingConstructor)
}

type PingHealthCheck struct{}

func (h PingHealthCheck) Healthcheck() bool {
	return true // FIXME
}

func PingConstructor(h Healthcheck) (HealthChecker, error) {
	return PingHealthCheck{}, nil
}
