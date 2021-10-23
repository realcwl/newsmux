package app_setting

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// This is the panoptic config for panoptic execution.
type PanopticAppSetting struct {
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
	// Scheduler config polling interval in seconds
	SCHEDULER_CONFIG_POLL_INTERVAL_SECOND int64 `yaml:"SCHEDULER_CONFIG_POLL_INTERVAL_SECOND"`
	// Path pointing to local panoptic config
	LOCAL_PANOPTIC_CONFIG_PATH string `yaml:"LOCAL_PANOPTIC_CONFIG_PATH"`
	// Do not execute job on Lambda if debug is set to true, otherwise it will
	// execute on Lambda (though it won't be published to SNS due to Collector's
	// debug mode handling)
	DO_NOT_EXECUTE_ON_LAMBDA_FOR_DEBUG_JOB bool `yaml:"DO_NOT_EXECUTE_ON_LAMBDA_FOR_DEBUG_JOB"`
}

func ParsePanopticAppSetting(path string) PanopticAppSetting {
	c := PanopticAppSetting{}
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
