package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	limiter "github.com/ulule/limiter/v3"
	stdlib "github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	memory "github.com/ulule/limiter/v3/drivers/store/memory"
)

// limiter middleware pointer
var limiterMiddleware *stdlib.Middleware

// initialize Limiter middleware pointer
func initLimiter(period string) {
	log.Printf("limiter rate='%s'", period)
	// create rate limiter with 5 req/second
	rate, err := limiter.NewRateFromFormatted(period)
	if err != nil {
		panic(err)
	}
	store := memory.NewStore()
	instance := limiter.New(store, rate)
	limiterMiddleware = stdlib.NewMiddleware(instance)
}

/*
// helper to auth/authz incoming requests to the server
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// perform authentication
		status := CMSAuth.CheckAuthnAuthz(r.Header)
		if !status {
			log.Printf("ERROR: fail to authenticate, HTTP headers %+v\n", r.Header)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if Config.Verbose > 2 {
			log.Printf("Auth layer status: %v headers: %+v\n", status, r.Header)
		}
		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
*/

// Validate should implement input validation
func Validate(r *http.Request) error {
	return nil
}

// helper to validate incoming requests' parameters
func validateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			next.ServeHTTP(w, r)
			return
		}
		// perform validation of input parameters
		err := Validate(r)
		if err != nil {
			uri, _ := url.QueryUnescape(r.RequestURI)
			log.Printf("HTTP %s %s validation error %v\n", r.Method, uri, err)
			w.WriteHeader(http.StatusBadRequest)
			rec := make(map[string]string)
			rec["error"] = fmt.Sprintf("Validation error %v", err)
			if r, e := json.Marshal(rec); e == nil {
				w.Write(r)
			}
			return
		}
		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// limit middleware limits incoming requests
func limitMiddleware(next http.Handler) http.Handler {
	return limiterMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	}))
}

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the
// written HTTP status code to be captured for logging.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

// wrapper for response writer
// based on https://blog.questionable.services/article/guide-logging-middleware-go/
func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}

	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true

	return
}

// loggingMiddleware logs the incoming HTTP request and its duration.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "POST" {
			atomic.AddUint64(&TotalPostRequests, 1)
		} else if r.Method == "GET" {
			atomic.AddUint64(&TotalGetRequests, 1)
		} else if r.Method == "DELETE" {
			atomic.AddUint64(&TotalDeleteRequests, 1)
		}
		start := time.Now()
		tstamp := int64(start.UnixNano() / 1000000) // use milliseconds for MONIT

		wrapped := wrapResponseWriter(w)
		status := wrapped.status
		if status == 0 { // the status code was not set, i.e. everything is fine
			status = 200
		}
		next.ServeHTTP(wrapped, r)
		var dataSize int64
		logRequest(w, r, start, wrapped.status, tstamp, dataSize)
	})
}
