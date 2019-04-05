package redfish

import (
	"errors"
	"regexp"
	"strings"
)

// CollectRule is a set of rules of traversing and converting Redfish data.
type CollectRule struct {
	TraverseRule traverseRule `yaml:"Traverse"`
	MetricRules  []metricRule `yaml:"Metrics"`
}

type traverseRule struct {
	Root          string   `yaml:"Root"`
	ExcludeRules  []string `yaml:"Excludes"`
	excludeRegexp *regexp.Regexp
}

type metricRule struct {
	Path          string         `yaml:"Path"`
	PropertyRules []propertyRule `yaml:"Properties"`
}

type propertyRule struct {
	Pointer     string    `yaml:"Pointer"`
	Name        string    `yaml:"Name"`
	Description string    `yaml:"Description"`
	Converter   converter `yaml:"Type"`
}

type converter func(interface{}) (float64, error)

// Validate checks CollectRule.
func (cr *CollectRule) Validate() error {
	if err := cr.TraverseRule.validate(); err != nil {
		return err
	}

	for _, metricRule := range cr.MetricRules {
		if err := metricRule.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (tr *traverseRule) validate() error {
	if tr.Root == "" {
		return errors.New("Root is mandatory for traverse rule")
	}

	// TODO: this is not validation; refactor this in removing statik
	if len(tr.ExcludeRules) > 0 {
		excludes := strings.Join(tr.ExcludeRules, "|")
		r, err := regexp.Compile(excludes)
		if err != nil {
			return err
		}
		tr.excludeRegexp = r
	}

	return nil
}

func (mr metricRule) validate() error {
	if mr.Path == "" {
		return errors.New("Path is mandatory for metric rule")
	}

	for _, propertyRule := range mr.PropertyRules {
		if err := propertyRule.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (pr propertyRule) validate() error {
	if pr.Pointer == "" {
		return errors.New("Pointer is mandatory for property rule")
	}
	if pr.Name == "" {
		return errors.New("Name is mandatory for property rule")
	}
	if pr.Converter == nil {
		return errors.New("Converter is mandatory for property rule")
	}

	return nil
}

func (c *converter) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var typeName string
	err := unmarshal(&typeName)
	if err != nil {
		return err
	}

	converter, ok := typeToConverters[typeName]
	if !ok {
		return errors.New("unknown metrics type: " + typeName)
	}

	*c = converter
	return nil
}

var typeToConverters = map[string]converter{
	"number": numberConverter,
	"health": healthConverter,
	"state":  stateConverter,
}

func numberConverter(data interface{}) (float64, error) {
	value, ok := data.(float64)
	if !ok {
		return 0, errors.New("value was not float64")
	}
	return value, nil
}

func healthConverter(data interface{}) (float64, error) {
	if data == nil {
		return -1, nil
	}
	health, ok := data.(string)
	if !ok {
		return -1, errors.New("health value was not string")
	}
	switch health {
	case "OK":
		return 0, nil
	case "Warning":
		return 1, nil
	case "Critical":
		return 2, nil
	}
	return -1, errors.New("unknown health value: " + health)
}

func stateConverter(data interface{}) (float64, error) {
	state, ok := data.(string)
	if !ok {
		return -1, errors.New("state value was not string")
	}
	switch state {
	case "Enabled":
		return 0, nil
	case "Disabled":
		return 1, nil
	case "Absent":
		return 2, nil
	case "Deferring":
		return 3, nil
	case "InTest":
		return 4, nil
	case "Quiesced":
		return 5, nil
	case "StandbyOffline":
		return 6, nil
	case "StandbySpare":
		return 7, nil
	case "Starting":
		return 8, nil
	case "UnavailableOffline":
		return 9, nil
	case "Updating":
		return 10, nil
	}
	return -1, errors.New("unknown state value: " + state)
}
