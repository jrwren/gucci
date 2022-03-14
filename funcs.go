package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

func service(k string) interface{} {
	return services()[k]
}

func services() map[string]interface{} {
	var m map[string]interface{}
	json.Unmarshal([]byte(os.Getenv("services")), &m)
	return m
}

type KVP struct {
	Name, Value string
}

func ls(k string) (m []KVP) {
	vars := os.Environ()
	for i := range vars {
		if strings.HasPrefix(vars[i], k) {
			k, v, _ := strings.Cut(vars[i], "=")
			m = append(m, KVP{k, v})
		}
	}
	return m
}
