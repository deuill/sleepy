// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

package database

import (
	"fmt"
	"strings"
)

func parseSelect(db, tbl string, cols []interface{}) (string, error) {
	var err error
	var c = make([]string, 0)

	for _, name := range cols {
		var col, alias string

		switch n := name.(type) {
		case string:
			col = n
		case map[string]interface{}:
			if len(n) != 1 {
				return "", fmt.Errorf("Field name in SELECT portion is malformed")
			}

			for c, a := range n {
				if _, isStr := a.(string); !isStr {
					return "", fmt.Errorf("Alias name in SELECT portion is malformed")
				}

				col = c
				alias = a.(string)
			}
		default:
			return "", fmt.Errorf("Field name in SELECT portion is malformed")
		}

		fields := strings.Split(col, ".")
		if len(fields) > 2 || len(fields) < 1 {
			return "", fmt.Errorf("Field name in SELECT portion is malformed")
		}

		if len(fields) == 2 {
			if err = checkMeta(db, fields[0], ""); err != nil {
				return "", err
			}

			if fields[1] != "*" {
				if err = checkMeta(db, fields[0], fields[1]); err != nil {
					return "", err
				}

				if alias != "" {
					c = append(c, "`"+fields[0]+"`.`"+fields[1]+"` AS "+alias)
				} else {
					c = append(c, "`"+fields[0]+"`.`"+fields[1]+"`")
				}
			} else {
				c = append(c, "`"+fields[0]+"`.*")
			}
		} else {
			if fields[0] != "*" {
				if err = checkMeta(db, tbl, fields[0]); err != nil {
					return "", err
				}

				if alias != "" {
					c = append(c, "`"+fields[0]+"` AS "+alias)
				} else {
					c = append(c, "`"+fields[0]+"`")
				}
			} else {
				c = append(c, "*")
			}
		}
	}

	return " " + strings.Join(c, ", "), nil
}

func parseJoin(db, tbl string, joins []JoinRequest) (string, []interface{}, error) {
	var err error
	var query string
	var values = make([]interface{}, 0)

	for _, join := range joins {
		// Validate table name.
		if err = checkMeta(db, join.Table, ""); err != nil {
			return "", nil, err
		}

		// Validate conditions.
		c := make([]string, 0)
		for _, condition := range join.Conditions {
			fields := strings.Fields(condition)
			if len(fields) != 3 {
				return "", nil, fmt.Errorf("Condition in JOIN portion is malformed")
			}

			switch fields[1] {
			case "=", "<=>", "!=", "<>", ">", "<", ">=", "<=":
			default:
				return "", nil, fmt.Errorf("Operator '%s' in JOIN condition is invalid", fields[1])
			}

			cond := make([]string, 2)

			for i, val := range []string{fields[0], fields[2]} {
				t, err := column(db, tbl, val)
				if err != nil && i == 1 {
					t = "?"
					values = append(values, strings.Trim(val, "\\\""))
				} else if err != nil {
					return "", nil, fmt.Errorf("Field name in JOIN condition is malformed: %s", err)
				}

				cond[i] = t
			}

			c = append(c, cond[0]+" "+fields[1]+" "+cond[1])
		}

		// Validate JOIN type.
		switch strings.ToUpper(join.Type) {
		case "LEFT", "RIGHT", "OUTER", "INNER", "LEFT OUTER", "RIGHT OUTER":
		default:
			return "", nil, fmt.Errorf("JOIN type '%s' is invalid", join.Type)
		}

		query += " " + strings.ToUpper(join.Type) + " JOIN `" + join.Table + "` ON " + strings.Join(c, " AND ")
	}

	return query, values, nil
}

func parseFilter(db, tbl string, filters []interface{}) (string, []interface{}, error) {
	var err error
	var or, not bool
	var kind, query string

	values := make([]interface{}, 0)

	for _, filter := range filters {
		if k, isStr := filter.(string); isStr {
			switch k {
			case "where", "where-in", "like":
				if query == "" {
					query = " WHERE"
				}

				kind = k
			case "or":
				or = true
			case "not":
				not = true
			default:
				return "", nil, fmt.Errorf("Unexpected type '%s' in WHERE/WHERE IN/LIKE portion", k)
			}
		} else if f, isMap := filter.(map[string]interface{}); isMap {
			switch kind {
			case "where":
				c := make([]string, 0)

				for col, value := range f {
					fields := strings.Fields(col)
					if len(fields) > 2 || len(fields) < 1 {
						return "", nil, fmt.Errorf("Column definition in WHERE portion is malformed")
					}

					col, err = column(db, tbl, fields[0])
					if err != nil {
						return "", nil, fmt.Errorf("Column name in WHERE portion is malformed: %s", err)
					}

					op := "="

					// Validate operator, if any exists.
					if len(fields) == 2 {
						switch fields[1] {
						case "=", "<=>", "!=", "<>", ">", "<", ">=", "<=":
							op = fields[1]
						default:
							return "", nil, fmt.Errorf("Operator '%s' in WHERE portion is invalid", fields[1])
						}
					}

					c = append(c, " ("+col+" "+op+" ?)")
					values = append(values, value)
				}

				if or {
					if query != " WHERE" {
						query += " OR"
					}

					query += strings.Join(c, " OR")
				} else {
					if query != " WHERE" {
						query += " AND"
					}

					query += strings.Join(c, " AND")
				}
			case "where-in":
				if _, isStr := f["column"].(string); !isStr {
					return "", nil, fmt.Errorf("Column in WHERE IN portion is not a string")
				}

				col, err := column(db, tbl, f["column"].(string))
				if err != nil {
					return "", nil, fmt.Errorf("Column name in WHERE IN portion is malformed: %s", err)
				}

				v, isArr := f["in"].([]interface{})
				if !isArr {
					return "", nil, fmt.Errorf("Values in WHERE IN portion are not in an array")
				}

				if query != " WHERE" {
					if or {
						query += " OR"
					} else {
						query += " AND"
					}
				}

				p := strings.TrimRight(strings.Repeat("?, ", len(v)), ", ")

				if not {
					query += " " + col + " NOT IN (" + p + ")"
				} else {
					query += " " + col + " IN (" + p + ")"
				}

				values = append(values, v...)
			case "like":
				c := make([]string, 0)

				for col, value := range f {
					col, err = column(db, tbl, col)
					if err != nil {
						return "", nil, fmt.Errorf("Column name in WHERE LIKE portion is malformed: %s", err)
					}

					if not {
						c = append(c, " "+col+" NOT LIKE ?")
					} else {
						c = append(c, " "+col+" LIKE ?")
					}

					values = append(values, value)
				}

				if or {
					if query != " WHERE" {
						query += " OR"
					}

					query += strings.Join(c, " OR")
				} else {
					if query != " WHERE" {
						query += " AND"
					}

					query += strings.Join(c, " AND")
				}
			default:
				return "", nil, fmt.Errorf("Unexpected type '%s' in WHERE/WHERE IN/LIKE portion", kind)
			}

			kind = ""
			or = false
			not = false
		} else {
			return "", nil, fmt.Errorf("Unexpected argument in WHERE/WHERE IN/LIKE portion")
		}
	}

	return query, values, nil
}

func parseHaving(db, tbl string, filters []interface{}) (string, []interface{}, error) {
	var err error
	var or bool

	query := " HAVING"
	values := make([]interface{}, 0)

	for _, filter := range filters {
		switch f := filter.(type) {
		case string:
			if f == "or" {
				or = true
			} else {
				return "", nil, fmt.Errorf("Unexpected type '%s' in HAVING portion", f)
			}
		case map[string]interface{}:
			c := make([]string, 0)

			for col, value := range f {
				fields := strings.Fields(col)
				if len(fields) > 2 || len(fields) < 1 {
					return "", nil, fmt.Errorf("Column definition in HAVING portion is malformed")
				}

				col, err = column(db, tbl, fields[0])
				if err != nil {
					return "", nil, fmt.Errorf("Column name in WHERE portion is malformed: %s", err)
				}

				op := "="

				// Validate operator, if any exists.
				if len(fields) == 2 {
					switch fields[1] {
					case "=", "<=>", "!=", "<>", ">", "<", ">=", "<=":
						op = fields[1]
					default:
						return "", nil, fmt.Errorf("Operator '%s' in HAVING portion is invalid", fields[1])
					}
				}

				c = append(c, " ("+col+" "+op+" ?)")
				values = append(values, value)
			}

			if or {
				if query != " HAVING" {
					query += " OR"
				}

				query += strings.Join(c, " OR")
			} else {
				if query != " HAVING" {
					query += " AND"
				}

				query += strings.Join(c, " AND")
			}

			or = false
		default:
			return "", nil, fmt.Errorf("Unexpected argument in HAVING portion")
		}
	}

	return query, values, nil
}

func parseOrderBy(db, tbl string, orders []OrderRequest) (string, error) {
	c := make([]string, 0)

	for _, order := range orders {
		col, err := column(db, tbl, order.Column)
		if err != nil {
			return "", fmt.Errorf("Column name in ORDER BY condition is malformed: %s", err)
		}

		switch strings.ToUpper(order.Order) {
		case "ASC", "DESC", "RANDOM":
		default:
			return "", fmt.Errorf("Invalid order type '%s' in ORDER BY condition", order.Order)
		}

		c = append(c, col+" "+strings.ToUpper(order.Order))
	}

	return " ORDER BY " + strings.Join(c, ", "), nil
}

func column(db, tbl, col string) (string, error) {
	var err error

	split := strings.Split(col, ".")
	if len(split) > 2 || len(split) < 1 {
		return "", fmt.Errorf("Column definition is incorrect")
	}

	if len(split) == 2 {
		if err = checkMeta(db, split[0], ""); err != nil {
			return "", err
		} else if err = checkMeta(db, split[0], split[1]); err != nil {
			return "", err
		}

		return "`" + split[0] + "`.`" + split[1] + "`", nil
	} else if err = checkMeta(db, tbl, split[0]); err != nil {
		return "", err
	} else {
		return "`" + split[0] + "`", nil
	}
}
