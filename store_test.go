package main

import (
	"bytes"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "images2"
	pathname := CASPathTransformFunc(key)
	exp := "8a274/96697/3ad7f/c10ee/40ceb/f5621/a462a/b997a"
	if pathname != exp {
		t.Errorf("expected: %s, got %s", exp, pathname)
	}
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)

	data := bytes.NewReader([]byte("image file data"))
	if err := s.writeStream("images", data); err != nil {
		t.Error(err)
	}
}
