package testutil

// NoopLogger implements the common.Logger interface without producing output.
// Duplicate minimal interface methods inline to avoid importing a heavy logger dependency.
// Adjust if common.Logger interface evolves.

type NoopLogger struct{}

func (NoopLogger) Debug(...interface{})          {}
func (NoopLogger) Info(...interface{})           {}
func (NoopLogger) Warn(...interface{})           {}
func (NoopLogger) Error(...interface{})          {}
func (NoopLogger) Debugf(string, ...interface{}) {}
func (NoopLogger) Infof(string, ...interface{})  {}
func (NoopLogger) Warnf(string, ...interface{})  {}
func (NoopLogger) Errorf(string, ...interface{}) {}
