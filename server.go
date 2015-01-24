package engineio

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"github.com/googollee/go-engine.io/polling"
	"github.com/googollee/go-engine.io/websocket"
	"net/http"
	"time"
)

type config struct {
	PingTimeout   time.Duration
	PingInterval  time.Duration
	AllowRequest  func(*http.Request) error
	AllowUpgrades bool
	Cookie        string
	NewId         func(r *http.Request) string
}

// Server is the server of engine.io.
type Server struct {
	config         config
	socketChan     chan Conn
	serverSessions *serverSessions
	creaters       transportCreaters
}

// NewServer returns the server suppported given transports. If transports is nil, server will use ["polling", "webosocket"] as default.
func NewServer(transports []string) (*Server, error) {
	if transports == nil {
		transports = []string{"polling", "websocket"}
	}
	creaters := make(transportCreaters)
	for _, t := range transports {
		switch t {
		case "polling":
			creaters[t] = polling.Creater
		case "websocket":
			creaters[t] = websocket.Creater
		default:
			return nil, InvalidError
		}
	}
	return &Server{
		config: config{
			PingTimeout:   60000 * time.Millisecond,
			PingInterval:  25000 * time.Millisecond,
			AllowRequest:  func(*http.Request) error { return nil },
			AllowUpgrades: true,
			Cookie:        "io",
			NewId:  nil,
		},
		socketChan:     make(chan Conn),
		serverSessions: newServerSessions(),
		creaters:       creaters,
	}, nil
}

// SetPingTimeout sets the timeout of ping. When time out, server will close connection. Default is 60s.
func (s *Server) SetPingTimeout(t time.Duration) {
	s.config.PingTimeout = t
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

// SetNewId sets the callback func to generate new connection id. By default, id is generated from remote addr + current time stamp
func (s *Server) SetNewId(f func(*http.Request) string) {
	s.config.NewId = f
}

// ServeHTTP handles http request.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	cookies := r.Cookies()
	sid := r.URL.Query().Get("sid")
	conn := s.serverSessions.Get(sid)
	if conn == nil {
		if sid != "" {
			http.Error(w, "invalid sid", http.StatusBadRequest)
			return
		}

		if err := s.config.AllowRequest(r); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if s.config.NewId != nil {
			sid = s.config.NewId(r)
		} else {
			sid = s.newId(r)
		}
		var err error
		conn, err = newServerConn(sid, w, r, s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		s.serverSessions.Set(sid, conn)
		cookies = append(cookies, &http.Cookie{
			Name:  s.config.Cookie,
			Value: sid,
		})

		s.socketChan <- conn
	}
	for _, c := range cookies {
		w.Header().Set("Set-Cookie", c.String())
	}
	conn.ServeHTTP(w, r)
}

// Accept returns Conn when client connect to server.
func (s *Server) Accept() (Conn, error) {
	return <-s.socketChan, nil
}

func (s *Server) configure() config {
	return s.config
}

func (s *Server) transports() transportCreaters {
	return s.creaters
}

func (s *Server) onClose(id string) {
	s.serverSessions.Remove(id)
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
