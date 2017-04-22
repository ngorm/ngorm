package engine

import "github.com/ngorm/ngorm/model"

var l *list

func init() {
	l = &list{c: make(chan *Engine, 1000*3)}
}

type list struct {
	c chan *Engine
}

func (l *list) Get() (e *Engine) {
	select {
	case e = <-l.c:
	default:
		e = &Engine{
			Scope:  model.NewScope(),
			Search: &model.Search{},
		}
	}
	return
}

func (l *list) Put(e *Engine) {
	if e != nil {
		e.reset()
		select {
		case l.c <- e:
		default:
		}
	}

}

func Get() *Engine {
	return l.Get()
}

func Put(e *Engine) {
	l.Put(e)
}
