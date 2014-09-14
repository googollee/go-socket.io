package engineio

import (
	"github.com/googollee/go-engine.io/polling"
	"github.com/googollee/go-engine.io/transport"
	"github.com/googollee/go-engine.io/websocket"
)

type transportCreaters map[string]transport.Creater

var creaters transportCreaters

func init() {
	creaters = make(transportCreaters)
	for _, creater := range []transportCreaters{polling.Creater, websocket.Creater} {
		creaters[creater.Name] = creater
	}
}

func transportUpgrades() []string {
	var ret []string
	for _, creater := range creaters {
		if creater.Upgrading {
			ret = append(ret, creater.Name)
		}
	}
	return ret
}

func getCreater(name string) transport.Creater {
	return creaters[name]
}
