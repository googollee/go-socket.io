package engineio

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

// Config is the configuration of engine.io server.
type Config struct {
	// PingTimeout is the timeout of ping. When time out, server will close connection.
	PingTimeout time.Duration
	// PingInterval is the interval of ping.
	PingInterval time.Duration
	// AllowRequest is middleware when establish connection. If it return non-nil, connection won't be established.
	AllowRequest func(*http.Request) error
	// Transports are the list of supported transport.
	Transports []string
	// AllowUpgrades specify whether server allows transport upgrade.
	AllowUpgrades bool
	// Cookie is the name of cookie which used by engine.io.
	Cookie string
	// MaxHttpBufferSize int
}

// DefaultConfig is the default configuration.
var DefaultConfig = Config{
	PingTimeout:   60000 * time.Millisecond,
	PingInterval:  25000 * time.Millisecond,
	AllowRequest:  func(*http.Request) error { return nil },
	Transports:    []string{"polling", "websocket"},
	AllowUpgrades: true,
	Cookie:        "io",
	// MaxHttpBufferSize: 0x10E7,
}

// Server is the server of engine.io.
type Server struct {
	config     Config
	socketChan chan Conn
	sessions   *sessions
}

// NewServer returns the server.
func NewServer(conf Config) *Server {
	return &Server{
		config:     conf,
		socketChan: make(chan Conn),
		sessions:   newSessions(),
	}
}

// ServeHTTP handles http request.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	cookies := r.Cookies()
	sid := r.URL.Query().Get("sid")
	if sid == "" {
		if err := s.config.AllowRequest(r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		transportName := r.URL.Query().Get("transport")
		transportCreater := transports.GetCreater(transportName)
		if transportCreater == nil {
			http.Error(w, "invalid transport", http.StatusBadRequest)
			return
		}
		transport, err := transportCreater(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sid = s.newId(r)
		conn, err := newSocket(sid, s, transport, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		s.sessions.Set(sid, conn)
		cookies = append(cookies, &http.Cookie{
			Name:  s.config.Cookie,
			Value: sid,
		})
		s.socketChan <- conn
	}
	conn := s.sessions.Get(sid)
	if conn == nil {
		http.Error(w, "invalid sid", http.StatusBadRequest)
		return
	}

	for _, c := range cookies {
		w.Header().Set("Set-Cookie", c.String())
	}
	conn.serveHTTP(w, r)
}

// Accept returns Conn when client connect to server.
func (s *Server) Accept() (Conn, error) {
	return <-s.socketChan, nil
}

func (s *Server) onClose(so *conn) {
	s.sessions.Remove(so.Id())
}

func (s *Server) newId(r *http.Request) string {
	hash := fmt.Sprintf("%s %s", r.RemoteAddr, time.Now())
	buf := bytes.NewBuffer(nil)
	sum := md5.Sum([]byte(hash))
	encoder := base64.NewEncoder(base64.URLEncoding, buf)
	encoder.Write(sum[:])
	encoder.Close()
	return buf.String()[:20]
}
