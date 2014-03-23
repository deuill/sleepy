// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

package template

import (
	"encoding/json"

	"github.com/melvinmt/gt"
)

type Parser struct {
	body       string
	leftDelim  string
	rightDelim string

	table gt.Strings
}

func (t *Parser) peek(str string, pos int) bool {
	if pos+len(str) >= len(t.body) {
		return false
	}

	if t.body[pos:pos+len(str)] == str {
		return true
	}

	return false
}

func (t *Parser) Execute(origin, target string) string {
	var start = -1
	var buf string

	var i18n = &gt.Build{
		Index:  t.table,
		Origin: origin,
		Target: target,
	}

	for p := 0; p < len(t.body); p++ {
		switch t.body[p] {
		case t.leftDelim[0]:
			if t.peek(t.leftDelim, p) && !t.peek(t.rightDelim, p+2) {
				start = p + 2
				p++
			}
		case t.rightDelim[0]:
			if start >= 0 && t.peek(t.rightDelim, p) {
				buf = buf + t.body[:start-len(t.leftDelim)] + i18n.T(string(t.body[start:p]))
				t.body = t.body[p+len(t.rightDelim):]

				start = -1
				p = 0
			}
		}
	}

	t.body = buf + t.body

	return t.body
}

func New(template string, tables []string, leftDelim, rightDelim string) (*Parser, error) {
	table, error := parseTables(tables)
	if error != nil {
		return nil, error
	}

	return &Parser{
		body:       template,
		leftDelim:  leftDelim,
		rightDelim: rightDelim,
		table:      table,
	}, nil
}

func parseTables(tables []string) (gt.Strings, error) {
	table := make(gt.Strings, 0)

	for _, tbl := range tables {
		var t gt.Strings

		error := json.Unmarshal([]byte(tbl), &t)
		if error != nil {
			return nil, error
		}

		for k, v := range t {
			if _, ok := table[k]; !ok {
				table[k] = make(map[string]string, 0)
			}

			table[k] = v
		}
	}

	return table, nil
}
