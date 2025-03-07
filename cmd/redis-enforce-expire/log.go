package main

import (
	"fmt"
	"log/slog"
	"os"
)

func infof(format string, a ...any) {
	slog.Info(fmt.Sprintf(format, a...))
}

func errorf(format string, a ...any) {
	slog.Error(fmt.Sprintf(format, a...))
}

func fatalf(format string, a ...any) {
	slog.Error("FATAL: " + fmt.Sprintf(format, a...))
	os.Exit(1)
}
