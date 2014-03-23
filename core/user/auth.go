// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

package user

import (
	"fmt"
)

func Auth(authkey string) (*User, error) {
	user := new(User)

	query := `SELECT id, authkey FROM users WHERE authkey = ?`
	error := db.QueryRow(query, authkey).Scan(&user.Id, &user.Authkey)
	if error != nil {
		return nil, fmt.Errorf("User with authkey '%s' did not authenticate: %s", authkey, error)
	}

	return user, nil
}
