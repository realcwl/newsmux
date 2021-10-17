package app_config

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// This is the panoptic config for panoptic execution.
type PanopticAppConfig struct {
	// Number of Lambdas maintained at a given time.
	LAMBDA_POOL_SIZE int `yaml:"LAMBDA_POOL_SIZE"`
	// Lambda life span in second. Any lambda function that exceed this
	// value will be cleaned up and replaced with a new one.
	LAMBDA_LIFE_SPAN_SECOND int64 `yaml:"LAMBDA_LIFE_SPAN_SECOND"`
	// Maintain the lambda pool every other interval.
	MAINTAIN_EVERY_SECOND int64 `yaml:"MAINTAIN_EVERY_SECOND"`
	// Force to use remote config file, instead of using local Panoptic schedule.
	// Otherwise, we use remote fetch for production config, and use local for
	// dev and testing.
	FORCE_REMOTE_SCHEDULE_PULL bool `yaml:"FORCE_REMOTE_SCHEDULE_PULL"`
}

func ParsePanopticAppConfig(path string) PanopticAppConfig {
	c := PanopticAppConfig{}
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("yamlFile. get err: ", err.Error())
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		log.Fatal("Unmarshal: ", err)
	}
	return c
}
