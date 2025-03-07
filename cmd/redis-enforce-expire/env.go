package main

import "os"

// envString extracts string from env var.
// It returns the provided defaultValue if the env var is empty.
// The value returned is also recorded in logs.
func envString(name string, defaultValue string) string {
	str := os.Getenv(name)
	if str != "" {
		infof("%s=[%s] using %s=%s default=%s", name, str, name, str, defaultValue)
		return str
	}
	infof("%s=[%s] using %s=%s default=%s", name, str, name, defaultValue, defaultValue)
	return defaultValue
}
