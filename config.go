package groot

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type config struct {
	LogLevel           string   `yaml:"log_level"`
	LogFormat          string   `yaml:"log_format"`
	InsecureRegistries []string `yaml:"insecure_registries"`
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

	contents, err := os.ReadFile(configFilePath)
	if err != nil {
		return config{}, fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(contents, &conf); err != nil {
		return config{}, fmt.Errorf("parsing config file: %w", err)
	}

	return conf, nil
}

func applyDefaults(conf config) config {
	if conf.LogLevel == "" {
		conf.LogLevel = "info"
	}
	if conf.LogFormat == "" {
		conf.LogFormat = "rfc3339"
	}
	return conf
}
