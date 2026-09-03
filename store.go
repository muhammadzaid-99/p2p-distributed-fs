package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
)

const defaultRootFolderName = "pineapple"

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

func DefaultPathTransformFunc(key string) PathKey {
	return PathKey{
		PathName: key,
		Filename: key,
	}
}

type StoreOpts struct {
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}

	if opts.Root == "" {
		opts.Root = defaultRootFolderName
	}

	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	_, err := os.Stat(s.fromRoot(pathKey.FullPath()))
	return !errors.Is(err, fs.ErrNotExist)
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)
	defer log.Printf("deleted %s from disk", pathKey.Filename)
	return os.RemoveAll(s.fromRoot(pathKey.FirstPathName()))
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

func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	return os.Open(s.fromRoot(pathKey.FullPath()))
}

func (s *Store) writeStream(key string, r io.Reader) (int64, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameFromRoot := s.fromRoot(pathKey.PathName)
	if err := os.MkdirAll(pathNameFromRoot, os.ModePerm); err != nil {
		return 0, err
	}

	fullPathFromRoot := s.fromRoot(pathKey.FullPath())

	f, err := os.Create(fullPathFromRoot)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return 0, err
	}

	log.Printf("written %d bytes to disk: %s", n, fullPathFromRoot)

	return n, nil
}

func (s *Store) fromRoot(path string) string {
	return fmt.Sprintf("%s/%s", s.Root, path)
}
