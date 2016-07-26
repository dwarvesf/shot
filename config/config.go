package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config is a configuration to define some information which is necessary to setup server
type Target struct {
	Host     string   `yaml:"host"`
	User     string   `yaml:"user"`
	Port     int      `yaml:"port"`
	Branches []string `yaml:"branches"`
}

type Project struct {
	Name string `yaml:"name"`
	Seed string `yaml:"seed"`
	Port int    `yaml:"port"`
}

type Notification struct {
	Slacks []string `yaml:"slacks"`
	Emails []string `yaml:"emails"`
}

type Config struct {
	Targets      *[]Target     `yaml:"targets"`
	Project      *Project      `yaml:"project"`
	Notification *Notification `yaml:"notification"`
	Registry     string        `yaml:"registry"`
}

func Init(configFile string) (conf *Config, err error) {
	if configBytes, err := ioutil.ReadFile(configFile); err != nil {
		return nil, err
	} else {
		conf = &Config{}
		if err = yaml.Unmarshal(configBytes, conf); err != nil {
			return nil, err
		}
	}
	return conf, nil
}
