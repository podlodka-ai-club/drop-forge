package main

import (
	"io"
	"os"

	"orchv3/internal/config"
	"orchv3/internal/steplog"
)

func buildLogger(stderr io.Writer, cfg config.Config, warnOut io.Writer) (steplog.Logger, io.Writer, io.Closer, error) {
	return buildLoggerWithTerminalCheck(stderr, cfg, warnOut, isTerminalWriter)
}

func buildLoggerWithTerminalCheck(
	stderr io.Writer,
	cfg config.Config,
	warnOut io.Writer,
	isTerminal func(io.Writer) bool,
) (steplog.Logger, io.Writer, io.Closer, error) {
	consoleOut := consoleLogWriter(stderr, isTerminal)
	if cfg.Logstash.Addr == "" {
		return steplog.NewWithService(consoleOut, cfg.AppName), consoleOut, nil, nil
	}

	sink := steplog.NewTCPSink(
		cfg.Logstash.Addr,
		cfg.Logstash.BufferSize,
		cfg.Logstash.DialTimeout,
		warnOut,
	)

	out := io.MultiWriter(consoleOut, sink)
	return steplog.NewWithService(out, cfg.AppName), out, sink, nil
}

func consoleLogWriter(stderr io.Writer, isTerminal func(io.Writer) bool) io.Writer {
	if isTerminal == nil || !isTerminal(stderr) {
		return stderr
	}

	return steplog.NewConsoleColorWriter(stderr, true)
}

func isTerminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}

	stat, err := file.Stat()
	if err != nil {
		return false
	}

	return stat.Mode()&os.ModeCharDevice != 0
}
