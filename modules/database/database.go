// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

// Package database provides methods for querying an SQL database
// through a stable interface.
package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/bradfitz/gomemcache/memcache"
	_ "github.com/go-sql-driver/mysql"
	"github.com/deuill/sleepy/core/config"
	"github.com/deuill/sleepy/core/server"
	"github.com/deuill/sleepy/core/user"
)

type Database struct {
	// Contains private or unexported fields.
	conf   *config.Config
	conn   map[string]*sql.DB
	client map[string]*config.Config
}

type Request struct {
	Sig        string
	Auth       string
	Db         string
	Table      string
	Select     []interface{}
	Distinct   bool
	Join       []JoinRequest
	Filter     []interface{}
	Group      []string
	Having     []interface{}
	Order      []OrderRequest
	Limit      int64
	Offset     int64
	Data       map[string]interface{}
	Query      string
	Parameters []interface{}
}

type JoinRequest struct {
	Table      string
	Conditions []string
	Type       string
}

type OrderRequest struct {
	Column string
	Order  string
}

func (d *Database) Get(p Request) (interface{}, error) {
	if result := getCache(p.Sig); result != nil {
		return result, nil
	}

	db, error := d.prepare(&p)
	if error != nil {
		return false, error
	}

	values := make([]interface{}, 0)
	query := "SELECT"

	// Process DISTINCT portion.
	if p.Distinct {
		query += " DISTINCT"
	}

	// Process SELECT portion.
	if p.Select != nil {
		subquery, err := parseSelect(p.Db, p.Table, p.Select)
		if err != nil {
			return false, err
		}

		query += subquery
	} else {
		query += " *"
	}

	// Process table name.
	query += " FROM `" + p.Db + "`.`" + p.Table + "`"

	// Process JOIN portion.
	if p.Join != nil {
		subquery, subvalues, err := parseJoin(p.Db, p.Table, p.Join)
		if err != nil {
			return false, err
		}

		query += subquery
		values = append(values, subvalues...)
	}

	// Process WHERE, WHERE IN and LIKE portions.
	if p.Filter != nil {
		subquery, subvalues, err := parseFilter(p.Db, p.Table, p.Filter)
		if err != nil {
			return false, err
		}

		query += subquery
		values = append(values, subvalues...)
	}

	// Process GROUP BY portion.
	if p.Group != nil {
		c := make([]string, 0)

		for _, column := range p.Group {
			if error = checkMeta(p.Db, p.Table, column); error != nil {
				return false, error
			}

			c = append(c, "`"+column+"`")
		}

		query += " GROUP BY " + strings.Join(c, ", ")
	}

	// Process HAVING portion.
	if p.Having != nil {
		subquery, subvalues, err := parseHaving(p.Db, p.Table, p.Having)
		if err != nil {
			return false, err
		}

		query += subquery
		values = append(values, subvalues...)
	}

	// Process ORDER BY portion.
	if p.Order != nil {
		subquery, err := parseOrderBy(p.Db, p.Table, p.Order)
		if err != nil {
			return false, err
		}

		query += subquery
	}

	// Process LIMIT portion.
	if p.Limit != 0 {
		limit := strconv.FormatInt(p.Limit, 10)

		if p.Offset != 0 {
			offset := strconv.FormatInt(p.Offset, 10)

			query += " LIMIT " + limit + " OFFSET " + offset
		} else {
			query += " LIMIT " + limit
		}
	}

	// Execute query and return results.
	result, error := d.query(db, query, values)
	if error != nil {
		return false, error
	}

	storeCache(result, p.Sig, p.Db, p.Table)

	return result, nil
}

func (d *Database) Put(p Request) (interface{}, error) {
	db, error := d.prepare(&p)
	if error != nil {
		return false, error
	}

	values := make([]interface{}, 0)

	var query string
	columns := make([]string, 0)

	if p.Filter != nil {
		query = "UPDATE "
	} else {
		query = "INSERT INTO "
	}

	// Process table name.
	query += "`" + p.Db + "`.`" + p.Table + "` "

	// Process data for INSERT/UPDATE query.
	for column, value := range p.Data {
		if error = checkMeta(p.Db, p.Table, column); error != nil {
			return false, error
		}

		columns = append(columns, column)
		values = append(values, value)
	}

	if p.Filter != nil {
		query += "SET "

		for _, column := range columns {
			query += "`" + column + "` = ?, "
		}

		query = strings.TrimRight(query, ", ")

		subquery, subvalues, err := parseFilter(p.Db, p.Table, p.Filter)
		if err != nil {
			return false, err
		}

		query += subquery
		values = append(values, subvalues...)
	} else {
		vals := strings.TrimRight(strings.Repeat("?, ", len(columns)), ", ")
		query += "(`" + strings.Join(columns, "`, `") + "`) VALUES (" + vals + ")"
	}

	// Execute query.
	result, error := d.exec(db, query, values)
	if error != nil {
		return false, error
	}

	clearCache(p.Db, p.Table)

	return result, nil
}

func (d *Database) Delete(p Request) (interface{}, error) {
	db, error := d.prepare(&p)
	if error != nil {
		return false, error
	}

	var query string
	values := make([]interface{}, 0)

	// Process table name.
	query = "DELETE FROM `" + p.Db + "`.`" + p.Table + "`"

	// Process WHERE portion.
	if p.Filter != nil {
		subquery, subvalues, err := parseFilter(p.Db, p.Table, p.Filter)
		if err != nil {
			return false, err
		}

		query += subquery
		values = append(values, subvalues...)
	} else {
		return false, fmt.Errorf("No WHERE clause found")
	}

	// Execute query.
	result, error := d.exec(db, query, values)
	if error != nil {
		return false, error
	}

	clearCache(p.Db, p.Table)

	return result, nil
}

func (d *Database) Query(p Request) (interface{}, error) {
	db, error := d.prepare(&p)
	if error != nil {
		return false, error
	}

	var result interface{}
	action := strings.ToUpper(strings.Fields(p.Query)[0])

	switch action {
	case "SELECT":
		result, error = d.query(db, p.Query, p.Parameters)
		if error != nil {
			return false, error
		}
	default:
		result, error = d.exec(db, p.Query, p.Parameters)
		if error != nil {
			return false, error
		}
	}
	// Invalidate cache under certain circumstances.
	switch action {
	case "ALTER":
		if p.Table != "" {
			getMetaCache(db, p.Db, p.Table)
		}
	case "CREATE", "DROP":
		getMetaCache(db, "", "")
	case "INSERT", "UPDATE", "DELETE":
		clearCache(p.Db, p.Table)
	}

	return result, nil
}

// Execute query string on db with optional params in place of positional
// parameters. Returns results as rows of columns mapped to values.
func (d *Database) query(db *sql.DB, query string, params []interface{}) ([]map[string]interface{}, error) {
	// Execute query.
	rows, error := db.Query(query, params...)
	if error != nil {
		return nil, fmt.Errorf("Error executing query: %s", error)
	}

	// Process query results
	columns, error := rows.Columns()
	if error != nil {
		return nil, fmt.Errorf("Error fetching column list: %s", error)
	}

	results := make([]map[string]interface{}, 0)

	for c := 0; rows.Next(); c++ {
		var row = make([]interface{}, len(columns))

		results = append(results, make(map[string]interface{}))
		for i := range columns {
			row[i] = new(interface{})
		}

		rows.Scan(row...)
		for i, value := range row {
			v := reflect.ValueOf(value).Elem().Elem()

			// Convert []byte to string.
			if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
				results[c][columns[i]] = string(v.Bytes())
			} else {
				results[c][columns[i]] = *value.(*interface{})
			}
		}
	}

	if error = rows.Err(); error != nil {
		return nil, error
	}

	return results, nil
}

// Execute query string on db with optional params in place of positional
// parameters. Returns the last inserted ID in case of an INSERT query, and
// the number of rows affected in any other query.
func (d *Database) exec(db *sql.DB, query string, params []interface{}) (interface{}, error) {
	summary, error := db.Exec(query, params...)
	if error != nil {
		return nil, fmt.Errorf("Error executing query: %s", error)
	}

	var result interface{}

	switch strings.ToUpper(strings.Fields(query)[0]) {
	case "INSERT":
		result, error = summary.LastInsertId()
		if error != nil {
			return nil, error
		}
	default:
		result, error = summary.RowsAffected()
		if error != nil {
			return nil, error
		}
	}

	return result, nil
}

// Parse configuration, connect to database and validate data
func (d *Database) prepare(p *Request) (*sql.DB, error) {
	var error error

	// Load configuration for user.
	u, error := user.Auth(p.Auth)
	if error != nil {
		return nil, error
	}

	c, error := u.Conf("database")
	if error != nil {
		return nil, error
	}

	d.client[p.Auth] = c
	name, _ := d.client[p.Auth].String("database", "name")

	// Connect to database, if no connection exists.
	if _, exists := d.conn[name]; !exists {
		db, error := d.connect(name)
		if error != nil {
			return nil, error
		}

		d.conn[name] = db
	}

	// Extract database/table name from query.
	if p.Query != "" {
		split := strings.Fields(p.Query)
		stmt := strings.ToUpper(split[0])
		switch stmt {
		case "DELETE", "INSERT", "UPDATE":
			var token string
			var pos, offset int
			if stmt == "DELETE" {
				token = "FROM"
				offset = 1
			} else if stmt == "INSERT" {
				token = "INTO"
				offset = 1
			} else {
				token = "SET"
				offset = -1
			}

			for i, value := range split {
				if strings.ToUpper(value) == token {
					pos = i
					break
				}
			}

			if pos > 0 {
				p.Table = split[pos+offset]
			}
		case "CREATE", "ALTER", "DROP":
			var pos int
			var obj string
			for i, value := range split {
				value = strings.ToUpper(value)
				if value == "TABLE" || value == "DATABASE" {
					pos = i
					obj = value
					break
				}
			}

			if stmt == "CREATE" && obj == "TABLE" {
				s := strings.Split(split[pos+1], ".")
				if len(s) > 2 || len(s) < 1 {
					return nil, fmt.Errorf("Table name is malformed")
				}

				if len(split) == 2 {
					p.Db = s[0]
				} else {
					p.Db = name
				}

				if error = checkMeta(p.Db, "", ""); error != nil {
					return nil, error
				}
			} else if obj == "TABLE" {
				p.Table = split[pos+1]
			} else if obj == "DATABASE" {
				p.Db = split[pos+1]

				if stmt == "ALTER" || stmt == "DROP" {
					if error = checkMeta(p.Db, "", ""); error != nil {
						return nil, error
					}
				}
			}
		}
	}

	// Validate table name against database tables.
	if p.Table != "" {
		split := strings.Split(p.Table, ".")
		if len(split) > 2 || len(split) < 1 {
			return nil, fmt.Errorf("Table name is malformed")
		}

		if len(split) == 2 {
			p.Db = split[0]
			p.Table = split[1]

			// Validate table name against database tables.
			if error = checkMeta(split[0], split[1], ""); error != nil {
				return nil, error
			}
		} else {
			p.Db = name

			// Validate table name against database tables.
			if error = checkMeta(p.Db, p.Table, ""); error != nil {
				return nil, error
			}
		}
	}

	return d.conn[name], nil
}

// Connect to database 'db'
func (d *Database) connect(db string) (*sql.DB, error) {
	addr, _ := d.conf.String("mysql", "address")
	port, _ := d.conf.String("mysql", "port")
	uname, _ := d.conf.String("mysql", "username")
	pass, _ := d.conf.String("mysql", "password")

	conn, error := sql.Open("mysql", uname+":"+pass+"@tcp("+addr+":"+port+")/"+db+"?charset=utf8")
	if error != nil {
		return nil, fmt.Errorf("Error connecting to database: %s\n", error)
	}

	error = getMetaCache(conn, "", "")
	if error != nil {
		return nil, error
	}

	return conn, nil
}

func (d *Database) Setup(config *config.Config) error {
	d.conf = config

	// Initialize memcache client.
	address, _ := config.String("memcache", "address")
	port, _ := config.String("memcache", "port")
	dataCache = memcache.New(address + ":" + port)

	// Initialize metadata cache.
	metaCache.data = make(map[string]map[string]map[string]bool)

	return nil
}

func init() {
	server.Register(&Database{
		&config.Config{},
		make(map[string]*sql.DB),
		make(map[string]*config.Config),
	})
}
