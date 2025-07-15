// Copyright (c) 2025 True Tickets, Inc.
// SPDX-License-Identifier: MIT

package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// loggingMiddleware logs requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code and response size
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// Create log event with appropriate level based on status code
		var logEvent *zerolog.Event
		if rw.statusCode >= http.StatusBadRequest {
			logEvent = log.Warn()
		} else {
			logEvent = log.Info()
		}

		logEvent.
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("query", r.URL.RawQuery).
			Int("status", rw.statusCode).
			Dur("duration", duration).
			Int64("response_size", rw.responseSize).
			Str("user_agent", r.UserAgent()).
			Str("remote_addr", r.RemoteAddr).
			Str("referer", r.Referer()).
			Str("content_type", r.Header.Get("Content-Type")).
			Int64("content_length", r.ContentLength).
			Msg("HTTP request")
	})
}

// tracingMiddleware adds tracing to requests
func (s *Server) tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := s.tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		defer span.End()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// compressionMiddleware adds gzip compression to responses
func (s *Server) compressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip encoding
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Create gzip response writer
		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			minSize:        compressionMinSize,
		}
		defer func() {
			if err := gzw.Close(); err != nil {
				s.logger.Warn().Err(err).Msg("Failed to close gzip writer")
			}
		}()

		// Serve with gzip writer
		next.ServeHTTP(gzw, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(data)
	rw.responseSize += int64(n)
	return n, err
}

// gzipResponseWriter wraps http.ResponseWriter to provide gzip compression
type gzipResponseWriter struct {
	http.ResponseWriter
	writer      *gzip.Writer
	minSize     int
	buf         bytes.Buffer
	statusCode  int
	headersDone bool
}

// WriteHeader captures the status code and delays header writing
func (gzw *gzipResponseWriter) WriteHeader(statusCode int) {
	gzw.statusCode = statusCode
}

// Write buffers data and compresses if size threshold is met
func (gzw *gzipResponseWriter) Write(data []byte) (int, error) {
	// Buffer the data
	n, err := gzw.buf.Write(data)
	if err != nil {
		return n, err
	}

	// Check if we've reached the minimum size for compression
	if gzw.buf.Len() >= gzw.minSize && gzw.writer == nil {
		// Initialize gzip writer
		gzw.writer = gzip.NewWriter(gzw.ResponseWriter)
		gzw.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		gzw.ResponseWriter.Header().Del("Content-Length") // Content-Length is not valid with compression
		gzw.ResponseWriter.Header().Set("Vary", "Accept-Encoding")

		// Write status code
		if gzw.statusCode != 0 {
			gzw.ResponseWriter.WriteHeader(gzw.statusCode)
		}
		gzw.headersDone = true

		// Write buffered data to gzip writer
		if _, err := gzw.writer.Write(gzw.buf.Bytes()); err != nil {
			return n, err
		}
		gzw.buf.Reset()
		return n, nil
	}

	// If gzip writer is already initialized, write directly to it
	if gzw.writer != nil {
		if _, err := gzw.writer.Write(gzw.buf.Bytes()); err != nil {
			return n, err
		}
		gzw.buf.Reset()
	}

	return n, nil
}

// Close flushes any remaining data
func (gzw *gzipResponseWriter) Close() error {
	// If we have buffered data but haven't started compression
	if gzw.buf.Len() > 0 && gzw.writer == nil {
		// Write headers if not done
		if !gzw.headersDone {
			if gzw.statusCode != 0 {
				gzw.ResponseWriter.WriteHeader(gzw.statusCode)
			}
		}
		// Write uncompressed data
		_, err := gzw.ResponseWriter.Write(gzw.buf.Bytes())
		return err
	}

	// If gzip writer exists, close it
	if gzw.writer != nil {
		return gzw.writer.Close()
	}

	return nil
}
