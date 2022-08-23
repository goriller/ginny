package mux

import (
	"fmt"
	"io"
	"net/http"

	"github.com/goriller/ginny/interceptor/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
)

// bodyReWriterFunc
type bodyReWriterFunc func(http.ResponseWriter, []byte, *status.Status) (int, error)

// responseWriter
type responseWriter struct {
	s                 *status.Status
	w                 http.ResponseWriter
	header            int
	withoutHTTPStatus bool
}

// Header implement responseWriter
func (l *responseWriter) Header() http.Header {
	return l.w.Header()
}

// Write implement responseWrite
func (l *responseWriter) Write(b []byte) (i int, err error) {
	if defaultMuxOption.bodyWriter != nil {
		i, err = defaultMuxOption.bodyWriter(l.w, b, l.s)
	} else {
		i, err = l.w.Write(b)
	}
	if err != nil {
		fallbackFunc(l, l.s.Code(), l.s.Message())
	}
	return i, nil
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
func defaultBodyWriter(w http.ResponseWriter, orgBody []byte, s *status.Status) (int, error) {
	var msg = s.Message()
	if msg == "" {
		msg = CodeToError(s.Code())
	}
	statusBody := &MuxError{
		Code:    uint32(s.Code()),
		Message: msg,
	}
	var marshal = defaultMuxOption.bodyMarshaler
	if s.Code() != codes.OK {
		marshal = defaultMuxOption.errorMarshaler
	}

	body, err := marshal.Marshal(statusBody)
	if err != nil {
		return 0, err
	}
	var resBody []byte
	if orgBody != nil {
		resBody = body[:len(body)-1]
		if len(orgBody) > 2 {
			resBody = append(resBody, ',')
		}
		resBody = append(resBody, []byte(`"data":`)...)
		resBody = append(resBody, orgBody[:len(orgBody)-1]...)
		resBody = append(resBody, []byte(`}}`)...)
	} else {
		resBody = body

	}
	if !defaultMuxOption.withoutHTTPStatus {
		w.WriteHeader(CodeToStatus(s.Code()))
	}
	return w.Write(resBody)
}

// fallbackFunc
func fallbackFunc(w http.ResponseWriter, code codes.Code, msg string) {
	if code == 0 {
		code = 13
	}
	if msg == "" {
		msg = "internal error"
	}
	if !defaultMuxOption.withoutHTTPStatus {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Header().Set(logging.ResponseStatusHeader, fmt.Sprintf("%v", code))

	if _, err := io.WriteString(w, fmt.Sprintf(fallback, code, msg)); err != nil {
		grpclog.Infof("Failed to write response: %v", err)
	}
}
