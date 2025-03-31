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
	"io/fs"
	"strings"

	"github.com/fogfish/stream"
)

type XFS struct {
	fsys stream.CreateFS[struct{}]
}

type File struct {
	Path  string
	Bytes []byte
}

func New(fsys stream.CreateFS[struct{}]) *XFS {
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

// Write file
func (xfs *XFS) Write(file File) (File, error) {
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
