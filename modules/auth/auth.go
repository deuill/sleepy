// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

// Package auth provides methods for generating and validating passwords
// for consumption via RPC.
package auth

import (
	"code.google.com/p/go.crypto/bcrypt"
	"github.com/deuill/sleepy/core/server"
)

type Auth struct {}

func (a *Auth) GeneratePassword(passwd string) (string, error) {
	// Hash password with bcrypt.
	hash, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	if err != nil {
		return "", nil
	}

	return string(hash), nil
}

func (a *Auth) ValidatePassword(passwd, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(passwd))
	if err != nil {
		return false, nil
	}

	return true, nil
}

func init() {
	server.Register(&Auth{})
}
