package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestGetEnv(t *testing.T) {
	t.Setenv("POD_RESOURCES_SET", "from-environment")

	if got := GetEnv("POD_RESOURCES_SET", "fallback", Logger()); got != "from-environment" {
		t.Errorf("got %q, want the value from the environment", got)
	}
	if got := GetEnv("POD_RESOURCES_UNSET", "fallback", Logger()); got != "fallback" {
		t.Errorf("got %q, want the default", got)
	}
}

func TestGetEnvKeepsAnEmptyValue(t *testing.T) {
	t.Setenv("POD_RESOURCES_EMPTY", "")

	if got := GetEnv("POD_RESOURCES_EMPTY", "fallback", Logger()); got != "" {
		t.Errorf("got %q, want the empty value that was set rather than the default", got)
	}
}

func TestLogWithFieldsSplitsOnTheFirstColon(t *testing.T) {
	entry := captureEntry(t, func() {
		LogWithFields(logrus.InfoLevel, []string{
			"container: web",
			"namespace:shop",
			"message: usage 500m, above its limit",
			"no colon at all",
		}, "Resource(s) need adjusting")
	})

	if entry["msg"] != "Resource(s) need adjusting" {
		t.Errorf("msg is %v, want the message passed in", entry["msg"])
	}
	if entry["container"] != "web" {
		t.Errorf("container is %v, want web with the space trimmed", entry["container"])
	}
	if entry["namespace"] != "shop" {
		t.Errorf("namespace is %v, want shop", entry["namespace"])
	}
	if entry["message"] != "usage 500m, above its limit" {
		t.Errorf("message is %v, want everything after the first colon", entry["message"])
	}
	if _, exists := entry["no colon at all"]; exists {
		t.Error("a field with no colon became a key")
	}
}

func TestLogWithFieldsRecordsErrors(t *testing.T) {
	entry := captureEntry(t, func() {
		LogWithFields(logrus.ErrorLevel, nil, "Cannot get pods", errors.New("connection refused"))
	})

	if entry["level"] != "error" {
		t.Errorf("level is %v, want error", entry["level"])
	}
	if entry["error"] == nil {
		t.Error("the error was not recorded on the entry")
	}
}

func TestLoggerIsASingleton(t *testing.T) {
	if Logger() != Logger() {
		t.Error("Logger returned two different instances")
	}
}

func captureEntry(t *testing.T, log func()) map[string]any {
	t.Helper()

	var buf bytes.Buffer
	logger := Logger()
	logger.SetOutput(&buf)
	t.Cleanup(func() { logger.SetOutput(os.Stderr) })

	log()

	entry := map[string]any{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log line %q is not JSON: %v", buf.String(), err)
	}
	return entry
}
