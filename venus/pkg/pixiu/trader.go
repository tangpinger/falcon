package pixiu

import (
	"fmt"
	"time"
	"context"
	"strconv"
	"sync"

	"github.com/adshao/go-binance/v2"
	glog "github.com/vjoke/falcon/pkg/log"
	model "github.com/vjoke/falcon/venus/pkg/model/pixiu"
)

var tLog = glog.RegisterScope("trader", "trader", 0)

// Trader places orders according to events
type Trader struct {
	arb         *Arbitrager
	span		*model.Span
	stop_profit float64
	stop_loss   float64
	position    float64
	usdt_per_buy float64
	max_usdt_per_buy float64
	dryrun		bool
	one_by_one	bool
	client      *binance.Client
}

// NewTrader creates a new trader instance
func NewTrader(arb *Arbitrager) *Trader {
	t := &Trader{
		arb:         arb,
		span: arb.config.Policy.Trade.Span,
		stop_profit: arb.config.Policy.Trade.StopProfit,
		stop_loss:   arb.config.Policy.Trade.StopLoss,
		position:    arb.config.Policy.Trade.Position,
		usdt_per_buy: arb.config.Policy.Trade.USDTPerBuy,
		max_usdt_per_buy: arb.config.Policy.Trade.MaxUSDTPerBuy,
		dryrun:    	 arb.config.Policy.Dryrun,
		one_by_one:  arb.config.Policy.Trade.OneByOne,
		client:      binance.NewClient(arb.config.Exchange.ApiKey, arb.config.Exchange.SecretKey),
	}

	return t
}

// Run begins the trading process
func (t *Trader) Run(stopCh <-chan struct{}) {
	tLog.Info("worker is running")
	for {
		select {
		case <-stopCh:
			tLog.Info("worker is stopped")
			return
		case o := <-t.arb.tradeChannel:
			tLog.Debug("order request: %v", o)
			if !t.timeInSpan() {
				tLog.Warn("out of timespan for trading, ignored")
				break
			}
			if t.dryrun {
				tLog.Info("ignore order request in dryrun mode")
				break
			}
			if o.Type == model.BUY_ORDER {
				t.processBuyOrder(o.Symbols)
			} else {
				t.processSellOrder(o.Symbols)
			}
		}
	}
}

// timeInSpan check if current time is within the timespan for trading
func (t *Trader) timeInSpan() bool {
	// China doesn't have daylight saving. It uses a fixed 8 hour offset from UTC.
	// TODO: make timezone configurable
	secondsEastOfUTC := int((8 * time.Hour).Seconds())
	beijing := time.FixedZone("Beijing Time", secondsEastOfUTC)

	now := time.Now()
	begin := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, beijing)
	start := begin.Add(t.span.From.Duration)
	end := begin.Add(t.span.To.Duration)
	
	tLog.Debugf("timespan for trading is %v - %v", start.Format(TIME_FORMAT), end.Format(TIME_FORMAT))
	// FIXME: should we adjust timezone before comparing time?
	return start.Before(now) && end.After(now)

}

// processBuyOrder processes buy orders
func (t *Trader) processBuyOrder(symbols []string) {
	// Check balance
	free, _, err := t.arb.account.GetBalance("USDT")
	if err != nil {
		tLog.Error(err)
		return
	}
	// Place orders 
	total := free * t.position
	for _, symbol := range symbols {
		if total < t.usdt_per_buy {
			tLog.Warnf("insufficient usdt: %v < %v", total, t.usdt_per_buy)
			break
		} 
		total -= t.usdt_per_buy
		go t.buyOrder(symbol, t.usdt_per_buy)
		if t.one_by_one {
			tLog.Warnf("buy %v, one by one", symbol)
			break
		}
	}
}

// buyOrder places a market order for a symbol
func (t *Trader) buyOrder(symbol string, quantity float64) {
	strQuantity := strconv.FormatFloat(quantity, 'f', 8, 64)
	tLog.Infof("will buy %v with %v USDT", symbol, strQuantity)
	res, err := t.client.NewCreateOrderService().Symbol(symbol).
		Side(binance.SideTypeBuy).Type(binance.OrderTypeMarket).
		// TimeInForce(binance.TimeInForceTypeGTC).
		QuoteOrderQty(strQuantity).Do(context.Background())
	if err != nil {
		tLog.Error(err)
		return
	}
	tLog.Infof("created order %+v", res)
	// Calculate average price
	avgPrice, base, err := t.getMarketOrderInfo(res)
	if err != nil {
		// TODO: require manual operation to sell order?
		tLog.Errorf("failed to get average price, err:%v", err)
		return
	}

	baseStr := t.arb.exch.NormalizeQuantity(symbol, base)
	// Calculate sell price
	sellPrice := avgPrice * (1 + t.stop_profit)
	stopPrice := avgPrice * (1 - t.stop_loss)

	avgPriceStr := t.arb.exch.NormalizePrice(symbol, avgPrice)
	sellPriceStr := t.arb.exch.NormalizePrice(symbol, sellPrice)
	stopPriceStr := t.arb.exch.NormalizePrice(symbol, stopPrice)
	stopLimitPriceStr := stopPriceStr // FIXME: use the same value?
	// Create sell order or OTC order
	tLog.Infof("will sell %v %v avg: %v with sellPrice: %v stopPrice: %v, stopLimitPrice: %v", 
		baseStr, symbol, avgPriceStr, sellPriceStr, stopPriceStr, stopLimitPriceStr)
	ocoRes, err := t.client.NewCreateOCOService().
		Symbol(symbol).
		Side(binance.SideTypeSell).
		Quantity(baseStr).
		Price(sellPriceStr).
		StopPrice(stopPriceStr).
		StopLimitPrice(stopLimitPriceStr).
		StopLimitTimeInForce(binance.TimeInForceTypeGTC). // FIXME: GTC/IOC/FOK
		Do(context.Background())

	if err != nil {
		// TODO: retry?
		tLog.Errorf("failed to create oco order for %v, err:%v", symbol, err)
		return
	}

	tLog.Infof("created oco order: %+v", ocoRes)
}

// getMarketOrderInfo gets info for market order
func (t *Trader) getMarketOrderInfo(res *binance.CreateOrderResponse) (float64, float64, error) {
	if res.Status != binance.OrderStatusTypeFilled {
		return 0, 0, fmt.Errorf("order %v is not filled", res.OrderID)
	}

	var quote float64
	var base float64
	var totalCommission float64
	for _, f := range res.Fills {
		// TODO: set response type to avoid unnecessary conversion
		price, err := strconv.ParseFloat(f.Price, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("convert price error: %v", err)
		}

		quantity, err := strconv.ParseFloat(f.Quantity, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("convert quantity error: %v", err)
		}

		commission, err := strconv.ParseFloat(f.Commission, 64)
		if err != nil{
			return 0, 0, fmt.Errorf("convert commission error: %v", err)
		}

		quote += price * quantity
		base += quantity
		totalCommission += commission
	}

	tLog.Debugf("market order: quote: %v, base: %v, totalCommission: %v", 
		quote, base, totalCommission)

	if base <= 0 {
		return 0, 0, fmt.Errorf("total base is zero")
	}
	// FIXME: is commissionAsset the same with base asset?
	baseLeft := base - totalCommission
	if baseLeft <= 0 {
		return 0, 0, fmt.Errorf("total base left is zero")
	}	

	// FIXME: use base to calculate average instead of the base left
	return quote / base, baseLeft, nil
}

// processSellOrder processes sell orders
func (t *Trader) processSellOrder(symbols []string) {
	// cancel all the pending orders if any
	var wg sync.WaitGroup
	for _, symbol := range symbols {
		wg.Add(1)
		go t.cancelOrders(symbol, &wg)
	}
	wg.Wait()

	tLog.Infof("all the open orders for %v have been cancelled", symbols)
	// get remaining balances
	balanceMap, err := t.arb.account.GetBalanceMap()
	if err != nil {
		tLog.Error(err)
		return // FIXME:
	}
	// send sell order with market price
	for _, symbol := range symbols {
		go func(sym string) {
			asset := sym[:len(sym)-4] // FIXME：remove suffix USDT
			quantity := balanceMap[asset]
			if quantity <= 0 {
				tLog.Warnf("quantity for selling is invalid: %v", quantity)
			} else {
				t.sellOrder(sym, quantity)
			}
		}(symbol)
	}
}

// sellOrder creates sell order
func (t *Trader) sellOrder(symbol string, quantity float64) error {
	strQuantity := t.arb.exch.NormalizeQuantity(symbol, quantity)
	tLog.Infof("will sell %v %v", strQuantity, symbol)
	res, err := t.client.NewCreateOrderService().Symbol(symbol).
		Side(binance.SideTypeSell).Type(binance.OrderTypeMarket).
		// TimeInForce(binance.TimeInForceTypeGTC).
		Quantity(strQuantity).Do(context.Background())
	if err != nil {
		tLog.Errorf("failed to sell %v order %v", symbol, err)
		return err
	}

	tLog.Infof("created sell order %v", res)
	return nil
}

// cancelOrders cancel all the open orders
// TODO: check orders before cancelling
func (t *Trader) cancelOrders(symbol string, wg *sync.WaitGroup) {
	defer wg.Done()
	res, err := t.client.NewCancelOpenOrdersService().Symbol(symbol).Do(context.Background())
	if err != nil {
		tLog.Errorf("failed to cancel open orders of %v, err:%v", symbol, err)
		// TODO: retry?
		return
	}

	tLog.Info("cancelled open orders of %v, got %v", symbol, res)
}
