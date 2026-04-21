package main

import (
	"log"

	"orchv3/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Printf(
		"%s starting in %s on port %d",
		cfg.AppName,
		cfg.AppEnv,
		cfg.HTTPPort,
	)
}
