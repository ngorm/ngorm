package hooks

import "github.com/gernest/ngorm/engine"

//Hook is a function that is executed at a paricular point in time. This allows
//doing additional transformation of any portion of the engine.Engine, and it
//can be a way to overide default behaviour.
type Hook func(e *engine.Engine) error
