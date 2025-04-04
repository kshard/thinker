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
)

// A FileSystem provides access to a hierarchical file system.
// The abstraction support I/O to local file system or AWS S3.
// Use it with https://github.com/fogfish/stream
type FileSystem interface {
	fs.FS
	Create(path string, attr *struct{}) (File, error)
	Remove(path string) error
}

// File provides I/O access to individual object on the file system.
type File = interface {
	io.Writer
	io.Closer
	Stat() (fs.FileInfo, error)
	Cancel() error
}

// File system worker
type Worker struct {
	Reader FileSystem
	Writer FileSystem
}

func NewWorker(reader, writer FileSystem) *Worker {
	return &Worker{
		Reader: reader,
		Writer: writer,
	}
}

func (lib *Worker) Walk(ctx context.Context, dir string, f func(context.Context, *Worker, string) error) error {
	return fs.WalkDir(lib.Reader, dir,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			if err := f(ctx, lib, path); err != nil {
				return err
			}

			return nil
		},
	)
}

func (lib *Worker) ReadFile(path string) ([]byte, error) {
	return fs.ReadFile(lib.Reader, path)
}

func (lib *Worker) WriteFile(path string, data []byte) error {
	fd, err := lib.Writer.Create(path, nil)
	if err != nil {
		return err
	}

	_, err = fd.Write(data)
	if err != nil {
		return err
	}

	err = fd.Close()
	if err != nil {
		return err
	}

	return nil
}
