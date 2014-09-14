package engineio

import (
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
