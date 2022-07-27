package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
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
