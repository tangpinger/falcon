package model

import (
	"time"
)

// Config defines the configuration
type Config struct {
	Exchange *Exchange `toml:"exchange"`
	Policy   *Policy   `toml:"policy"`
	Res      *Res      `toml:"res"`
}

// Exchange defines the information for an exchange
type Exchange struct {
	Name      string `toml:"name"`
	Api       string `toml:"api"`
	ApiKey    string `toml:"api_key"`
	SecretKey string `toml:"secret_key"`
}

// Policy defines a particular policy for trading
type Policy struct {
	Name      string     `toml:"name"`
	Testnet   bool       `toml:"testnet"`
	Dryrun	  bool       `toml:"dryrun"`
	Symbols   []string   `toml:"symbols"`
	Sample    *Sample    `toml:"sample"`
	Condition *Condition `toml:"condition"`
	Trigger   *Trigger   `toml:"trigger"`
	Trade     *Trade     `toml:"trade"`
}

// Sample defines configuration for sampling
type Sample struct {
	Interval duration `toml:"interval"`
	Window   duration `toml:"window"`
	SlideDetect bool  `toml:"slide_detect"`
}

// Condition defines the total trading amout of an exchange pair
// We only select the non-mainstream pairs
type Condition struct {
	Min uint `toml:"min"`
	Max uint `toml:"max"`
}

// Trigger defines the threshold for trading
type Trigger struct {
	SellThreshold float64 `toml:"sell_threshold"`
	BuyThreshold  float64 `toml:"buy_threshold"`
}

// Trade define parameters for trading
type Trade struct {
	ChaseUp	    bool   `toml:"chase_up"`
	OneByOne	bool   `toml:"one_by_one"`
	Fee        float64 `toml:"fee"`
	StopLoss   float64 `toml:"stop_loss"`
	StopProfit float64 `toml:"stop_profit"`
	Position   float64 `toml:"position"`
	USDTPerBuy float64 `toml:"usdt_per_buy"`
	MaxUSDTPerBuy float64 `toml:"max_usdt_per_buy"`
}

// Res defines the database configurations
type Res struct {
	// TODO:
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}
