package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	log = logrus.WithFields(logrus.Fields{"logger": "config"})
)

// Secret is a string that must not be revealed on marshaling.
type Secret string

// MarshalYAML implements the yaml.Marshaler interface.
func (s Secret) MarshalYAML() (interface{}, error) {
	if s != "" {
		return "<secret>", nil
	}
	return nil, nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface for Secrets.
func (s *Secret) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Secret
	return unmarshal((*plain)(s))
}

// LoadConfig parses the YAML input into a Config.
func LoadConfig(s string) (*Config, error) {
	cfg := &Config{}
	err := yaml.Unmarshal([]byte(s), cfg)
	if err != nil {
		return nil, err
	}
	if log.Level == logrus.DebugLevel {
		fmt.Printf("Loaded config:\n%+v", cfg)
	}
	return cfg, nil
}

// LoadConfigFile parses the given YAML file into a Config.
func LoadConfigFile(filename string) (*Config, []byte, error) {
	log.Infof("Loading configuration from '%s'", filename)
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := LoadConfig(string(content))
	if err != nil {
		return nil, nil, err
	}

	//resolveFilepaths(filepath.Dir(filename), cfg)
	return cfg, content, nil
}

// Config is the top-level configuration for JIRAlert's config file.
type Config struct {
	Template               string   `yaml:"Template" json:"Template"`
	CheckYaml              bool     `yaml:"CheckYaml" json:"CheckYaml"`
	Selectors              []string `yaml:"Selectors,omitempty" json:"Selectors,omitempty"`
	CheckSelfConfig        bool     `yaml:"CheckSelfConfig" json:"CheckSelfConfig"`
	CheckJSON              bool     `yaml:"CheckJSON" json:"CheckJSON"`
	CheckCommand           string   `yaml:"CheckCommand" json:"CheckCommand"`
	CheckCommandOKExitCode []int    `yaml:"CheckCommandOKExitCode" json:"CheckCommandOKExitCode"`
	TmpDirectory           string   `yaml:"TmpDirectory" json:"TmpDirectory"`
	RemoveComment          bool     `yaml:"RemoveComment" json:"RemoveComment"`
	RemoveEmptyLines       bool     `yaml:"RemoveEmptyLines" json:"RemoveEmptyLines"`
	ToFileName             string   `yaml:"ToFileName" json:"ToFileName"`
	ToDirectory            string   `yaml:"ToDirectory" json:"ToDirectory"`
	ToNamespace            string   `yaml:"ToNamespace" json:"ToNamespace"`
	ToSecretName           string   `yaml:"ToSecretName" json:"ToSecretName"`
	ToConfigMapName        string   `yaml:"ToConfigMapName" json:"ToConfigMapName"`
	FromNamespace          string   `yaml:"FromNamespace" json:"FromNamespace"`
	URLRealoads            []string `yaml:"URLRealoads,omitempty" json:"URLRealoads,omitempty"`
	PrometheusMetricsPort  int      `yaml:"PrometheusMetricsPort" json:"PrometheusMetricsPort"`
	PrometheusMetricsURL   string   `yaml:"PrometheusMetricsURL" json:"PrometheusMetricsURL"`

	// Catches all undefined fields and must be empty after parsing.
	XXX map[string]interface{} `yaml:",inline" json:"-"`
}

func (c Config) String() string {
	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("<error creating config string: %s>", err)
	}
	return string(b)
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// We want to set c to the defaults and then overwrite it with the input.
	// To make unmarshal fill the plain data struct rather than calling UnmarshalYAML
	// again, we have to hide it using a type indirection.
	type plain Config
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	if c.Template != "" && c.ToFileName == "" {
		return fmt.Errorf("missing ToFileName")
	}

	if (c.ToSecretName != "" || c.ToConfigMapName != "") && c.ToNamespace == "" {
		return fmt.Errorf("missing ToNamespace")
	}

	if len(c.Selectors) == 0 {
		return fmt.Errorf("missing Selectors")
	}

	if c.ToSecretName == "" && c.ToConfigMapName == "" && c.ToFileName == "" {

	}

	for _, selector := range c.Selectors {
		sel := strings.Split(selector, "/")
		if len(sel) != 2 {
			return fmt.Errorf("missing Selectors")
		}

		if sel[0] != "configmap" && sel[0] != "secret" {
			return fmt.Errorf("Wrong kind for Selectors")
		}
	}

	if c.CheckYaml && c.CheckJSON {
		return fmt.Errorf("Check syntax for Yaml and Json (Yaml!=Json)")
	}

	if c.PrometheusMetricsPort == 0 {
		c.PrometheusMetricsPort = 2112
	}

	if c.PrometheusMetricsURL == "" {
		c.PrometheusMetricsURL = "/metrics"
	}

	if len(c.CheckCommandOKExitCode) == 0 {
		c.CheckCommandOKExitCode = []int{0}
	}

	return checkOverflow(c.XXX, "config")
}

func checkOverflow(m map[string]interface{}, ctx string) error {
	if len(m) > 0 {
		var keys []string
		for k := range m {
			keys = append(keys, k)
		}
		log.Warningf("unknown fields in %s: %s", ctx, strings.Join(keys, ", "))
	}
	return nil
}
