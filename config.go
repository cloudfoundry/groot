package groot

import (
	"errors"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type config struct {
	LogLevel string `yaml:"log_level"`
}

func parseConfig(configFilePath string) (config, error) {
	if configFilePath == "" {
		return config{}, errors.New("please provide --config")
	}

	contents, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return config{}, err
	}

	var conf config
	if err := yaml.Unmarshal(contents, &conf); err != nil {
		return config{}, err
	}

	conf = applyDefaults(conf)

	return conf, nil
}

func applyDefaults(conf config) config {
	if conf.LogLevel == "" {
		conf.LogLevel = "info"
	}
	return conf
}
