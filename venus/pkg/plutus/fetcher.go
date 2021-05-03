package plutus

import (
	"time"
	"strconv"
	"context"

	"github.com/adshao/go-binance/v2"
	glog "github.com/vjoke/falcon/pkg/log"
	"github.com/vjoke/falcon/venus/pkg/model"
)

var fLog = glog.RegisterScope("fetcher", "fetcher", 0)
// Fetcher reads price and update price periodically
type Fetcher struct {
	arb      *Arbitrager
	interval time.Duration
	symbols  []string
	client   *binance.Client
	Tick	 uint64
}

// NewFetcher creates a new fetcher instance
func NewFetcher(arb *Arbitrager) *Fetcher {
	f := &Fetcher{
		arb:      arb,
		interval: arb.config.Policy.Sample.Interval.Duration,
		symbols:  arb.config.Policy.Symbols,
		client:   binance.NewClient(arb.config.Exchange.ApiKey, arb.config.Exchange.SecretKey),
	}

	return f
}

// Run begins the fetching process which will get price every tick
func (f *Fetcher) Run(stopCh <-chan struct{}) {
	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()
	fLog.Info("worker is running")

	for {
		select {
		case <-stopCh:
			fLog.Info("worker is stopped")
			return
		case <-ticker.C:
			f.Tick++
			tick := f.Tick
			for _, symbol := range f.symbols {
				go f.queryPrice(symbol, tick)
			}
		}
	}
}

// queryPrice querys price from binance api server
func (f *Fetcher) queryPrice(symbol string, tick uint64) {
	r, err := f.client.NewAveragePriceService().Symbol(symbol).Do(context.Background())
	if err != nil {
		fLog.Errorf("get price of %v error: %v", symbol, err)
		// FIXME: just ignore
		return
	}
	price, err := strconv.ParseFloat(r.Price, 64)
	if err != nil {
		fLog.Errorf("convert price error: %v", err)
		return
	}

	sp := &model.SamplePrice{
		Tick: tick,
		Symbol: symbol,
		Price:  price,
		Start:  time.Now().Format(TIME_FORMAT),
	}

	f.arb.UpdatePrice(sp)
}
