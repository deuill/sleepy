// Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
// Use of this source code is governed by the MIT License, the
// full text of which can be found in the LICENSE file.

// Package server contains the core server functionality for Sleepy,
// including the RPC service, the embedded HTTP server for files uploaded
// via Sleepy, and an FTP server, containing a subset of functionality
// required for uploading files to Sleepy in an efficient manner.
package server

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/thoughtmonster/sleepy/core/config"
	"github.com/thoughtmonster/sleepy/core/user"
)

// A table of module methods registered to be called.
var methods map[string]map[string]interface{}

// Request represents the parameters of an RPC call to Sleepy.
type Request struct {
	Module  string      // Module is the name of the module that is to be called.
	Method  string      // Method is the method name to be called.
	Authkey string      // Authkey is the authkey for the connecting user.
	Params  interface{} // Parameters are the RPC method call parameters.
}

// Server is a receiver value for RPC calls from the outside world.
type Server struct{}

// Call calls into module methods and is used as an intermediary between server and modules.
func (s *Server) Call(req *Request, reply *interface{}) error {
	result, err := call(req)
	if err != nil {
		return err
	}

	*reply = result
	return nil
}

func (s *Server) CallMany(req []*Request, reply *interface{}) error {
	var err error
	results := make([]interface{}, len(req))

	for i, r := range req {
		if results[i], err = call(r); err != nil {
			return err
		}
	}

	*reply = results
	return nil
}

func Setup(conf *config.Config) error {
	confdir := conf.S("directories", "config")
	for module := range methods {
		filename := confdir + "/modules.d/" + strings.ToLower(module) + ".conf"
		if _, err := os.Stat(filename); err != nil {
			delete(methods, module)
			continue
		}

		if _, exists := methods[module]["Setup"]; exists {
			method := methods[module]["Setup"].(reflect.Value)

			modconf, err := config.Load(filename)
			if err != nil {
				return fmt.Errorf("Error loading configuration for module '%s': %s", module, err)
			}

			merged, _ := config.Merge(conf, modconf)

			result := method.Call([]reflect.Value{reflect.ValueOf(merged)})
			if err, ok := result[0].Interface().(error); ok {
				return err
			}

			delete(methods[module], "Setup")
		}
	}

	return nil
}

func Register(rcvr interface{}) error {
	r := reflect.ValueOf(rcvr)

	rname := reflect.Indirect(r).Type().Name()
	methods[rname] = make(map[string]interface{}, r.NumMethod())

	for i := 0; i < r.NumMethod(); i++ {
		mname := r.Type().Method(i).Name
		methods[rname][mname] = r.Method(i)
	}

	return nil
}

func call(req *Request) (interface{}, error) {
	// Load and authenticate user against predefined rules.
	_, err := user.Auth(req.Authkey)
	if err != nil {
		return nil, err
	}

	// Validate and call module method.
	if _, exists := methods[req.Module][req.Method]; exists {
		var params []reflect.Value
		method := methods[req.Module][req.Method].(reflect.Value)

		// Validate and prepare parameters for inclusion in call.
		switch p := req.Params.(type) {
		case []interface{}:
			if method.Type().NumIn() != len(p) {
				return nil, fmt.Errorf("Incorrect number of parameters passed to method '%s.%s', expecting %d, given %d.",
					req.Module, req.Method, method.Type().NumIn(), len(p))
			}

			params = make([]reflect.Value, len(p))
			for i, param := range p {
				params[i] = reflect.ValueOf(param)
				if method.Type().In(i).Kind() != params[i].Kind() {
					return nil, fmt.Errorf("Incorrect parameter #%d for method '%s.%s' should be '%s', is '%s'.",
						i, req.Module, req.Method, method.Type().In(i).Kind().String(), params[i].Kind().String())
				}
			}
		case map[string]interface{}:
			if method.Type().NumIn() != 1 {
				return nil, fmt.Errorf("Incorrect number of parameters passed to method '%s.%s', expected single parameter.",
					req.Module, req.Method)
			}

			value := reflect.New(method.Type().In(0)).Elem()
			if err = unpack(p, value); err != nil {
				return nil, err
			}

			params = []reflect.Value{value}
		default:
			return nil, fmt.Errorf("Incorrect parameter types for method '%s.%s'.", req.Module, req.Method)
		}

		result := method.Call(params)
		if len(result) != 2 {
			return nil, fmt.Errorf("Incorrect number of return values for method '%s.%s'.", req.Module, req.Method)
		}

		// Check for error message returned.
		if err, ok := result[1].Interface().(error); ok {
			return nil, err
		}

		return result[0].Interface(), nil
	}

	return nil, fmt.Errorf("Method '%s.%s' does not exist.", req.Module, req.Method)
}

func unpack(data map[string]interface{}, dest reflect.Value) error {
	// Check if we're unpacking to a struct.
	if dest.Kind() != reflect.Struct {
		return fmt.Errorf("Cannot unpack map[string]interface{} to value of type '%s', aborting.",
			dest.Kind().String())
	}

	// Create map of first-level fields in destination struct.
	fields := make(map[string]int, dest.NumField())
	for i := 0; i < dest.NumField(); i++ {
		f := dest.Type().Field(i)
		fields[strings.ToLower(f.Name)] = i
	}

	// Copy fields from map to struct recursively.
	for key, v := range data {
		key = strings.ToLower(key)
		field := dest.Field(fields[key])
		switch value := v.(type) {
		case map[string]interface{}:
			if field.Kind() == reflect.Struct {
				if err := unpack(value, field); err != nil {
					return err
				}
			} else if field.Kind() == reflect.Map && field.Type().Key().Kind() == reflect.String {
				field.Set(reflect.MakeMap(field.Type()))

				for k, val := range value {
					assignMap(k, val, field.Type().Elem(), field)
				}
			} else {
				assign(value, field)
			}
		case []interface{}:
			if field.Kind() != reflect.Slice {
				return fmt.Errorf("Cannot unpack []interface{} to value of type '%s', aborting.",
					field.Kind().String())
			}

			field.Set(reflect.MakeSlice(field.Type(), len(value), cap(value)))

			for i, v := range value {
				switch value := v.(type) {
				case map[string]interface{}:
					t := field.Type().Elem()
					if t.Kind() == reflect.Struct {
						if err := unpack(value, field.Index(i)); err != nil {
							return err
						}
					} else if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
						field.Index(i).Set(reflect.MakeMap(field.Index(i).Type()))

						for k, val := range value {
							assignMap(k, val, t.Elem(), field.Index(i))
						}
					} else {
						assign(value, field.Index(i))
					}
				default:
					assign(value, field.Index(i))
				}
			}
		default:
			assign(value, field)
		}
	}

	return nil
}

func assign(val interface{}, dest reflect.Value) {
	if val != nil {
		v := reflect.ValueOf(val)
		if ok := v.Type().AssignableTo(dest.Type()); ok {
			dest.Set(v)
		} else if ok := v.Type().ConvertibleTo(dest.Type()); ok {
			dest.Set(v.Convert(dest.Type()))
		} else {
			dest.Set(reflect.Zero(dest.Type()))
		}
	}
}

func assignMap(key, val interface{}, elem reflect.Type, dest reflect.Value) {
	if val != nil {
		if v, ok := val.(map[string]interface{}); ok &&
			elem.Kind() == reflect.Map && elem.Key().Kind() == reflect.String {

			dest.SetMapIndex(reflect.ValueOf(key), reflect.MakeMap(elem))

			for k, value := range v {
				assignMap(k, value, elem.Elem(), dest.MapIndex(reflect.ValueOf(key)))
			}
		} else {
			v := reflect.ValueOf(val)
			if ok := v.Type().AssignableTo(elem); ok {
				dest.SetMapIndex(reflect.ValueOf(key), v)
			} else if ok := v.Type().ConvertibleTo(elem); ok {
				dest.SetMapIndex(reflect.ValueOf(key), v.Convert(elem))
			} else {
				dest.SetMapIndex(reflect.ValueOf(key), reflect.Zero(elem))
			}
		}
	}
}

func init() {
	// Initialize the method table.
	methods = make(map[string]map[string]interface{})
}
