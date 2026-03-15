package main

type Config struct {
	ListenAddr     string
	Backends       []string
	BackendSockets int
}

func LoadConfig() Config {

	return Config{
		ListenAddr: "0.0.0.0:2905",

		BackendSockets: 8,

		Backends: []string{
			"127.0.0.1:9001",
			"127.0.0.1:9002",
			"127.0.0.1:9003",
		},
	}
}
