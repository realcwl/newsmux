package collector

// Actual Settings
var Settings *CollectorSettings

// This is the setting for collectors
type CollectorSettings struct {
	// BEARER_TOKEN used to access Twitter V2 API
	TWITTER_BEARER_TOKEN string `yaml:"TWITTER_BEARER_TOKEN"`
}

func InitializeCollectorSettings(s *CollectorSettings) {
	Settings = s
}
