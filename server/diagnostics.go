package server

import (
	"log"
	"net/http"
	"sort"
)

var diagnosticHeaders = []string{
	"User-Agent",
	"Origin",
	"Host",
	"X-Forwarded-For",
	"X-Forwarded-Proto",
	"X-Forwarded-Host",
	"CF-Connecting-IP",
}

func (server *Server) debugf(format string, v ...interface{}) {
	if server.options != nil && server.options.Debug {
		log.Printf("[debug] "+format, v...)
	}
}

func (server *Server) debugHTTPRequest(event string, r *http.Request) {
	if server.options == nil || !server.options.Debug {
		return
	}

	server.debugf(
		"%s method=%s path=%s raw_query=%q remote=%s host=%s proto=%s tls=%t",
		event,
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		r.RemoteAddr,
		r.Host,
		r.Proto,
		r.TLS != nil,
	)

	for _, name := range diagnosticHeaders {
		if value := r.Header.Get(name); value != "" {
			server.debugf("%s header %s=%q", event, name, value)
		}
	}
}

func (server *Server) debugWSEvent(remoteAddr string, event string, fields map[string]interface{}) {
	if server.options == nil || !server.options.Debug {
		return
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	args := []interface{}{event, remoteAddr}
	format := "ws %s remote=%s"
	for _, key := range keys {
		format += " " + key + "=%v"
		args = append(args, fields[key])
	}
	server.debugf(format, args...)
}
