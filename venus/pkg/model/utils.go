package model

import (
	"os"
	"fmt"

	"github.com/BurntSushi/toml"
)

// LoadConfigFromFile loads config from file 
func LoadConfigFromFile(filePath string) (*Config, error) {
	if _, err := os.Stat(filePath); err != nil {
		return nil, err
	}

	var conf Config
	if _, err := toml.DecodeFile(filePath, &conf); err != nil {
		return nil, err
	}

	return &conf, nil
}

// VerifyConfig verify if config is ok or not
func VerifyConfig(conf *Config) error {
	if len(conf.Policy.Symbols) == 0 {
		return fmt.Errorf("symbols should not be empty")
	}

	if conf.Policy.Trigger.BuyThreshold <= 0 || conf.Policy.Trigger.BuyThreshold > 1 {
		return fmt.Errorf("invalid buy threshold, should be within (0,1]")
	} 

	if conf.Policy.Trigger.SellThreshold <= 0 || conf.Policy.Trigger.SellThreshold > 1 {
		return fmt.Errorf("invalid sell threshold, should be within (0,1]")
	}

	if conf.Policy.Trade.Position <= 0 || conf.Policy.Trade.Position > 1 {
		return fmt.Errorf("invalid position %v, should be within (0,1]", conf.Policy.Trade.Position)
	}

	if conf.Policy.Trade.USDTPerBuy < 10 || conf.Policy.Trade.USDTPerBuy >= conf.Policy.Trade.MaxUSDTPerBuy {
		return fmt.Errorf("invalid max usdt per buy, should be within [10.0, MaxUSDTPerBuy]")
	}

	if conf.Policy.Trade.MaxUSDTPerBuy < 10 || conf.Policy.Trade.MaxUSDTPerBuy > 100 {
		return fmt.Errorf("invalid max usdt per buy, should be within [10.0, 100.0]")
	}
	// TODO: more checks
	return nil
}