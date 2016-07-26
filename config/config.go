package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Target is a configuration to define some information which is necessary to setup server
type Target struct {
	Host     string   `yaml:"host"`
	User     string   `yaml:"user"`
	Port     int      `yaml:"port"`
	Branches []string `yaml:"branches"`
}

// Project ...
type Project struct {
	Name     string   `yaml:"name"`
	Database Database `yaml:"database"`
	Port     int      `yaml:"port"`
}

// Database ...
type Database struct {
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Seed     string `yaml:"seed"`
}

// Notification ...
type Notification struct {
	Slack Slack `yaml:"slack"`
	Email Email `yaml:"email"`
}

// Slack ...
type Slack struct {
	Enable   bool     `yaml:"enable"`
	Channels []string `yaml:"channels"`
}

// Email ...
type Email struct {
	Enable     bool     `yaml:"enable"`
	Recipients []string `yaml:"recipients"`
}

// Config ...
type Config struct {
	Targets      []Target     `yaml:"targets"`
	Project      Project      `yaml:"project"`
	Notification Notification `yaml:"notification"`
	Registry     string       `yaml:"registry"`
}

// Init ...
func Init(configFile string) (conf *Config, err error) {
	configBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	conf = &Config{}
	if err = yaml.Unmarshal(configBytes, conf); err != nil {
		return nil, err
	}

	return conf, nil
}
