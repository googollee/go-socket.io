package engineio

import (
	"github.com/googollee/go-engine.io/transport"
)

type transportCreaters map[string]transport.Creater

func (t transportCreaters) Upgrades() []string {
	var ret []string
	for _, creater := range t {
		if creater.Upgrading {
			ret = append(ret, creater.Name)
		}
	}
	return ret
}

func (t transportCreaters) Get(name string) transport.Creater {
	return t[name]
}
