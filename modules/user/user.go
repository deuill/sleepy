// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

// Package user contains methods for retrieving, adding and removing
// users, or API consumers, to Sleepy, via RPC.
package user

import (
	"github.com/thoughtmonster/sleepy/core/server"
	"github.com/thoughtmonster/sleepy/core/user"
)

type User struct {}

func (u *User) Auth(authkey string) (interface{}, error) {
	result, err := user.Auth(authkey)
	if err != nil {
		return false, err
	}

	return result, nil
}

func (u *User) Get(id float64) (interface{}, error) {
	result, err := user.Get(int(id))
	if err != nil {
		return false, err
	}

	return result, nil
}

func (u *User) Save() (interface{}, error) {
	result, err := user.Save()
	if err != nil {
		return false, err
	}

	return result, nil
}

func (u *User) Remove(id float64) (bool, error) {
	result, err := user.Remove(int(id))
	if err != nil {
		return false, err
	}

	return result, nil
}

func (u *User) List() (interface{}, error) {
	result, err := user.List()
	if err != nil {
		return false, err
	}

	return result, nil
}

func init() {
	server.Register(&User{})
}
