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

	for i := range 10 {
		// key := "images"
		key := fmt.Sprintf("images_%d", i)
		id := generateID()
		data := []byte("image file data")

		if _, err := s.writeStream(id, key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}

		if ok := s.Has(id, key); !ok {
			t.Errorf("expected to have key %s", key)
		}

		_, r, err := s.Read(id, key)
		if err != nil {
			t.Error(err)
		}

		b, _ := io.ReadAll(r)
		fmt.Println(string(b))

		// I don't like it
		if rc, ok := r.(io.ReadCloser); ok {
			rc.Close()
		}

		if string(b) != string(data) {
			t.Errorf("expected: %s, got: %s", data, b)
		}

		if err := s.Delete(id, key); err != nil {
			t.Error(err)
		}

		if ok := s.Has(id, key); ok {
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
