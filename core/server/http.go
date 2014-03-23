// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

package server

import (
	"net/http"
	"os"
	"path"
	"strings"
)

type fileHandler struct {
	root string
}

func HTTPHandler(root string) http.Handler {
	return &fileHandler{root}
}

func (f *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}

	upath = f.root + path.Clean(upath)

	// Return 404 for directories.
	i, error := os.Stat(upath)
	if error != nil || i.IsDir() {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, upath)
}
