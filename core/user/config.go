// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

package user

import (
	"github.com/thoughtmonster/sleepy/core/config"
)

func (u *User) Conf(module string) (*config.Config, error) {
	query := `SELECT section, option, value FROM user_conf WHERE user_id = ? AND module = ?`
	rows, error := db.Query(query, u.Id, module)
	if error != nil {
		return nil, error
	}

	defer rows.Close()

	conf := make(config.Config)

	for rows.Next() {
		var section, option string
		var value interface{}

		error = rows.Scan(&section, &option, &value)
		if error != nil {
			return nil, error
		}

		conf[section] = make(map[string]interface{})
		conf[section][option] = value
	}

	return &conf, nil
}

func (u *User) SetOption(module, section, option string, value interface{}) (bool, error) {
	var exists int

	query := `SELECT EXISTS(SELECT 1 FROM user_conf WHERE user_id = ? AND module = ? AND section = ? AND option = ?)`
	row := db.QueryRow(query, u.Id, module, section, option)
	row.Scan(&exists)

	if exists > 0 {
		query = `UPDATE user_conf SET value = ? WHERE user_id = ? AND module = ? AND section = ? AND option = ?`
	} else {
		query = `INSERT INTO user_conf (value, user_id, module, section, option) VALUES (?, ?, ?, ?, ?)`
	}

	_, error := db.Exec(query, value, u.Id, module, section, option)
	if error != nil {
		return false, error
	}

	return true, nil
}

func (u *User) DeleteOption(module, section, option string) (bool, error) {
	params := make([]interface{}, 1)
	query := `DELETE FROM user_conf WHERE user_id = ?`

	params[0] = u.Id

	if module != "" {
		query = query + ` AND module = ?`
		params = append(params, module)
	}

	if section != "" {
		query = query + ` AND section = ?`
		params = append(params, section)
	}

	if option != "" {
		query = query + ` AND option = ?`
		params = append(params, option)
	}

	_, error := db.Exec(query, params...)
	if error != nil {
		return false, error
	}

	return true, nil
}
