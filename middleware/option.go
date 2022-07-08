package middleware

import "net/http"

// MuxMiddleware
type MuxMiddleware func(http.Handler) http.Handler
