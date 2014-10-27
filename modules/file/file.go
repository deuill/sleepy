// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

// Package file contains methods for uploading, retrieving and
// removing files efficiently and transparently via RPC.
package file

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/deuill/sleepy/core/config"
	"github.com/deuill/sleepy/core/server"
	"github.com/deuill/sleepy/core/user"
)

type File struct {
	// Contains private or unexported fields.
	conf *config.Config
	id   map[string]string
}

type Request struct {
	Auth     string
	Remote   string
	Checksum string
	Filename string
}

func (f *File) Get(p Request) (string, error) {
	path, err := f.filepath(&p)
	if err != nil {
		return "", nil
	}

	dir, err := os.Open(f.conf.S("directories", "data") + "/serve" + path)
	if err != nil {
		return "", nil
	}

	defer dir.Close()

	files, err := dir.Readdir(-1)
	if len(files) == 0 {
		return "", nil
	}

	var filename string

	// Get the first file in the directory.
	for _, file := range files {
		if file.IsDir() == false {
			filename = file.Name()
			break
		}
	}

	if filename != "" {
		return f.conf.S("http", "address") + ":" + f.conf.S("http", "port") + path + filename, nil
	}

	return "", nil
}

func (f *File) Upload(p Request) (string, error) {
	path, err := f.filepath(&p)
	if err != nil {
		return "", nil
	}

	var src io.ReadCloser
	if p.Remote != "" {
		resp, err := http.Get(p.Remote)
		if err != nil {
			return "", nil
		} else if resp.StatusCode != 200 {
			resp.Body.Close()
			return "", nil
		}

		src = resp.Body
	} else {
		tmpfile := os.TempDir() + "/sleepy/" + f.id[p.Auth] + "/" + p.Checksum
		if src, err = os.Open(tmpfile); err != nil {
			return "", nil
		}

		defer os.Remove(tmpfile)
	}

	defer src.Close()

	datadir := f.conf.S("directories", "data")
	if err = os.MkdirAll(datadir+"/serve"+path, 0755); err != nil {
		return "", nil
	}

	dst, err := os.Create(datadir + "/serve" + path + p.Filename)
	if err != nil {
		return "", nil
	}

	defer dst.Close()
	io.Copy(dst, src)

	return f.conf.S("http", "address") + ":" + f.conf.S("http", "port") + path + p.Filename, nil
}

func (f *File) Delete(p Request) (bool, error) {
	path, err := f.filepath(&p)
	if err != nil {
		return false, err
	}

	if err = os.RemoveAll(f.conf.S("directories", "data") + "/serve" + path); err != nil {
		return false, err
	}

	return true, nil
}

func (f *File) filepath(p *Request) (string, error) {
	if len(p.Checksum) != 40 {
		return "", fmt.Errorf("checksum does not appear to be an SHA1 hash.")
	}

	if _, exists := f.id[p.Auth]; !exists {
		u, err := user.Auth(p.Auth)
		if err != nil {
			return "", err
		}

		f.id[p.Auth] = strconv.FormatInt(int64(u.Id), 10)
	}

	c := p.Checksum
	hash := c[:2] + "/" + c[2:6] + "/" + c[6:14] + "/" + c[14:27] + "/" + c[27:]
	path := "/" + f.id[p.Auth] + "/" + hash + "/"

	return path, nil
}

func (f *File) Setup(config *config.Config) error {
	f.conf = config

	return nil
}

func init() {
	server.Register(&File{
		&config.Config{},
		make(map[string]string),
	})
}
