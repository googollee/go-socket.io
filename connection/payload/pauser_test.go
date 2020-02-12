package payload

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPauserTrigger(t *testing.T) {
	should := assert.New(t)
	p := newPauser()

	ok := p.Working()
	should.True(ok)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ok := p.Pause()
		should.True(ok)
		defer p.Resume()
	}()

	select {
	case <-p.PausingTrigger():
	case <-time.After(time.Second / 10):
		should.True(false, "should not run here")
	}
	select {
	case <-p.PausedTrigger():
		should.True(false, "should not run here")
	case <-time.After(time.Second / 10):
	}

	go func() {
		time.Sleep(time.Second / 10)
		p.Done()
	}()

	select {
	case <-p.PausedTrigger():
	case <-time.After(time.Second):
		should.True(false, "should not run here")
	}

	wg.Wait()

	select {
	case <-p.PausingTrigger():
		should.True(false, "should not run here")
	case <-p.PausedTrigger():
		should.True(false, "should not run here")
	case <-time.After(time.Second / 10):
	}

}

func TestPauserPauseOnlyOnce(t *testing.T) {
	should := assert.New(t)
	p := newPauser()
	s := make(chan int)

	go func() {
		ok := p.Pause()
		should.True(ok)
		defer p.Resume()
		s <- 1
		<-s
	}()

	<-s
	ok := p.Pause()
	should.False(ok)
	s <- 1
}

func TestPauserPauseAfterResume(t *testing.T) {
	should := assert.New(t)
	p := newPauser()

	ok := p.Pause()
	should.True(ok)
	p.Resume()

	ok = p.Pause()
	should.True(ok)
	p.Resume()
}

func TestPauserPauseMultiplyResumeOnce(t *testing.T) {
	should := assert.New(t)
	p := newPauser()

	ok := p.Pause()
	should.True(ok)
	for i := 0; i < 10; i++ {
		ok = p.Pause()
		should.False(ok)
	}
	p.Resume()

	// check if it reset to normal
	ok = p.Pause()
	should.True(ok)
	p.Resume()
}

func TestPauserConcurrencyWorkingDone(t *testing.T) {
	p := newPauser()
	wg := sync.WaitGroup{}

	f := func() {
		defer wg.Done()
		should := assert.New(t)
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		time.Sleep(time.Microsecond * time.Duration(r.Intn(100)))
		ok := p.Working()
		should.True(ok)
		defer p.Done()
		time.Sleep(time.Microsecond * time.Duration(r.Intn(100)))
	}

	max := 1000
	wg.Add(max)
	for i := 0; i < max; i++ {
		go f()
	}

	wg.Wait()
}

func TestPauserCanWorkingDuringPauseWaiting(t *testing.T) {
	should := assert.New(t)
	p := newPauser()
	wg := sync.WaitGroup{}

	ok := p.Working()
	should.True(ok)

	wg.Add(1)
	go func() {
		defer wg.Done()
		o := p.Pause()
		should.True(o)
		defer p.Resume()
	}()
	<-p.PausingTrigger()

	ok = p.Working()
	should.True(ok)

	p.Done()
	p.Done()
	wg.Wait()
}

func TestPauserPauseWhenAllDone(t *testing.T) {
	should := assert.New(t)
	p := newPauser()

	n := 10
	for i := 0; i < n; i++ {
		ok := p.Working()
		should.True(ok)
	}
	for i := 0; i < n; i++ {
		p.Done()
	}

	ok := p.Pause()
	should.True(ok)

	ok = p.Pause()
	should.False(ok)

	p.Resume()
}

func TestPauserOnlyOnePauseAfterWaiting(t *testing.T) {
	should := assert.New(t)
	count := int64(0)
	wg := sync.WaitGroup{}
	p := newPauser()

	ok := p.Working()
	should.True(ok)

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			ok := p.Pause()
			if ok {
				atomic.AddInt64(&count, 1)
			}
		}()
	}
	time.Sleep(time.Second / 10) // Wait all goroutines pausing

	p.Done()
	wg.Wait()
	should.Equal(int64(1), count)
	p.Resume()
}

func TestPauserCannotWorkingAfterPause(t *testing.T) {
	should := assert.New(t)
	p := newPauser()

	ok := p.Pause()
	should.True(ok)
	defer p.Resume()

	ok = p.Working()
	should.False(ok)
	p.Done()
}

func TestPauserRandom(t *testing.T) {
	p := newPauser()
	wg := sync.WaitGroup{}
	n := 100

	f := func() {
		defer wg.Done()
		should := assert.New(t)
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		time.Sleep(time.Millisecond * time.Duration(r.Intn(n)))
		ok := p.Working()
		should.True(ok)
		defer p.Done()
		time.Sleep(time.Millisecond * time.Duration(r.Intn(n)))
	}

	max := 1000
	wg.Add(max)
	for i := 0; i < max; i++ {
		go f()
	}

	should := assert.New(t)
	// Make sure waiting pause.
	ok := p.Working()
	should.True(ok)
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * time.Duration(n/2))
		p.Done()
	}()

	start := time.Now()
	ok = p.Pause()
	end := time.Now()
	should.True(ok)
	should.True(end.Sub(start) > time.Millisecond)
	wg.Wait()
}
