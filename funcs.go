package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func getFuncMap(t *template.Template) template.FuncMap {
	f := sprig.TxtFuncMap()
	delete(f, "expandenv")

	f["include"] = include(t)
	f["shell"] = shell
	f["toYaml"] = toYaml
	f["key"] = key
	f["keyOrDefault"] = keyOrDefault
	f["ls"] = ls
	f["service"] = service
	f["services"] = services
	f["parseInt"] = parseInt
	f["add"] = add
	f["subtract"] = subtract
	f["multiply"] = multiply
	f["divide"] = divide
	f["modulo"] = modulo
	f["minimum"] = minimum
	f["maximum"] = maximum

	return f
}

func include(t *template.Template) func(templateName string, vars ...interface{}) (string, error) {
	return func(templateName string, vars ...interface{}) (string, error) {
		if len(vars) > 1 {
			return "", errors.New(fmt.Sprintf("Call to include may pass zero or one vars structure, got %v.", len(vars)))
		}
		buf := bytes.NewBuffer(nil)
		included := t.Lookup(templateName)
		if included == nil {
			return "", errors.New(fmt.Sprintf("No such template '%v' found while calling 'include'.", templateName))
		}

		if err := included.ExecuteTemplate(buf, templateName, vars[0]); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
}

func shell(cmd ...string) (string, error) {
	out, err := exec.Command("bash", "-c", strings.Join(cmd[:], "")).Output()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return "", errors.Wrap(err, "Issue running command: "+output)
	}

	return output, nil
}

func toYaml(v interface{}) (string, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return "", errors.Wrap(err, "Issue marsahling yaml")
	}
	return string(data), nil
}

func key(k string) string {
	return os.Getenv(k)
}

func keyOrDefault(k, d string) string {
	if v, e := os.LookupEnv(k); e {
		return v
	}
	return d
}

func service(k string) []interface{} {
	var matches []interface{}
	ss := services()
	for i := range ss {
		if s, ok := ss[i]["Name"]; ok && s == k {
			matches = append(matches, ss[i])
		}
	}
	return matches
}

func services() []map[string]interface{} {
	var m []map[string]interface{}
	json.Unmarshal([]byte(os.Getenv("services")), &m)
	return m
}

type KVP struct {
	Key, Value string
}

func ls(k string) (m []KVP) {
	vars := os.Environ()
	for i := range vars {
		if strings.HasPrefix(vars[i], k) {
			k, v, _ := strings.Cut(vars[i], "=")
			_, k = path.Split(k)
			m = append(m, KVP{k, v})
		}
	}
	return m
}

// copied from https://github.com/hashicorp/consul-template/blob/8a02e2ac7fa33c5a9f5a093cdcf65691ab673f1f/template/funcs.go#L1043
// under MPL
// Copyright (c) 2014 HashiCorp, Inc.
// parseInt parses a string into a base 10 int
func parseInt(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}

	result, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "parseInt")
	}
	return result, nil
}

// add returns the sum of a and b.
func add(b, a interface{}) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Int() + bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Int() + int64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Int()) + bv.Float(), nil
		default:
			return nil, fmt.Errorf("add: unknown type for %q (%T)", bv, b)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int64(av.Uint()) + bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Uint() + bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Uint()) + bv.Float(), nil
		default:
			return nil, fmt.Errorf("add: unknown type for %q (%T)", bv, b)
		}
	case reflect.Float32, reflect.Float64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Float() + float64(bv.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Float() + float64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return av.Float() + bv.Float(), nil
		default:
			return nil, fmt.Errorf("add: unknown type for %q (%T)", bv, b)
		}
	default:
		return nil, fmt.Errorf("add: unknown type for %q (%T)", av, a)
	}
}

// subtract returns the difference of b from a.
func subtract(b, a interface{}) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Int() - bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Int() - int64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Int()) - bv.Float(), nil
		default:
			return nil, fmt.Errorf("subtract: unknown type for %q (%T)", bv, b)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int64(av.Uint()) - bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Uint() - bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Uint()) - bv.Float(), nil
		default:
			return nil, fmt.Errorf("subtract: unknown type for %q (%T)", bv, b)
		}
	case reflect.Float32, reflect.Float64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Float() - float64(bv.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Float() - float64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return av.Float() - bv.Float(), nil
		default:
			return nil, fmt.Errorf("subtract: unknown type for %q (%T)", bv, b)
		}
	default:
		return nil, fmt.Errorf("subtract: unknown type for %q (%T)", av, a)
	}
}

// multiply returns the product of a and b.
func multiply(b, a interface{}) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Int() * bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Int() * int64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Int()) * bv.Float(), nil
		default:
			return nil, fmt.Errorf("multiply: unknown type for %q (%T)", bv, b)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int64(av.Uint()) * bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Uint() * bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Uint()) * bv.Float(), nil
		default:
			return nil, fmt.Errorf("multiply: unknown type for %q (%T)", bv, b)
		}
	case reflect.Float32, reflect.Float64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Float() * float64(bv.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Float() * float64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return av.Float() * bv.Float(), nil
		default:
			return nil, fmt.Errorf("multiply: unknown type for %q (%T)", bv, b)
		}
	default:
		return nil, fmt.Errorf("multiply: unknown type for %q (%T)", av, a)
	}
}

// divide returns the division of b from a.
func divide(b, a interface{}) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Int() / bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Int() / int64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Int()) / bv.Float(), nil
		default:
			return nil, fmt.Errorf("divide: unknown type for %q (%T)", bv, b)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int64(av.Uint()) / bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Uint() / bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			return float64(av.Uint()) / bv.Float(), nil
		default:
			return nil, fmt.Errorf("divide: unknown type for %q (%T)", bv, b)
		}
	case reflect.Float32, reflect.Float64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Float() / float64(bv.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Float() / float64(bv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return av.Float() / bv.Float(), nil
		default:
			return nil, fmt.Errorf("divide: unknown type for %q (%T)", bv, b)
		}
	default:
		return nil, fmt.Errorf("divide: unknown type for %q (%T)", av, a)
	}
}

// modulo returns the modulo of b from a.
func modulo(b, a interface{}) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return av.Int() % bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Int() % int64(bv.Uint()), nil
		default:
			return nil, fmt.Errorf("modulo: unknown type for %q (%T)", bv, b)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int64(av.Uint()) % bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return av.Uint() % bv.Uint(), nil
		default:
			return nil, fmt.Errorf("modulo: unknown type for %q (%T)", bv, b)
		}
	default:
		return nil, fmt.Errorf("modulo: unknown type for %q (%T)", av, a)
	}
}

// minimum returns the minimum between a and b.
func minimum(b, a interface{}) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if av.Int() < bv.Int() {
				return av.Int(), nil
			}
			return bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if av.Int() < int64(bv.Uint()) {
				return av.Int(), nil
			}
			return bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			if float64(av.Int()) < bv.Float() {
				return av.Int(), nil
			}
			return bv.Float(), nil
		default:
			return nil, fmt.Errorf("minimum: unknown type for %q (%T)", bv, b)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if int64(av.Uint()) < bv.Int() {
				return av.Uint(), nil
			}
			return bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if av.Uint() < bv.Uint() {
				return av.Uint(), nil
			}
			return bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			if float64(av.Uint()) < bv.Float() {
				return av.Uint(), nil
			}
			return bv.Float(), nil
		default:
			return nil, fmt.Errorf("minimum: unknown type for %q (%T)", bv, b)
		}
	case reflect.Float32, reflect.Float64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if av.Float() < float64(bv.Int()) {
				return av.Float(), nil
			}
			return bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if av.Float() < float64(bv.Uint()) {
				return av.Float(), nil
			}
			return bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			if av.Float() < bv.Float() {
				return av.Float(), nil
			}
			return bv.Float(), nil
		default:
			return nil, fmt.Errorf("minimum: unknown type for %q (%T)", bv, b)
		}
	default:
		return nil, fmt.Errorf("minimum: unknown type for %q (%T)", av, a)
	}
}

// maximum returns the maximum between a and b.
func maximum(b, a interface{}) (interface{}, error) {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if av.Int() > bv.Int() {
				return av.Int(), nil
			}
			return bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if av.Int() > int64(bv.Uint()) {
				return av.Int(), nil
			}
			return bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			if float64(av.Int()) > bv.Float() {
				return av.Int(), nil
			}
			return bv.Float(), nil
		default:
			return nil, fmt.Errorf("maximum: unknown type for %q (%T)", bv, b)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if int64(av.Uint()) > bv.Int() {
				return av.Uint(), nil
			}
			return bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if av.Uint() > bv.Uint() {
				return av.Uint(), nil
			}
			return bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			if float64(av.Uint()) > bv.Float() {
				return av.Uint(), nil
			}
			return bv.Float(), nil
		default:
			return nil, fmt.Errorf("maximum: unknown type for %q (%T)", bv, b)
		}
	case reflect.Float32, reflect.Float64:
		switch bv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if av.Float() > float64(bv.Int()) {
				return av.Float(), nil
			}
			return bv.Int(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if av.Float() > float64(bv.Uint()) {
				return av.Float(), nil
			}
			return bv.Uint(), nil
		case reflect.Float32, reflect.Float64:
			if av.Float() > bv.Float() {
				return av.Float(), nil
			}
			return bv.Float(), nil
		default:
			return nil, fmt.Errorf("maximum: unknown type for %q (%T)", bv, b)
		}
	default:
		return nil, fmt.Errorf("maximum: unknown type for %q (%T)", av, a)
	}
}