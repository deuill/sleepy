// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

package user

import (
	"github.com/deuill/sleepy/core/user"
)

type GetRequest struct {
	Id      int
	Module  string
	Section string
	Option  string
}

type SetRequest struct {
	Id   int
	Data map[string]map[string]map[string]interface{}
}

func (u *User) GetOption(p GetRequest) (string, error) {
	data, err := user.Get(p.Id)
	if err != nil {
		return "", err
	}

	conf, err := data.Conf(p.Module)
	if err != nil {
		return "", err
	}

	value, err := conf.String(p.Section, p.Option)
	if err != nil {
		return "", err
	}

	return value, nil
}

func (u *User) SetOption(p SetRequest) (bool, error) {
	data, err := user.Get(p.Id)
	if err != nil {
		return false, err
	}

	for module, sections := range p.Data {
		for section, options := range sections {
			for option, value := range options {
				_, err := data.SetOption(module, section, option, value)
				if err != nil {
					return false, err
				}
			}
		}
	}

	return true, nil
}

func (u *User) DeleteOption(p GetRequest) (bool, error) {
	data, err := user.Get(p.Id)
	if err != nil {
		return false, err
	}

	_, err = data.DeleteOption(p.Module, p.Section, p.Option)
	if err != nil {
		return false, err
	}

	return true, nil
}
