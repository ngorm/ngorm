package hooks

import (
	"sync"

	"github.com/gernest/ngorm/engine"
)

//Hook is a nterface that is executed at a particular point in time. This allows
//doing additional transformation of any portion of the engine.Engine, and it
//can be a way to overide default behaviour.
type Hook interface {
	Name() string
	Exec(e *engine.Engine) error
}

type Hooks struct {
	h  map[string]Hook
	mu sync.RWMutex
}

func (h *Hooks) Set(hk Hook) {
	h.mu.Lock()
	h.h[hk.Name()] = hk
	h.mu.Unlock()
}

func (h *Hooks) Get(name string) (Hook, bool) {
	h.mu.RLock()
	hk, ok := h.h[name]
	h.mu.RUnlock()
	return hk, ok
}

func NewHooks() *Hooks {
	return &Hooks{h: make(map[string]Hook)}
}

type simpleHook struct {
	name string
	e    func(*engine.Engine) error
}

func (s *simpleHook) Name() string {
	return s.name
}

func (s *simpleHook) Exec(e *engine.Engine) error {
	return s.e(e)
}

func HookFunc(name string, f func(*engine.Engine) error) Hook {
	return &simpleHook{name: name, e: f}
}
