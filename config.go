package groot

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type config struct {
	LogLevel string `yaml:"log_level"`
}

func parseConfig(configFilePath string) (conf config, err error) {
	defer func() {
		if err == nil {
			conf = applyDefaults(conf)
		}
	}()

	if configFilePath == "" {
		return conf, nil
	}

	contents, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return config{}, err
	}

	if err := yaml.Unmarshal(contents, &conf); err != nil {
		return config{}, err
	}

	return conf, nil
}

func applyDefaults(conf config) config {
	if conf.LogLevel == "" {
		conf.LogLevel = "info"
	}
	return conf
}
