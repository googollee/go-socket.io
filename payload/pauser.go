package payload

import "sync"

type pauser struct {
	l       sync.Mutex
	c       *sync.Cond
	worker  int
	pausing chan struct{}
	paused  chan struct{}
}

func newPauser() *pauser {
	ret := &pauser{
		pausing: make(chan struct{}),
		paused:  make(chan struct{}),
	}
	ret.c = sync.NewCond(&ret.l)
	return ret
}

func (p *pauser) Pause() bool {
	p.l.Lock()
	defer p.l.Unlock()

	if p.paused == nil {
		return false
	}
	if p.pausing != nil {
		close(p.pausing)
		p.pausing = nil
	}

	for p.worker != 0 {
		p.c.Wait()
	}

	if p.paused == nil {
		return false
	}
	close(p.paused)
	p.paused = nil
	p.c.Broadcast()

	return true
}

func (p *pauser) Resume() {
	p.l.Lock()
	defer p.l.Unlock()
	p.paused = make(chan struct{})
	p.pausing = make(chan struct{})
}

func (p *pauser) Working() bool {
	p.l.Lock()
	defer p.l.Unlock()
	if p.paused == nil {
		return false
	}
	p.worker++
	return true
}

func (p *pauser) Done() {
	p.l.Lock()
	defer p.l.Unlock()
	if p.paused == nil || p.worker == 0 {
		return
	}
	p.worker--
	p.c.Broadcast()
}

func (p *pauser) PausingTrigger() <-chan struct{} {
	p.l.Lock()
	defer p.l.Unlock()
	return p.pausing
}

func (p *pauser) PausedTrigger() <-chan struct{} {
	p.l.Lock()
	defer p.l.Unlock()
	return p.paused
}
