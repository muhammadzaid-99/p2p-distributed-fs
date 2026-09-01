package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashstr := hex.EncodeToString(hash[:])

	blockSize := 5
	sliceLen := len(hashstr) / blockSize

	paths := make([]string, sliceLen)
	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, i*blockSize+blockSize
		paths[i] = hashstr[from:to]
	}

	return PathKey{
		PathName: strings.Join(paths, "/"),
		Filename: hashstr,
	}
}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	PathName string
	Filename string
}

func (p *PathKey) FirstPathName() string {
	first, _, _ := strings.Cut(p.PathName, "/")
	return first
}

func (p *PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.Filename)
}

func DefaultPathTransformFunc(key string) string {
	return key
}

type StoreOpts struct {
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	_, err := os.Stat(pathKey.FullPath())
	return err != nil
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)
	defer log.Printf("deleted %s from disk", pathKey.Filename)
	return os.RemoveAll(pathKey.FirstPathName())
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)

	return buf, err
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	return os.Open(pathKey.FullPath())
}

func (s *Store) writeStream(key string, r io.Reader) error {
	pathKey := s.PathTransformFunc(key)

	if err := os.MkdirAll(pathKey.PathName, os.ModePerm); err != nil {
		return err
	}

	fullPath := pathKey.FullPath()

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}

	log.Printf("Written %d bytes to disk: %s", n, fullPath)

	return nil
}
