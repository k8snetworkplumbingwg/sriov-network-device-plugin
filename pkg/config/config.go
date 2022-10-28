package config

import (
	"encoding/json"
	"fmt"
	"os"
)

var cfg *Config

// Config defines config structure
type Config struct {
	FeatureGates map[string]bool `json:"featureGates,omitempty"`
}

// NewConfig creates new config
func NewConfig() {
	if cfg == nil {
		cfg = newConfig()
	}
}

// GetConfig returns config
func GetConfig() (*Config, error) {
	var err error
	if cfg == nil {
		err = fmt.Errorf("config was not initialized")
	}
	return cfg, err
}

func newConfig() *Config {
	if cfg != nil {
		return cfg
	}
	newCfg := &Config{}
	newCfg.FeatureGates = make(map[string]bool)
	return newCfg
}

// ReadConfig loads config
func (cfg *Config) ReadConfig(configFile string) error {
	allCfg := make(map[string]json.RawMessage)
	rawBytes, err := os.ReadFile(configFile)

	if err != nil {
		return fmt.Errorf("error reading file %s, %v", configFile, err)
	}

	if err = json.Unmarshal(rawBytes, &allCfg); err != nil {
		return fmt.Errorf("error unmarshalling raw bytes %v", err)
	}

	fgMap := make(map[string]bool)

	if _, exists := allCfg["featureGates"]; !exists {
		return fmt.Errorf("no config for feature gate present")
	}

	if err = json.Unmarshal(allCfg["featureGates"], &fgMap); err != nil {
		return fmt.Errorf("error unmarshalling raw bytes %v", err)
	}

	for k, v := range fgMap {
		cfg.FeatureGates[k] = v
	}

	return nil
}
