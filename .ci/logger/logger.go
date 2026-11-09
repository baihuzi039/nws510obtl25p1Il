package logger

type SugaredLoggerT struct{}

var SugaredLogger SugaredLoggerT

func (SugaredLoggerT) Info(args ...interface{})  {}
func (SugaredLoggerT) Warn(args ...interface{})  {}
func (SugaredLoggerT) Error(args ...interface{}) {}
func (SugaredLoggerT) Debug(args ...interface{}) {}
func (SugaredLoggerT) Infof(format string, args ...interface{})  {}
func (SugaredLoggerT) Warnf(format string, args ...interface{})  {}
func (SugaredLoggerT) Errorf(format string, args ...interface{}) {}
func (SugaredLoggerT) Debugf(format string, args ...interface{}) {}
