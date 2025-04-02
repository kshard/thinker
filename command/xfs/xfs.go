//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/kshard/thinker
//

package xfs

import (
	"context"
	"io"
	"io/fs"
	"strings"
)

type FD = interface {
	io.Writer
	io.Closer
	Stat() (fs.FileInfo, error)
	Cancel() error
}

type FileSystem interface {
	fs.FS
	Create(path string, attr *struct{}) (FD, error)
	Remove(path string) error
}

type XFS struct {
	fsys FileSystem
}

type File struct {
	Path  string
	Bytes []byte
}

func New(fsys FileSystem) *XFS {
	return &XFS{
		fsys: fsys,
	}
}

// Walk (recursivly) dir at filesystem, matching files with extension
func (xfs *XFS) Walk(ctx context.Context, dir string, ext string) (<-chan string, <-chan error) {
	out := make(chan string)
	exx := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(exx)
		exx <- fs.WalkDir(xfs.fsys, dir,
			func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.IsDir() || !strings.HasSuffix(path, ext) {
					return nil
				}

				select {
				case out <- path:
				case <-ctx.Done():
					return fs.SkipAll
				}
				return nil
			},
		)
	}()

	return out, exx
}

// Read file
func (xfs *XFS) Read(path string) (File, error) {
	b, err := fs.ReadFile(xfs.fsys, path)
	if err != nil {
		return File{Path: path}, err
	}

	return File{Path: path, Bytes: b}, nil
}

// Create file, writinh the content out
func (xfs *XFS) Create(file File) (File, error) {
	fd, err := xfs.fsys.Create(file.Path, nil)
	if err != nil {
		return file, err
	}

	_, err = fd.Write(file.Bytes)
	if err != nil {
		return file, err
	}

	err = fd.Close()
	if err != nil {
		return file, err
	}

	return file, nil
}

// Remove file
func (xfs *XFS) Remove(file File) (File, error) {
	err := xfs.fsys.Remove(file.Path)
	if err != nil {
		return file, err
	}

	return file, nil
}

// Lift agent into file system I/O context
func Echo(w interface{ Echo(string) (string, error) }) func(File) (File, error) {
	return func(in File) (File, error) {
		reply, err := w.Echo(string(in.Bytes))
		if err != nil {
			return in, err
		}
		return File{Path: in.Path, Bytes: []byte(reply)}, nil
	}
}

// Lift agent into file system I/O context
func Seek(w interface{ Seek(string) (string, error) }) func(File) (File, error) {
	return func(in File) (File, error) {
		reply, err := w.Seek(string(in.Bytes))
		if err != nil {
			return in, err
		}
		return File{Path: in.Path, Bytes: []byte(reply)}, nil
	}
}
