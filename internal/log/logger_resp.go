package log

import "net/http"

type ResponseData struct {
	Status int
	Size   uint64
}

type LoggingRepsonseWriter struct {
	http.ResponseWriter
	Data *ResponseData
}

func (l *LoggingRepsonseWriter) WriteHeader(statusCode int) {
	l.Data.Status = statusCode
	l.ResponseWriter.WriteHeader(statusCode)
}

func (l *LoggingRepsonseWriter) Write(b []byte) (int, error) {
	size, err := l.ResponseWriter.Write(b)
	l.Data.Size += uint64(size)
	return size, err
}
