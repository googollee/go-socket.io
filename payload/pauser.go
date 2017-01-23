package payload

import "sync"

type pauser struct {
	l      sync.Mutex
	c      *sync.Cond
	paused bool
	worker int
}

func newPauser() *pauser {
	ret := &pauser{}
	ret.c = sync.NewCond(&ret.l)
	return ret
}

func (p *pauser) Pause() bool {
	p.l.Lock()
	defer p.l.Unlock()
	if p.paused {
		return false
	}
	for p.worker != 0 {
		p.c.Wait()
	}

	// trigger other pausing call
	p.c.Broadcast()

	if p.paused {
		return false
	}
	p.paused = true
	return true
}

func (p *pauser) Resume() {
	p.l.Lock()
	defer p.l.Unlock()
	p.paused = false
	p.c.Signal()
}

func (p *pauser) Working() bool {
	p.l.Lock()
	defer p.l.Unlock()
	if p.paused {
		return false
	}
	p.worker++
	return true
}

func (p *pauser) Done() {
	p.l.Lock()
	defer p.l.Unlock()
	p.worker--
	p.c.Signal()
}
