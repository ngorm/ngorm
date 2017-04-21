package hooks

import (
	"sync"

	"fmt"

	"github.com/ngorm/ngorm/engine"
	"github.com/ngorm/ngorm/model"
)

var hPool = sync.Pool{
	New: func() interface{} {
		return &simpleHook{}
	},
}

type HookType uint

const (
	CreateHook HookType = 1 << iota
	QueryHook
	UpdateHook
	DeleteHook
	SaveHook
)

//Hook is a interface that is executed at a particular point in time. This allows
//doing additional transformation of any portion of the engine.Engine, and it
//can be a way to overide default behavior.
type Hook interface {
	Name() string
	Exec(h *Book, e *engine.Engine) error
}

//Hooks a safe struct that holds a map of hooks. Use this to provide a safe
//group of related hooks.
type Hooks struct {
	typ HookType
	h   map[string]Hook
	mu  sync.RWMutex
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
	defer h.mu.RUnlock()
	if !ok {
		return defaultHook(h.typ, name)
	}
	return hk, ok
}

func (b *Book) Exec(typ HookType, key string, e *engine.Engine) error {
	h, ok := b.Get(typ, key)
	defer func() {
		if h != nil {
			if _, ok = h.(*simpleHook); ok {
				hPool.Put(h.(*simpleHook))
			}
		}
	}()
	if !ok {
		return nil
	}
	return h.Exec(b, e)
}

func (b *Book) MustExec(typ HookType, key string, e *engine.Engine) error {
	h, ok := b.Get(typ, key)
	defer func() {
		if h != nil {
			if _, ok = h.(*simpleHook); ok {
				hPool.Put(h.(*simpleHook))
			}
		}
	}()
	if !ok {
		return fmt.Errorf("ngorm: missing %s hook", key)
	}
	return h.Exec(b, e)
}

func (b *Book) Get(typ HookType, key string) (Hook, bool) {
	switch typ {
	case CreateHook:
		return b.Create.Get(key)
	case SaveHook:
		return b.Save.Get(key)
	case QueryHook:
		return b.Query.Get(key)
	case UpdateHook:
		return b.Update.Get(key)
	case DeleteHook:
		return b.Delete.Get(key)
	}
	return nil, false
}

func defaultHook(typ HookType, key string) (Hook, bool) {
	switch typ {
	case CreateHook:
		return defaultCreateHooks(key)
	case QueryHook:
		return defaultQueryHooks(key)
	case SaveHook:
		return defaultSaveHooks(key)
	case UpdateHook:
		return defaultUpdateHooks(key)
	case DeleteHook:
		return defaultDeleteHooks(key)
	}
	return nil, false
}

func defaultCreateHooks(key string) (Hook, bool) {
	s := hPool.Get().(*simpleHook)
	switch key {
	case model.Create:
		s.name = key
		s.e = Create
		return s, true
	case model.BeforeCreate:
		s.name = key
		s.e = BeforeCreate
		return s, true
	case model.HookCreateExec:
		s.name = key
		s.e = CreateExec
		return s, true
	case model.HookCreateSQL:
		s.name = key
		s.e = CreateSQL
		return s, true
	case model.HookSaveBeforeAss:
		s.name = key
		s.e = SaveBeforeAssociation
		return s, true
	case model.AfterCreate:
		s.name = key
		s.e = AfterCreate
		return s, true
	case model.HookUpdateTimestamp:
		s.name = key
		s.e = UpdateTimestamp
		return s, true
	}
	return nil, false
}

func defaultSaveHooks(key string) (Hook, bool) {
	return nil, false
}

func defaultQueryHooks(key string) (Hook, bool) {
	s := hPool.Get().(*simpleHook)
	switch key {
	case model.Query:
		s.name = key
		s.e = Query
		return s, true

	case model.HookQueryExec:
		s.name = key
		s.e = QueryExec
		return s, true

	case model.HookQuerySQL:
		s.name = key
		s.e = QuerySQL
		return s, true

	case model.HookAfterQuery:
		s.name = key
		s.e = AfterQuery
		return s, true

	case model.Preload:
		s.name = key
		s.e = Preload
		return s, true
	}
	return nil, false
}

func defaultUpdateHooks(key string) (Hook, bool) {
	s := hPool.Get().(*simpleHook)
	switch key {
	case model.BeforeUpdate:
		s.name = key
		s.e = BeforeUpdate
		return s, true

	case model.AfterUpdate:
		s.name = key
		s.e = AfterUpdate
		return s, true

	case model.HookUpdateTimestamp:
		s.name = key
		s.e = UpdateTimestamp
		return s, true

	case model.HookAssignUpdatingAttrs:
		s.name = key
		s.e = AssignUpdatingAttrs
		return s, true

	case model.HookSaveBeforeAss:
		s.name = key
		s.e = SaveBeforeAssociation
		return s, true

	case model.HookSaveAfterAss:
		s.name = key
		s.e = AfterAssociation
		return s, true

	case model.HookUpdateSQL:
		s.name = key
		s.e = UpdateSQL
		return s, true

	case model.HookUpdateExec:
		s.name = key
		s.e = UpdateExec
		return s, true

	case model.Update:
		s.name = key
		s.e = Update
		return s, true
	}
	return nil, false
}

func defaultDeleteHooks(key string) (Hook, bool) {
	s := hPool.Get().(*simpleHook)
	switch key {
	case model.Delete:
		s.name = key
		s.e = Delete
		return s, true

	case model.DeleteSQL:
		s.name = key
		s.e = DeleteSQL
		return s, true

	case model.BeforeDelete:
		s.name = key
		s.e = BeforeDelete
		return s, true

	case model.AfterDelete:
		s.name = key
		s.e = AfterDelete
		return s, true
	}
	return nil, false
}

//NewHooks retruns an initialized Hooks instance.
func NewHooks(typ HookType) *Hooks {
	return &Hooks{typ: typ, h: make(map[string]Hook)}
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

//HookFunc wraps the function f into a struct that satisfies Hook interface.
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
		Create: NewHooks(CreateHook),
		Delete: NewHooks(DeleteHook),
		Update: NewHooks(UpdateHook),
		Save:   NewHooks(SaveHook),
		Query:  NewHooks(QueryHook),
	}

	// // Create hooks
	// b.Create.Set(HookFunc(model.Create, Create))
	// b.Create.Set(HookFunc(model.BeforeCreate, BeforeCreate))
	// b.Create.Set(HookFunc(model.HookCreateExec, CreateExec))
	// b.Create.Set(HookFunc(model.HookCreateSQL, CreateSQL))
	// b.Create.Set(HookFunc(model.HookSaveBeforeAss, SaveBeforeAssociation))
	// b.Create.Set(HookFunc(model.AfterCreate, AfterCreate))
	// b.Create.Set(HookFunc(model.HookUpdateTimestamp, UpdateTimestamp))

	// // Query hooks
	// b.Query.Set(HookFunc(model.Query, Query))
	// b.Query.Set(HookFunc(model.HookQueryExec, QueryExec))
	// b.Query.Set(HookFunc(model.HookQuerySQL, QuerySQL))
	// b.Query.Set(HookFunc(model.HookAfterQuery, AfterQuery))
	// b.Query.Set(HookFunc(model.Preload, Preload))

	// // Update hooks
	// b.Update.Set(HookFunc(model.BeforeUpdate, BeforeUpdate))
	// b.Update.Set(HookFunc(model.AfterUpdate, AfterUpdate))
	// b.Update.Set(HookFunc(model.HookUpdateTimestamp, UpdateTimestamp))
	// b.Update.Set(HookFunc(model.HookAssignUpdatingAttrs, AssignUpdatingAttrs))
	// b.Update.Set(HookFunc(model.HookSaveBeforeAss, SaveBeforeAssociation))
	// b.Update.Set(HookFunc(model.HookSaveAfterAss, AfterAssociation))
	// b.Update.Set(HookFunc(model.HookUpdateSQL, UpdateSQL))
	// b.Update.Set(HookFunc(model.HookUpdateExec, UpdateExec))
	// b.Update.Set(HookFunc(model.Update, Update))

	// // Delete
	// b.Delete.Set(HookFunc(model.Delete, Delete))
	// b.Delete.Set(HookFunc(model.DeleteSQL, DeleteSQL))
	// b.Delete.Set(HookFunc(model.BeforeDelete, BeforeDelete))
	// b.Delete.Set(HookFunc(model.AfterDelete, AfterDelete))
	return b
}
