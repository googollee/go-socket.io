package engineio

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Config struct {
	PingTimeout       time.Duration
	PingInterval      time.Duration
	MaxHttpBufferSize int
	AllowRequest      func(*http.Request) (bool, error)
	Transports        []string
	AllowUpgrades     bool
	Cookie            string
}

var DefaultConfig = Config{
	PingTimeout:       60000 * time.Millisecond,
	PingInterval:      25000 * time.Millisecond,
	MaxHttpBufferSize: 0x10E7,
	AllowRequest:      func(*http.Request) (bool, error) { return true, nil },
	Transports:        []string{"polling", "websocket"},
	AllowUpgrades:     true,
	Cookie:            "io",
}

type Server struct {
	config     Config
	socketChan chan Socket
	sessions   map[string]*socket
}

func NewServer(conf Config) *Server {
	return &Server{
		config:     conf,
		socketChan: make(chan Socket),
		sessions:   make(map[string]*socket),
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	cookies := r.Cookies()
	var socket *socket
	sid := r.URL.Query().Get("sid")
	transportName := r.URL.Query().Get("transport")
	if sid == "" {
		transportCreater := getTransportCreater(transportName)
		if transportCreater == nil {
			http.Error(w, "invalid transport", http.StatusBadRequest)
			return
		}
		transport, err := transportCreater(r, s.config.PingInterval, s.config.PingTimeout)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		hash := fmt.Sprintf("%s %s", r.RemoteAddr, time.Now())
		buf := bytes.NewBuffer(nil)
		sum := md5.Sum([]byte(hash))
		encoder := base64.NewEncoder(base64.URLEncoding, buf)
		encoder.Write(sum[:])
		encoder.Close()
		sid = buf.String()[:20]
		socket = newSocket(sid, s, transport, r)
		if err := s.onOpen(socket); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		s.sessions[sid] = socket
		cookies = append(cookies, &http.Cookie{
			Name:  s.config.Cookie,
			Value: sid,
		})
		s.socketChan <- socket
	} else {
		var ok bool
		socket, ok = s.sessions[sid]
		if !ok {
			http.Error(w, "invalid sid", http.StatusBadRequest)
			return
		}
		if socket.transport().Name() != transportName && !socket.Upgraded() {
			creater := getTransportCreater(transportName)
			if creater == nil {
				http.Error(w, "invalid transport", http.StatusBadRequest)
				return
			}
			transport, err := creater(r, s.config.PingInterval, s.config.PingTimeout)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			socket.upgrade(transport)
		}
	}

	for _, c := range cookies {
		w.Header().Set("Set-Cookie", c.String())
	}
	socket.transport().ServeHTTP(w, r)
}

func (s *Server) Accept() (Socket, error) {
	return <-s.socketChan, nil
}

func (s *Server) onOpen(so *socket) error {
	resp := struct {
		Sid          string        `json:"sid"`
		Upgrades     []string      `json:"upgrades"`
		PingInterval time.Duration `json:"pingInterval"`
		PingTimeout  time.Duration `json:"pingTimeout"`
	}{
		Sid:          so.id,
		Upgrades:     getUpgradesHandlers(),
		PingInterval: s.config.PingInterval / time.Millisecond,
		PingTimeout:  s.config.PingTimeout / time.Millisecond,
	}
	w, err := so.transport().NextWriter(MessageText, OPEN)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(resp); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

func (s *Server) onClose(so *socket) {
	delete(s.sessions, so.id)
}
