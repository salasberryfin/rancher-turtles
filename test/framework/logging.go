//go:build e2e
// +build e2e

/*
Copyright © 2023 - 2024 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
)

// TestLogger provides structured logging for E2E tests with context propagation.
// It allows setting context that will be included in all log messages.
type TestLogger struct {
	testName  string
	suiteName string
	context   map[string]interface{}
	startTime time.Time
}

// NewTestLogger creates a new TestLogger for the given test.
func NewTestLogger(suiteName, testName string) *TestLogger {
	return &TestLogger{
		testName:  testName,
		suiteName: suiteName,
		context:   make(map[string]interface{}),
		startTime: time.Now().UTC(),
	}
}

// WithContext returns a copy of the logger with an additional context key-value pair.
func (l *TestLogger) WithContext(key string, value interface{}) *TestLogger {
	newLogger := &TestLogger{
		testName:  l.testName,
		suiteName: l.suiteName,
		context:   make(map[string]interface{}),
		startTime: l.startTime,
	}
	for k, v := range l.context {
		newLogger.context[k] = v
	}
	newLogger.context[key] = value
	return newLogger
}

// WithCluster returns a copy of the logger with cluster context set.
func (l *TestLogger) WithCluster(clusterName string) *TestLogger {
	return l.WithContext("cluster", clusterName)
}

// WithNamespace returns a copy of the logger with namespace context set.
func (l *TestLogger) WithNamespace(namespace string) *TestLogger {
	return l.WithContext("namespace", namespace)
}

// WithProvider returns a copy of the logger with provider context set.
func (l *TestLogger) WithProvider(provider string) *TestLogger {
	return l.WithContext("provider", provider)
}

// Step logs a test step with structured context.
func (l *TestLogger) Step(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	elapsed := time.Since(l.startTime).Round(time.Second)
	l.logWithContext("STEP", message, elapsed)
	By(message)
}

// Info logs an informational message with structured context.
func (l *TestLogger) Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	elapsed := time.Since(l.startTime).Round(time.Second)
	l.logWithContext("INFO", message, elapsed)
}

// Debug logs a debug message with structured context.
func (l *TestLogger) Debug(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	elapsed := time.Since(l.startTime).Round(time.Second)
	l.logWithContext("DEBUG", message, elapsed)
}

// Error logs an error message with structured context.
func (l *TestLogger) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	elapsed := time.Since(l.startTime).Round(time.Second)
	l.logWithContext("ERROR", message, elapsed)
}

// logWithContext formats and outputs the log message with all context.
func (l *TestLogger) logWithContext(level, message string, elapsed time.Duration) {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	logEntry := struct {
		Timestamp string                 `json:"ts"`
		Level     string                 `json:"level"`
		Suite     string                 `json:"suite"`
		Test      string                 `json:"test"`
		Elapsed   string                 `json:"elapsed"`
		Message   string                 `json:"msg"`
		Context   map[string]interface{} `json:"context,omitempty"`
	}{
		Timestamp: timestamp,
		Level:     level,
		Suite:     l.suiteName,
		Test:      l.testName,
		Elapsed:   elapsed.String(),
		Message:   message,
		Context:   l.context,
	}

	jsonBytes, err := json.Marshal(logEntry)
	if err != nil {
		GinkgoWriter.Printf("[%s] [%s] %s: %s (context marshal error: %v)\n",
			timestamp, level, l.testName, message, err)
		return
	}

	GinkgoWriter.Printf("%s\n", string(jsonBytes))
}

// StartTimer returns a function that, when called, logs the duration since Start was called.
// Useful for timing operations.
func (l *TestLogger) StartTimer(operationName string) func() {
	start := time.Now()
	l.Info("Starting operation: %s", operationName)
	return func() {
		duration := time.Since(start).Round(time.Millisecond)
		l.Info("Completed operation: %s (took %s)", operationName, duration)
	}
}

// LoggedStep executes a step with automatic timing and logging.
func (l *TestLogger) LoggedStep(stepName string, fn func()) {
	done := l.StartTimer(stepName)
	defer done()
	l.Step("%s", stepName)
	fn()
}

// GetElapsed returns the duration since the logger was created.
func (l *TestLogger) GetElapsed() time.Duration {
	return time.Since(l.startTime)
}
