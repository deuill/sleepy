// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

// Package template contains methods for rendering templates and
// accompanying i18n definitions via RPC.
package template

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thoughtmonster/sleepy/core/config"
	"github.com/thoughtmonster/sleepy/core/server"
	"github.com/Wuvist/mustache"
)

type Template struct {
	// Contains private or unexported fields.
	conf *config.Config
}

type Request struct {
	Auth     string
	Template struct {
		Checksum string
		Path     string
		Data     string
	}
	Layout struct {
		Checksum string
		Path     string
		Data     string
	}
	Partials []struct {
		Checksum string
		Path     string
		Data     string
	}
	I18n struct {
		Origin string
		Target string
		Tables []struct {
			Checksum string
			Path     string
			Data     string
		}
	}
	Data map[string]interface{}
}

func (t *Template) Render(p Request) (string, error) {
	// Check cache and fill in data, if available.
	if p.Template.Checksum != "" {
		p.Template.Data = t.check(p.Template.Path, p.Auth, p.Template.Checksum)

		if p.Template.Data == "" {
			return "", nil
		}
	}

	if p.Layout.Checksum != "" {
		p.Layout.Data = t.check(p.Layout.Path, p.Auth, p.Layout.Checksum)

		if p.Layout.Data == "" {
			return "", nil
		}
	}

	for i, partial := range p.Partials {
		if partial.Checksum != "" {
			p.Partials[i].Data = t.check(partial.Path, p.Auth, partial.Checksum)

			if p.Partials[i].Data == "" {
				return "", nil
			}
		}
	}

	for i, table := range p.I18n.Tables {
		if table.Checksum != "" {
			p.I18n.Tables[i].Data = t.check(table.Path, p.Auth, table.Checksum)

			if p.I18n.Tables[i].Data == "" {
				return "", nil
			}
		}
	}

	if p.Template.Data == "" {
		return "", fmt.Errorf("Template is empty, please specify a valid template")
	} else if p.Template.Checksum == "" && p.Template.Path != "" {
		t.store(p.Template.Path, p.Auth, p.Template.Data)
	}

	for _, partial := range p.Partials {
		if partial.Checksum == "" && partial.Path != "" {
			t.store(partial.Path, p.Auth, partial.Data)
		}
	}

	for _, table := range p.I18n.Tables {
		if table.Checksum == "" && table.Path != "" {
			t.store(table.Path, p.Auth, table.Data)
		}
	}

	// Navigate to template directory, for partials.
	datadir, _ := t.conf.String("directories", "data")
	os.Chdir(datadir + "/cache/" + p.Auth + "/template/" + filepath.Dir(p.Template.Path) + "/")

	var result string
	if p.Layout.Data != "" {
		// Render template in layout.
		if p.Layout.Checksum == "" && p.Layout.Path != "" {
			t.store(p.Layout.Path, p.Auth, p.Layout.Data)
		}

		result = mustache.RenderInLayout(p.Template.Data, p.Layout.Data, p.Data)
	} else {
		// Render simple template.
		result = mustache.Render(p.Template.Data, p.Data)
	}

	if len(p.I18n.Tables) > 0 && p.I18n.Origin != "" && p.I18n.Target != "" {
		tables := make([]string, len(p.I18n.Tables))

		for i, table := range p.I18n.Tables {
			tables[i] = table.Data
		}

		parser, err := New(result, tables, "[[", "]]")
		if err != nil {
			return "", nil
		}

		result = parser.Execute(p.I18n.Origin, p.I18n.Target)
	}

	return result, nil
}

func (t *Template) check(path, authkey, checksum string) string {
	datadir, _ := t.conf.String("directories", "data")

	file, err := os.Open(datadir + "/cache/" + authkey + "/template/" + path)
	if err != nil {
		return ""
	}

	buf, _ := ioutil.ReadAll(file)
	if fmt.Sprintf("%x", sha1.Sum(buf)) == checksum {
		return string(buf)
	}

	return ""
}

func (t *Template) store(path, authkey, template string) error {
	datadir, _ := t.conf.String("directories", "data")

	p := datadir + "/cache/" + authkey + "/template/" + filepath.Dir(path) + "/"
	n := filepath.Base(path)

	if err := os.MkdirAll(p, 0755); err != nil {
		return err
	}

	out, err := os.Create(p + n)
	if err != nil {
		return err
	}

	defer out.Close()

	if _, err = out.WriteString(template); err != nil {
		return err
	}

	return nil
}

func (t *Template) Setup(config *config.Config) error {
	t.conf = config
	return nil
}

func init() {
	server.Register(&Template{nil})
}
