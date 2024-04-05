package log

import "go.uber.org/zap"

type Logger interface {
	Warnf(template string, args ...interface{})
	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Fatalf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
}

func NewDevZapLogger() *zap.SugaredLogger {
	return zap.Must(zap.NewDevelopment()).Sugar()
}

func NewProdZapLogger() *zap.SugaredLogger {
	return zap.Must(zap.NewProduction()).Sugar()
}

type DummyLogger struct {
}

func (l DummyLogger) Warnf(template string, args ...interface{}) {

}

func (l DummyLogger) Debugf(template string, args ...interface{}) {

}

func (l DummyLogger) Infof(template string, args ...interface{}) {

}

func (l DummyLogger) Fatalf(template string, args ...interface{}) {

}

func (l DummyLogger) Errorf(template string, args ...interface{}) {

}

func NewDummyLogger() DummyLogger {
	return DummyLogger{}
}
