package errs

import (
	"net/http"
	"strconv"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/codes"
)

const (
	statusOKPrefix         = 2
	statusBadRequestPrefix = 4
)

// codesErrors some errors string for grpc codes
var codesErrors = map[codes.Code]string{
	codes.OK:                 "ok",
	codes.Canceled:           "canceled",
	codes.Unknown:            "unknown",
	codes.InvalidArgument:    "invalid_argument",
	codes.DeadlineExceeded:   "deadline_exceeded",
	codes.NotFound:           "not_found",
	codes.AlreadyExists:      "already_exists",
	codes.PermissionDenied:   "permission_denied",
	codes.ResourceExhausted:  "resource_exhausted",
	codes.FailedPrecondition: "failed_precondition",
	codes.Aborted:            "aborted",
	codes.OutOfRange:         "out_of_range",
	codes.Unimplemented:      "unimplemented",
	codes.Internal:           "internal",
	codes.Unavailable:        "unavailable",
	codes.DataLoss:           "data_loss",
	codes.Unauthenticated:    "unauthenticated",
}

// RegisterErrorCodes set custom error codes for DefaultHTTPError
// for exp: server.RegisterErrorCodes(pb.ErrorCode_name)
// SetCustomErrorCodes set custom error codes for DefaultHTTPError
// the map[int32]string is compact to protobuf's ENMU_name
// 2*** HTTP status 200
// 4*** HTTP status 400
// 5*** AND other HTTP status 500
// For exp:
// in proto
//
//	enum CommonError {
//		captcha_required = 4001;
//		invalid_captcha = 4002;
//	}
//
// in code
// server.RegisterErrorCodes(common.CommonError_name)
func RegisterErrorCodes(codeErrors map[int32]string) {
	for code, errorMsg := range codeErrors {
		codesErrors[codes.Code(code)] = errorMsg
	}
}

// httpStatusCode the 2xxx is 200, the 4xxx is 400, the 5xxx is 500
func httpStatusCode(code codes.Code) (httpStatusCode int) {
	// http status codes can be error codes
	if code >= 200 && code < 599 {
		return int(code)
	}
	for code >= 10 {
		code /= 10
	}
	switch code {
	case statusOKPrefix:
		httpStatusCode = http.StatusOK
	case statusBadRequestPrefix:
		httpStatusCode = http.StatusBadRequest
	default:
		httpStatusCode = http.StatusInternalServerError
	}
	return
}

// CodeToError
func CodeToError(c codes.Code) string {
	errStr, ok := codesErrors[c]
	if ok {
		return errStr
	}
	return strconv.FormatInt(int64(c), 10)
}

// CodeToStatus
func CodeToStatus(code codes.Code) int {
	st := int(code)
	if st > 100 {
		st = httpStatusCode(code)
	} else {
		st = runtime.HTTPStatusFromCode(code)
	}
	return st
}
