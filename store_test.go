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
	expFilename := "8a274966973ad7fc10ee40cebf5621a462ab997a"
	if pathKey.PathName != expPathname {
		t.Errorf("expected: %s, got: %s", expPathname, pathKey.PathName)
	}
	if pathKey.Filename != expFilename {
		t.Errorf("expected: %s, got: %s", expFilename, pathKey.Filename)
	}
}

func TestStore(t *testing.T) {
	s := newStore()
	defer teardown(t, s)

	for i := range 50 {
		// key := "images"
		key := fmt.Sprintf("images_%d", i)
		data := []byte("image file data")

		if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}

		if ok := s.Has(key); !ok {
			t.Errorf("expected to have key %s", key)
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

		if err := s.Delete(key); err != nil {
			t.Error(err)
		}

		if ok := s.Has(key); ok {
			t.Errorf("expected to not have key %s", key)
		}
	}

}

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	return NewStore(opts)
}

func teardown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}
