package rewriter

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/goriller/ginny/errs"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
)

const (
	fallback             = `{"code": %d,"message":"%s"}`
	responseStatusHeader = "x-response-status"
)

// BodyReWriterFunc
type BodyReWriterFunc func(http.ResponseWriter, []byte, *status.Status) (int, error)

// ResponseWriter
type ResponseWriter struct {
	Status            *status.Status
	Writer            http.ResponseWriter
	HeaderStatus      int
	BodyWriter        BodyReWriterFunc
	WithoutHTTPStatus bool
}

// Header implement responseWriter
func (l *ResponseWriter) Header() http.Header {
	return l.Writer.Header()
}

// Write implement responseWrite
func (l *ResponseWriter) Write(b []byte) (i int, err error) {
	if l.BodyWriter != nil {
		i, err = l.BodyWriter(l.Writer, b, l.Status)
	} else {
		i, err = l.Writer.Write(b)
	}
	if err != nil {
		fallbackFunc(l, l.Status.Code(), l.Status.Message(), l.WithoutHTTPStatus)
	}
	return i, nil
}

// WriteHeader WriteHeader
func (l *ResponseWriter) WriteHeader(s int) {
	l.HeaderStatus = s
	l.Writer.Header().Set(responseStatusHeader, fmt.Sprintf("%v", s))
	if !l.WithoutHTTPStatus {
		l.Writer.WriteHeader(s)
	}
}

// DefaultBodyWriter
func DefaultBodyWriter(bodyMarshaler, errorMarshaler runtime.Marshaler, withoutHTTPStatus bool) BodyReWriterFunc {
	return func(w http.ResponseWriter, orgBody []byte, s *status.Status) (int, error) {
		var msg = s.Message()
		if msg == "" {
			msg = errs.CodeToError(s.Code())
		}
		statusBody := &MuxError{
			Code:    uint32(s.Code()),
			Message: msg,
		}
		var marshal = bodyMarshaler
		if s.Code() != codes.OK {
			marshal = errorMarshaler
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
		if !withoutHTTPStatus {
			w.WriteHeader(errs.CodeToStatus(s.Code()))
		}
		return w.Write(resBody)
	}
}

// fallbackFunc
func fallbackFunc(w http.ResponseWriter, code codes.Code, msg string, withoutHTTPStatus bool) {
	if code == 0 {
		code = 13
	}
	if msg == "" {
		msg = "internal error"
	}
	if !withoutHTTPStatus {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Header().Set(responseStatusHeader, fmt.Sprintf("%v", code))

	if _, err := io.WriteString(w, fmt.Sprintf(fallback, code, msg)); err != nil {
		grpclog.Infof("Failed to write response: %v", err)
	}
}

// WriteHTTPErrorResponse  set HTTP status code and write error description to the body.
func WriteHTTPErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	s, ok := status.FromError(err)
	if !ok {
		s = status.New(codes.Unknown, err.Error())
	}
	w.Header().Set("Content-Type", "application/json")
	preTags := grpc_logging. //tags.Extract(r.Context())
					preTags.Set("code", strconv.Itoa(int(s.Code())))
	preTags.Set("message", s.Message())

	if wt, ok := w.(*ResponseWriter); ok {
		wt.Status = s
		wt.WriteHeader(errs.CodeToStatus(s.Code()))
		_, err = wt.Write(nil)
		if err != nil {
			fallbackFunc(w, s.Code(), s.Message(), wt.WithoutHTTPStatus)
		}
		return
	}
	fallbackFunc(w, s.Code(), s.Message(), true)
}
