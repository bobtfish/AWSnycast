package config

type Config struct {
}

func New(filename string) *Config {
	c := new(Config)
	return c
}
