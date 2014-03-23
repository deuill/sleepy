// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

// Package config implements configuration file support for Sleepy.
// The files are loaded using Load and parsed using the various parsing
// methods.
package config

import (
	"fmt"
	"strconv"

	"github.com/miguel-branco/goconfig"
)

// Config represents a parsed configuration file.
type Config map[string]map[string]interface{}

// String returns a value of 'option' in 'section' as a string. If the
// value cannot be found, or it cannot be converted into a string, an
// empty string is returned along with an error.
func (c *Config) String(section, option string) (string, error) {
	if exists, error := c.exists(section, option); !exists {
		return "", error
	}

	// Return value as string, after conversion, if necessary.
	switch value := (*c)[section][option].(type) {
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	case int:
		return strconv.Itoa(value), nil
	case bool:
		return strconv.FormatBool(value), nil
	case float64:
		return strconv.FormatFloat(value, 'g', -1, 64), nil
	case nil:
		return "", nil
	default:
		return "", fmt.Errorf("Couldn't convert type '%T' to string", value)
	}

	return "", fmt.Errorf("Unknown parse error")
}

// S is like String, but never returns an error.
func (c *Config) S(section, option string) string {
	v, _ := c.String(section, option)
	return v
}

// Int returns a value of 'option' in 'section' as an integer. It returns
// a zero (0) integer along with an error message on failure.
func (c *Config) Int(section, option string) (int64, error) {
	if exists, error := c.exists(section, option); !exists {
		return 0, error
	}

	// Return value as int, after conversion, if necessary.
	switch value := (*c)[section][option].(type) {
	case string:
		num, error := strconv.ParseInt(value, 10, 64)
		if error != nil {
			return 0, fmt.Errorf("Couldn't convert string '%s' to int", value)
		}

		return num, nil
	case int:
		return int64(value), nil
	case bool:
		if value {
			return 1, nil
		}

		return 0, nil
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("Couldn't convert type '%T' to int", value)
	}

	return 0, fmt.Errorf("Unknown parse error")
}

// I is like Int, but never returns an error.
func (c *Config) I(section, option string) int64 {
	v, _ := c.Int(section, option)
	return v
}

// Bool returns a value of 'option' in 'section' as an boolean. It returns
// 'false' along with an error message on failure.
func (c *Config) Bool(section, option string) (bool, error) {
	if exists, error := c.exists(section, option); !exists {
		return false, error
	}

	// Return value as bool, after conversion, if necessary.
	switch value := (*c)[section][option].(type) {
	case string:
		status, error := strconv.ParseBool(value)
		if error != nil {
			return false, fmt.Errorf("Couldn't convert string '%s' to bool", value)
		}

		return status, nil
	case bool:
		return value, nil
	case nil:
		return false, nil
	default:
		return false, fmt.Errorf("Couldn't convert type '%T' to bool", value)
	}

	return false, fmt.Errorf("Unknown parse error")
}

// B is like Bool, but never returns an error.
func (c *Config) B(section, option string) bool {
	v, _ := c.Bool(section, option)
	return v
}

// Float returns a value of 'option' in 'section' as an float64. It returns
// 0.0 along with an error message on failure.
func (c *Config) Float(section, option string) (float64, error) {
	if exists, error := c.exists(section, option); !exists {
		return 0.0, error
	}

	// Return value as float, after conversion, if necessary.
	switch value := (*c)[section][option].(type) {
	case string:
		num, error := strconv.ParseFloat(value, 64)
		if error != nil {
			return 0.0, fmt.Errorf("Couldn't convert string '%s' to float", value)
		}

		return num, nil
	case float64:
		return value, nil
	case nil:
		return 0.0, nil
	default:
		return 0.0, fmt.Errorf("Couldn't convert type '%T' to float", value)
	}

	return 0.0, fmt.Errorf("Unknown parse error")
}

// F is like Float, but never returns an error.
func (c *Config) F(section, option string) float64 {
	v, _ := c.Float(section, option)
	return v
}

// Merge merges two or more configuration files, and returns a new,
// combined *Config type. Identical sections and options are overwritten
// according to the order of definition.
func Merge(conf ...*Config) (*Config, error) {
	c := new(Config)
	data := make(map[string]map[string]interface{})

	for _, current := range conf {
		if current == nil {
			continue
		}

		for key, value := range *current {
			data[key] = make(map[string]interface{})
			data[key] = value
		}
	}

	*c = data

	return c, nil
}

// Load reads the configuration file located in 'conf' and returns
// a new Config type. If the configuration file cannot be found,
// the function returns nil and an error message.
func Load(conf string) (*Config, error) {
	var error error

	c := new(Config)

	*c, error = parse(conf)
	if error != nil {
		return nil, error
	}

	return c, nil
}

// Checks if 'option' exists under 'section' in the Config file.
func (c *Config) exists(section, option string) (bool, error) {
	if c == nil {
		return false, fmt.Errorf("config is invalid")
	}

	if _, ok := (*c)[section]; !ok {
		return false, fmt.Errorf("section '%s' not found", section)
	}

	if _, ok := (*c)[section][option]; !ok {
		return false, fmt.Errorf("option '%s' not found", option)
	}

	return true, nil
}

// Parse parses the configuration file in 'conf' and returns the data
// as values mapped to options, mapped to sections.
func parse(conf string) (map[string]map[string]interface{}, error) {
	c, err := goconfig.ReadConfigFile(conf)
	if err != nil {
		return nil, err
	}

	data := make(map[string]map[string]interface{})

	for _, section := range c.GetSections() {
		data[section] = make(map[string]interface{})
		s, _ := c.GetOptions(section)

		for _, option := range s {
			v, err := c.GetString(section, option)
			if err != nil {
				return nil, err
			}

			data[section][option] = v
		}
	}

	return data, nil
}
