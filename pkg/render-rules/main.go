package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/cybozu-go/setup-hw/redfish"
	"github.com/ghodss/yaml"
)

func main() {
	rules, err := load()
	if err != nil {
		log.Fatal(err)
	}

	err = render(rules)
	if err != nil {
		log.Fatal(err)
	}
}

func load() (map[string]*redfish.CollectRule, error) {
	filenames, err := filepath.Glob("rules/*.yml")
	if err != nil {
		return nil, err
	}

	rules := make(map[string]*redfish.CollectRule)
	for _, filename := range filenames {
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		rule := new(redfish.CollectRule)
		err = yaml.Unmarshal(data, rule)
		if err != nil {
			return nil, err
		}

		err = rule.Validate()
		if err != nil {
			return nil, err
		}

		rules[filepath.Base(filename)] = rule
	}

	return rules, nil
}

func render(rules map[string]*redfish.CollectRule) error {
	f, err := os.OpenFile("rendered_rules.go", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tmpl.Execute(f, rules)
	if err != nil {
		return err
	}

	err = f.Sync()
	if err != nil {
		return err
	}

	return exec.Command("goimports", "-w", f.Name()).Run()
}

var tmpl = template.Must(template.New("").Parse(`// Code generated by render-rules.  DO NOT EDIT.
//go:generate go run ../pkg/render-rules

package redfish

import (
	"log"
)

func init() {
	for _, rule := range Rules {
		if err := rule.Compile(); err != nil {
			log.Fatal(err)
		}
	}
}

var Rules = map[string]*CollectRule{
	{{- range $key, $value := . }}
	{{ printf "%q" $key }}: {
		TraverseRule: TraverseRule{
			Root: {{ printf "%q" $value.TraverseRule.Root }},
			ExcludeRules: []string{
				{{- range $value.TraverseRule.ExcludeRules }}
				{{ printf "%q" . }},
				{{- end }}
			},
		},
		MetricRules: []*MetricRule{
			{{- range $value.MetricRules }}
			{
				Path: {{ printf "%q" .Path }},
				PropertyRules: []*PropertyRule{
					{{- range .PropertyRules }}
					{
						Pointer: {{ printf "%q" .Pointer }},
						Name: {{ printf "%q" .Name }},
						Help: {{ printf "%q" .Help }},
						Type: {{ printf "%q" .Type }},
					},
					{{- end }}
				},
			},
			{{- end }}
		},
	},
	{{- end }}
}
`))
