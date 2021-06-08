package pixiu

import (
	glog "github.com/vjoke/falcon/pkg/log"
	model "github.com/vjoke/falcon/venus/pkg/model/pixiu"
	"github.com/vjoke/falcon/venus/pkg/plutus"
	"github.com/adshao/go-binance/v2"
)

const (
	TIME_FORMAT = "2006-01-02 15:04:05.000000"
)

var aLog = glog.RegisterScope("arbitrager", "arbitrager", 0)

// ArbitragerBuilder implements the builder for pixiu arbitrager
type ArbitragerBuilder struct {
	configFile string
}

func NewBuilder() *ArbitragerBuilder {
	return &ArbitragerBuilder{}
} 

// WithConfig sets the config file
func (ab *ArbitragerBuilder) WithConfig(configFile string) plutus.ArbitragerBuilder {
	ab.configFile = configFile
	return ab
}

// Build creates an arbitrager instance
func (ab *ArbitragerBuilder) Build() (plutus.Arbitrager, error) {
	conf, err := model.LoadConfigFromFile(ab.configFile)
	if err != nil {
		return nil, err
	}

	if err := model.VerifyConfig(conf); err != nil {
		return nil, err
	}

	a, err := NewArbitrager(conf)
	if err != nil {
		return nil, err
	}

	return a, nil
}


// Arbitrager defines components for arbitraging
type Arbitrager struct {
	config *model.Config
	exch *Exchange
	account *Account
	fetcher *Fetcher
	oracle *Oracle
	trader *Trader
	priceChannel chan *model.SamplePrice
	tradeChannel chan *model.Order
}

// NewArbitrager creates a new arbitrager instance
func NewArbitrager(config *model.Config) (*Arbitrager, error) {
	binance.UseTestnet = config.Policy.Testnet
	aLog.Infof("has %v symbols, testnet: %v, dryrun: %v", len(config.Policy.Symbols), config.Policy.Testnet, config.Policy.Dryrun)
	priceChannel := make(chan *model.SamplePrice, 20)
	tradeChannel := make(chan *model.Order, 40)

	a := &Arbitrager{
		config: config,
		priceChannel: priceChannel,
		tradeChannel: tradeChannel,
	}

	exch, err := NewExchange(a)
	if err != nil {
		return nil, err
	}

	a.exch = exch
	a.account = NewAccount(a)
	a.fetcher = NewFetcher(a)
	a.oracle = NewOracle(a)
	a.trader = NewTrader(a)

	return a, nil
}

func (a *Arbitrager) Start(stopCh <-chan struct{}) {
	go func() {
		a.account.GetAccount()
	}()
	go a.fetcher.Run(stopCh)
	go a.oracle.Run(stopCh)
	go a.trader.Run(stopCh)
}

func (a *Arbitrager) Stop() {
	// TODO:
}

// UpdatePrice updates latest price for a symbol
func (a *Arbitrager) UpdatePrice(sp *model.SamplePrice) {
	a.priceChannel <- sp
}

// CreateOrders creates buy orders when uptrend is predicted
// creates sell orders when downtrend is predicted.
func (a *Arbitrager) CreateOrders(o *model.Order) {
	a.tradeChannel <- o
}