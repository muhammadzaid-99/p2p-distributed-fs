package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "images2"
	pathKey := CASPathTransformFunc(key)
	expPathname := "8a274/96697/3ad7f/c10ee/40ceb/f5621/a462a/b997a"
	expOrgKey := "8a274966973ad7fc10ee40cebf5621a462ab997a"
	if pathKey.PathName != expPathname {
		t.Errorf("expected: %s, got: %s", expPathname, pathKey.PathName)
	}
	if pathKey.Filename != expOrgKey {
		t.Errorf("expected: %s, got: %s", expOrgKey, pathKey.Filename)
	}
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)
	key := "images"
	data := []byte("image file data")

	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}

	b, _ := io.ReadAll(r)
	fmt.Println(string(b))

	if string(b) != string(data) {
		t.Errorf("expected: %s, got: %s", data, b)
	}
}
