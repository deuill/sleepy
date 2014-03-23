// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

// Package user contains functionality for querying, adding and removing
// users, or consumers of services provided by modules via RPC.
package user

import (
	"crypto/rand"
	"crypto/sha1"
	"database/sql"
	"fmt"
	"io"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type User struct {
	// Contains private or unexported fields.
	Id      int
	Authkey string
}

func Get(id int) (*User, error) {
	user := new(User)

	query := `SELECT id, authkey FROM users WHERE id = ?`
	error := db.QueryRow(query, id).Scan(&user.Id, &user.Authkey)
	if error != nil {
		return nil, fmt.Errorf("User with id '%d' not found.", id)
	}

	return user, nil
}

func Save() (*User, error) {
	// Generate random SHA1 authkey.
	buf := sha1.New()

	_, error := io.CopyN(buf, rand.Reader, 20)
	if error != nil {
		return nil, error
	}

	auth := fmt.Sprintf("%x", buf.Sum(nil))

	// Create user.
	query := `INSERT INTO users (authkey) VALUES (?)`
	result, error := db.Exec(query, auth)
	if error != nil {
		return nil, error
	}

	var id int64

	if id, error = result.LastInsertId(); error != nil {
		return nil, error
	}

	var user = &User{
		int(id),
		auth,
	}

	return user, nil
}

func Remove(id int) (bool, error) {
	var exists int

	// Check if user already exists.
	query := `SELECT id FROM users WHERE id = ?`
	db.QueryRow(query, id).Scan(&exists)

	if exists == 0 {
		return false, fmt.Errorf("User does not exist")
	}

	// Delete user.
	query = `DELETE FROM users WHERE id = ?`
	_, error := db.Exec(query, id)
	if error != nil {
		return false, error
	}

	// Delete user options.
	query = `DELETE FROM user_conf WHERE user_id = ?`
	_, error = db.Exec(query, id)
	if error != nil {
		return false, error
	}

	return true, nil
}

func List() ([]User, error) {
	// Execute query.
	rows, error := db.Query(`SELECT id, authkey FROM users ORDER BY id ASC`)
	if error != nil {
		return nil, fmt.Errorf("Error fetching user list: %s", error)
	}

	users := make([]User, 0)

	for i := 0; rows.Next(); i++ {
		user := User{}
		rows.Scan(&user.Id, &user.Authkey)
		users = append(users, user)
	}

	if error = rows.Err(); error != nil {
		return nil, error
	}

	return users, nil
}

func Setup(datadir, filename string) error {
	var error error

	// Connect to system database.
	db, error = sql.Open("sqlite3", datadir+"/"+filename)
	if error != nil {
		return fmt.Errorf("Error initializing database: %s\n", error)
	}

	return nil
}
