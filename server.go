package socketio

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

var (
	uriRegexp = regexp.MustCompile(`^(.+?)/(1)(?:/([^/]+)/([^/]+))?/?$`)
)

type Config struct {
	HeartbeatTimeout int
	ClosingTimeout   int
	NewSessionID     func() string
	Transports       *TransportManager
	Authorize        func(*http.Request) bool
}

type SocketIOServer struct {
	mutex            sync.RWMutex
	heartbeatTimeout int
	closingTimeout   int
	authorize        func(*http.Request) bool
	newSessionId     func() string
	transports       *TransportManager
	sessions         map[string]*Session
	eventEmitters    map[string]*EventEmitter
}

func NewSocketIOServer(config *Config) *SocketIOServer {
	server := new(SocketIOServer)
	if config != nil {
		server.heartbeatTimeout = config.HeartbeatTimeout
		server.closingTimeout = config.ClosingTimeout
		server.newSessionId = config.NewSessionID
		server.transports = config.Transports
		server.authorize = config.Authorize
	}
	if server.heartbeatTimeout == 0 {
		server.heartbeatTimeout = 15000
	}
	if server.closingTimeout == 0 {
		server.closingTimeout = 10000
	}
	if server.newSessionId == nil {
		server.newSessionId = NewSessionID
	}
	if server.transports == nil {
		server.transports = DefaultTransports
	}
	server.sessions = make(map[string]*Session)
	server.eventEmitters = make(map[string]*EventEmitter)
	return server
}

func (srv *SocketIOServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	pieces := uriRegexp.FindStringSubmatch(path)
	if pieces == nil {
		w.WriteHeader(404)
		fmt.Fprintln(w, "invalid uri: %s", r.URL)
		return
	}
	sessionId := pieces[4]
	session := srv.getSession(sessionId)
	if session == nil {
		http.Error(w, "invalid session id", 400)
		return
	}
	defer srv.removeSession(session)

	transportId := pieces[3]
	// connect
	if transportId == "" { // imply session==""
		srv.handShake(w, r)
		return
	}
	// open
	transport := srv.transports.Get(transportId, session, srv.heartbeatTimeout)
	if transport == nil {
		http.Error(w, "transport not supported", 400)
		return
	}

	session.transport = transport
	ns := session.Of("")
	ns.emit("connect", ns, nil)

	transport.OnData(w, r)
}

func (srv *SocketIOServer) Of(name string) *EventEmitter {
	ret, ok := srv.eventEmitters[name]
	if !ok {
		ret = NewEventEmitter()
		srv.eventEmitters[name] = ret
	}
	return ret
}

func (srv *SocketIOServer) On(name string, fn interface{}) error {
	return srv.Of("").On(name, fn)
}

func (srv *SocketIOServer) RemoveListener(name string, fn interface{}) {
	srv.Of("").RemoveListener(name, fn)
}

func (srv *SocketIOServer) RemoveAllListeners(name string) {
	srv.Of("").RemoveAllListeners(name)
}

// authorize origin!!
func (srv *SocketIOServer) handShake(w http.ResponseWriter, r *http.Request) {
	if srv.authorize != nil {
		if ok := srv.authorize(r); !ok {
			http.Error(w, "", 401)
			return
		}
	}
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("origin"))
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	sessionId := NewSessionID()
	if sessionId == "" {
		http.Error(w, "", 503)
		return
	}
	transportNames := srv.transports.GetTransportNames()
	fmt.Fprintf(w, "%s:%d:%d:%s",
		sessionId,
		srv.heartbeatTimeout,
		srv.closingTimeout,
		strings.Join(transportNames, ","))
	session := NewSession(srv.eventEmitters, sessionId)
	srv.addSession(session)
}

func (srv *SocketIOServer) addSession(ss *Session) {
	srv.mutex.Lock()
	defer srv.mutex.Unlock()
	srv.sessions[ss.SessionId] = ss
}

func (srv *SocketIOServer) removeSession(ss *Session) {
	srv.mutex.Lock()
	defer srv.mutex.Unlock()
	delete(srv.sessions, ss.SessionId)
}

func (srv *SocketIOServer) getSession(sessionId string) *Session {
	srv.mutex.RLock()
	defer srv.mutex.RUnlock()
	return srv.sessions[sessionId]
}

func (srv *SocketIOServer) heartbeat() {

}
