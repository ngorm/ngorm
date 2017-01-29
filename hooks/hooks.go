package hooks

import (
	"sync"

	"github.com/ngorm/ngorm/engine"
	"github.com/ngorm/ngorm/model"
)

//Hook is a nterface that is executed at a particular point in time. This allows
//doing additional transformation of any portion of the engine.Engine, and it
//can be a way to overide default behaviour.
type Hook interface {
	Name() string
	Exec(h *Book, e *engine.Engine) error
}

//Hooks a safe struct that holds a map of hooks. Use this to provide a safe
//group of related hooks.
type Hooks struct {
	h  map[string]Hook
	mu sync.RWMutex
}

//Set saves the hook.
func (h *Hooks) Set(hk Hook) {
	h.mu.Lock()
	h.h[hk.Name()] = hk
	h.mu.Unlock()
}

//Get returns a saved hook.
func (h *Hooks) Get(name string) (Hook, bool) {
	h.mu.RLock()
	hk, ok := h.h[name]
	h.mu.RUnlock()
	return hk, ok
}

//NewHooks retruns an initialized Hooks instance.
func NewHooks() *Hooks {
	return &Hooks{h: make(map[string]Hook)}
}

type simpleHook struct {
	name string
	e    func(*Book, *engine.Engine) error
}

func (s *simpleHook) Name() string {
	return s.name
}

func (s *simpleHook) Exec(h *Book, e *engine.Engine) error {
	return s.e(h, e)
}

//HookFunc wraps the function f into a struct that statisfies Hook interface.
func HookFunc(name string, f func(*Book, *engine.Engine) error) Hook {
	return &simpleHook{name: name, e: f}
}

//Book a collection of hooks used by ngorm.
type Book struct {
	Create *Hooks
	Delete *Hooks
	Update *Hooks
	Save   *Hooks
	Query  *Hooks
}

//DefaultBook returns the default ngorm Book. This has all default hooks set.
func DefaultBook() *Book {
	b := &Book{
		Create: NewHooks(),
		Delete: NewHooks(),
		Update: NewHooks(),
		Save:   NewHooks(),
		Query:  NewHooks(),
	}

	// Create hooks
	b.Create.Set(HookFunc(model.Create, Create))
	b.Create.Set(HookFunc(model.HookBeforeCreate, BeforeCreate))
	b.Create.Set(HookFunc(model.HookCreateExec, CreateExec))
	b.Create.Set(HookFunc(model.HookCreateSQL, CreateSQL))
	b.Create.Set(HookFunc(model.HookSaveBeforeAss, SaveBeforeAssociation))

	// Query hooks
	b.Query.Set(HookFunc(model.Query, Query))
	b.Query.Set(HookFunc(model.HookQueryExec, QueryExec))
	b.Query.Set(HookFunc(model.HookQuerySQL, QuerySQL))
	b.Query.Set(HookFunc(model.HookAfterQuery, AfterQuery))

	// Update hooks
	b.Update.Set(HookFunc(model.BeforeUpdate, BeforeUpdate))
	b.Update.Set(HookFunc(model.AfterUpdate, AfterUpdate))
	b.Update.Set(HookFunc(model.HookUpdateTimestamp, UpdateTimestamp))
	b.Update.Set(HookFunc(model.HookAssignUpdatingAttrs, AssignUpdatingAttrs))
	b.Update.Set(HookFunc(model.HookSaveBeforeAss, SaveBeforeAssociation))
	b.Update.Set(HookFunc(model.HookUpdateSQL, UpdateSQL))
	b.Update.Set(HookFunc(model.HookUpdateExec, UpdateExec))
	b.Update.Set(HookFunc(model.Update, Update))

	// Delete
	b.Delete.Set(HookFunc(model.Delete, Delete))
	b.Delete.Set(HookFunc(model.DeleteSQL, DeleteSQL))
	b.Delete.Set(HookFunc(model.BeforeDelete, BeforeDelete))
	b.Delete.Set(HookFunc(model.AfterDelete, AfterDelete))
	return b
}
