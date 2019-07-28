package main

import (
	"fmt"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type config struct {
	ServiceFiles   []string `yaml:"service_files"`
	Address        string   `yaml:"address"`
	Proto          string   `yaml:"proto"`
	UserAgent      string   `yaml:"user_agent"`
	Workers        int      `yaml:"workers"`
	Timeout        int      `yaml:"timeout"`
	NoProgress     bool     `yaml:"no_progress"`
	NoBashComments bool     `yaml:"no_bash_comments"`
	Output         string   `yaml:"output"`

	services map[string]service `yaml:"services"`
}

type service struct {
	Addresses []string        `yaml:"addresses"`
	Tests     map[string]test `yaml:"tests"`
}

type test struct {
	SSL            bool                `yaml:"ssl"`
	Status         int                 `yaml:"status"`
	Resources      map[string]resource `yaml:"resources"`
	RequestHeaders map[string]string   `yaml:"req_headers"`
}

type resource struct {
	URL         string   `json:"url"`
	ContentType string   `json:"content_type"`
	Contains    []string `yaml:"contains"`
}

func (c *config) ReadServiceFiles() error {
	s := map[string]service{}
	for _, file := range c.ServiceFiles {
		fi, err := os.Stat(file)
		if err != nil {
			return err
		}

		mode := fi.Mode()
		if mode.IsDir() {
			continue
		}

		yamlFile, err := ioutil.ReadFile(file)
		if err != nil {
			errOut := fmt.Errorf("Error while reading config file %s: %s\n", file, err)
			return errOut
		}

		err = yaml.Unmarshal(yamlFile, s)
		if err != nil {
			errOut := fmt.Errorf("Error while unmarshalling config file %s: %s\n", file, err)
			return errOut
		}
	}
	c.services = s
	return nil
}
