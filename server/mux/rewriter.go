package mux

import (
	"fmt"
	"net/http"

	"github.com/gorillazer/ginny/logging"
	"google.golang.org/grpc/status"
)

// bodyReWriterFunc
type bodyReWriterFunc func(http.ResponseWriter, *http.Request, *status.Status) (int, error)

// responseWriter
type responseWriter struct {
	w                 http.ResponseWriter
	header            int
	withoutHTTPStatus bool
}

// Header implement responseWriter
func (l *responseWriter) Header() http.Header {
	return l.w.Header()
}

// Write implement responseWrite
func (l *responseWriter) Write(b []byte) (int, error) {
	return l.w.Write(b)
}

// WriteHeader WriteHeader
func (l *responseWriter) WriteHeader(s int) {
	l.header = s
	l.w.Header().Set(logging.ResponseStatusHeader, fmt.Sprintf("%v", s))
	if !l.withoutHTTPStatus {
		l.w.WriteHeader(s)
	}
}

// defaultBodyWriter
func defaultBodyWriter(w http.ResponseWriter, r *http.Request, s *status.Status) (int, error) {
	var msg = s.Message()
	if msg == "" {
		msg = CodeToError(s.Code())
	}
	statusError := &MuxError{
		Code:    uint32(s.Code()),
		Message: msg,
	}

	buf, err := defaultMuxOption.errorMarshaler.Marshal(statusError)
	if err != nil {
		return 0, err
	}
	return w.Write(buf)
}
