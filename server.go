package engineio

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

type config struct {
	PingTimeout   time.Duration
	PingInterval  time.Duration
	AllowRequest  func(*http.Request) error
	AllowUpgrades bool
	Cookie        string
}

// Server is the server of engine.io.
type Server struct {
	config     config
	socketChan chan Conn
	sessions   *sessions
	transports transportsType
}

// NewServer returns the server suppported given transports. If transports is nil, server will support all kinds of transports.
func NewServer(transports []string) (*Server, error) {
	t, err := newTransportsType(transports)
	if err != nil {
		return nil, err
	}
	return &Server{
		config: config{
			PingTimeout:   60000 * time.Millisecond,
			PingInterval:  25000 * time.Millisecond,
			AllowRequest:  func(*http.Request) error { return nil },
			AllowUpgrades: true,
			Cookie:        "io",
		},
		socketChan: make(chan Conn),
		sessions:   newSessions(),
		transports: t,
	}, nil
}

// SetPingTimeout sets the timeout of ping. When time out, server will close connection. Default is 60s.
func (s *Server) SetPingTimeout(t time.Duration) {
	s.config.PingInterval = t
}

// SetPingInterval sets the interval of ping. Default is 25s.
func (s *Server) SetPingInterval(t time.Duration) {
	s.config.PingInterval = t
}

// SetAllowRequest sets the middleware function when establish connection. If it return non-nil, connection won't be established. Default will allow all request.
func (s *Server) SetAllowRequest(f func(*http.Request) error) {
	s.config.AllowRequest = f
}

// SetAllowUpgrades sets whether server allows transport upgrade. Default is true.
func (s *Server) SetAllowUpgrades(allow bool) {
	s.config.AllowUpgrades = allow
}

// SetCookie sets the name of cookie which used by engine.io. Default is "io".
func (s *Server) SetCookie(prefix string) {
	s.config.Cookie = prefix
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
		transportCreater := s.transports.GetCreater(transportName)
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
