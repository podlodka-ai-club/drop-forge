package main

import (
	"io"

	"orchv3/internal/config"
	"orchv3/internal/steplog"
)

func buildLogger(stderr io.Writer, cfg config.Config, warnOut io.Writer) (steplog.Logger, io.Writer, io.Closer, error) {
	if cfg.Logstash.Addr == "" {
		return steplog.NewWithService(stderr, cfg.AppName), stderr, nil, nil
	}

	sink := steplog.NewTCPSink(
		cfg.Logstash.Addr,
		cfg.Logstash.BufferSize,
		cfg.Logstash.DialTimeout,
		warnOut,
	)

	out := io.MultiWriter(stderr, sink)
	return steplog.NewWithService(out, cfg.AppName), out, sink, nil
}
