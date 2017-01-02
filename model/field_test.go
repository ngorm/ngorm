package model

import (
	"reflect"
	"testing"

	"github.com/gernest/ngorm/errmsg"
)

type sample struct {
	Key int
}

func TestField_Set(t *testing.T) {
	f := &Field{}
	a := 1
	err := f.Set(a)

	if err == nil {
		t.Error("expected ", errmsg.ErrInvalidFieldValue)
	} else {
		if err != errmsg.ErrInvalidFieldValue {
			t.Errorf("expected %v got %v", errmsg.ErrInvalidFieldValue, err)
		}
	}
	b := 10
	f.Field = reflect.ValueOf(&b)
	err = f.Set(a)
	if err == nil {
		t.Error("expected ", errmsg.ErrUnaddressable)
	} else {
		if err != errmsg.ErrUnaddressable {
			t.Errorf("expected %v got %v", errmsg.ErrUnaddressable, err)
		}
	}
	s := &sample{}
	f.Field = reflect.ValueOf(s).Elem().Field(0)
	err = f.Set(a)
	if err != nil {
		t.Error(err)
	}
}
