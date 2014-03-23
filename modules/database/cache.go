// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/bradfitz/gomemcache/memcache"
)

var metaCache struct {
	sync.Mutex
	data map[string]map[string]map[string]bool
}

var dataCache *memcache.Client

// Check cache for request with signature 'sig' and return data if cached entity exists.
func getCache(sig string) []map[string]interface{} {
	if item, error := dataCache.Get("sleepy/database/" + sig); error == nil {
		var data []map[string]interface{}
		json.Unmarshal(item.Value, &data)

		return data
	}

	return nil
}

func storeCache(data []map[string]interface{}, sig, database, table string) {
	var items []string

	buf, _ := json.Marshal(data)
	dataCache.Set(&memcache.Item{Key: "sleepy/database/" + sig, Value: buf})

	if item, error := dataCache.Get("sleepy/database/" + database + "." + table); error == nil {
		json.Unmarshal(item.Value, &items)
	}

	items = append(items, sig)

	buf, _ = json.Marshal(items)
	dataCache.Set(&memcache.Item{Key: "sleepy/database/" + database + "." + table, Value: buf})
}

func clearCache(database, table string) {
	var items []string

	if item, error := dataCache.Get("sleepy/database/" + database + "." + table); error == nil {
		json.Unmarshal(item.Value, &items)

		for _, sig := range items {
			dataCache.Delete("sleepy/database/" + sig)
		}

		dataCache.Delete("sleepy/database/" + database + "." + table)
	}
}

func getMetaCache(db *sql.DB, database, table string) error {
	var value string
	databases := make([]string, 0)
	tables := make([]string, 0)

	metaCache.Lock()
	defer metaCache.Unlock()

	if database == "" {
		for dt := range metaCache.data {
			delete(metaCache.data, dt)
		}

		dbs, error := db.Query("SHOW DATABASES WHERE `Database` != 'information_schema' AND `Database` != 'performance_schema'")
		if error != nil {
			return fmt.Errorf("Error executing query: %s", error)
		}

		for dbs.Next() {
			dbs.Scan(&value)

			metaCache.data[value] = make(map[string]map[string]bool)
			databases = append(databases, value)
		}
	} else {
		if _, exists := metaCache.data[database]; !exists {
			return fmt.Errorf("Invalid database name specified: '%s'", database)
		}

		databases = append(databases, database)
	}

	for _, dt := range databases {
		if table == "" {
			for tt := range metaCache.data[dt] {
				delete(metaCache.data[dt], tt)
			}

			tbls, error := db.Query("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?", dt)
			if error != nil {
				return fmt.Errorf("Error executing query: %s", error)
			}

			for tbls.Next() {
				tbls.Scan(&value)

				metaCache.data[dt][value] = make(map[string]bool)
				tables = append(tables, value)
			}
		} else {
			if _, exists := metaCache.data[dt][table]; !exists {
				return fmt.Errorf("Invalid table name specified: '%s.%s'", dt, table)
			}

			tables = append(tables, table)
		}

		for _, tt := range tables {
			for ct := range metaCache.data[dt][tt] {
				delete(metaCache.data[dt][tt], ct)
			}

			cols, error := db.Query("SELECT COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?", dt, tt)
			if error != nil {
				return fmt.Errorf("Error executing query: %s", error)
			}

			for cols.Next() {
				cols.Scan(&value)
				metaCache.data[dt][tt][value] = true
			}
		}
	}

	return nil
}

func checkMeta(database, table, column string) error {
	if database != "" && table == "" && column == "" {
		if _, exists := metaCache.data[database]; !exists {
			return fmt.Errorf("Invalid database name specified: '%s'", database)
		}
	} else if database != "" && table != "" && column == "" {
		if _, exists := metaCache.data[database][table]; !exists {
			return fmt.Errorf("Invalid table name specified: '%s.%s'", database, table)
		}
	} else if database != "" && table != "" && column != "" {
		if _, exists := metaCache.data[database][table][column]; !exists {
			return fmt.Errorf("Invalid column name specified: '%s.%s.%s'", database, table, column)
		}
	}

	return nil
}
