package pixiu

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/adshao/go-binance/v2"
	"github.com/jinzhu/copier"
	glog "github.com/vjoke/falcon/pkg/log"
)

var eLog = glog.RegisterScope("exchange", "exchange", 0)

// Exchange hold info for the binance exchange
type Exchange struct {
	arb *Arbitrager
	client *binance.Client
	info *binance.ExchangeInfo
	symbolMap map[string]*binance.Symbol
	extraMap map[string]*FilterExtra
}

// NewExchange creates a new exchange instance
func NewExchange(arb *Arbitrager) (*Exchange, error) {
	exch := &Exchange{
		arb: arb,
		client:  binance.NewClient(arb.config.Exchange.ApiKey, arb.config.Exchange.SecretKey),
		symbolMap: make(map[string]*binance.Symbol),
		extraMap: make(map[string]*FilterExtra),
	}

	info, err := exch.client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		eLog.Errorf("failed to get exchange info, err:%v", err)
		return nil, err
	}

	exch.info = info

	for _, symbol := range info.Symbols {
		// FIXME: if we use pointer of returned symbol, some data will be
		// modified with unknown reasons
		newSymbol := binance.Symbol{}
		copier.Copy(&newSymbol, &symbol)
		exch.symbolMap[symbol.Symbol] = &newSymbol
	}

	for _, symbol := range arb.config.Policy.Symbols {
		fLotSize := exch.symbolMap[symbol].LotSizeFilter()
		eLog.Debugf("%v lot filter is %+v", symbol, *fLotSize)

		fPrice := exch.symbolMap[symbol].PriceFilter()
		eLog.Debugf("%v price filter is %+v", symbol, *fPrice)

		fe := &FilterExtra{
			LotSize: GetLotExtra(fLotSize),
			Price: GetPriceExtra(fPrice),
		}

		exch.extraMap[symbol] = fe
	}

	return exch, nil
}

// NormalizeQuantity normalizes the quantity
func (exch *Exchange) NormalizeQuantity(symbol string, quantity float64) string {
	fe := exch.extraMap[symbol]
	if fe == nil {
		err_msg := fmt.Sprintf("failed to get extra map for %v", symbol)
		panic(err_msg)
	}

	temp := math.Trunc(quantity * fe.LotSize.fMinQuantityMultiply)
	r := temp * fe.LotSize.fMinQuantity

	return strconv.FormatFloat(r, 'f', fe.LotSize.nPrecision, 64)
}

// NormalizePrice normalizes the price
func (exch *Exchange) NormalizePrice(symbol string, quantity float64) string {
	fe := exch.extraMap[symbol]
	if fe == nil {
		err_msg := fmt.Sprintf("failed to get extra map for %v", symbol)
		panic(err_msg)
	}

	temp := math.Trunc(quantity * fe.Price.fMinPriceMultiply)
	r := temp * fe.Price.fMinPrice

	return strconv.FormatFloat(r, 'f', fe.Price.nPrecision, 64)
}

// GetLotExtra returns extra info for the lot filter
// TODO: move to a common place
func GetLotExtra(f *binance.LotSizeFilter) *LotSizeFilterExtra {
	fMaxQuantity := MustParseFloat(f.MaxQuantity)
	fMinQuantity := MustParseFloat(f.MinQuantity)
	fStepSize := MustParseFloat(f.StepSize)
	fMinQuantityMultiply := 1.0/fMinQuantity
	nPrecision := GetPrecison(f.MinQuantity)

	extra := &LotSizeFilterExtra{
		fMaxQuantity,
		fMinQuantity,
		fStepSize,
		fMinQuantityMultiply,
		nPrecision,
	}

	eLog.Debugf("lot extra:%+v", extra)
	
	return extra
}

// GetPriceExtra returns extra info for the price filter
func GetPriceExtra(f *binance.PriceFilter) *PriceFilterExtra {
	fMaxPrice := MustParseFloat(f.MaxPrice)
	fMinPrice := MustParseFloat(f.MinPrice)
	fTickSize := MustParseFloat(f.TickSize)

	fMinPriceMultiply := 1.0/fMinPrice
	nPrecision := GetPrecison(f.MinPrice)

	extra := &PriceFilterExtra{
		fMaxPrice,
		fMinPrice,
		fTickSize,
		fMinPriceMultiply,
		nPrecision,
	}

	eLog.Debugf("price extra:%+v", extra)

	return extra
}

// FilterExtra define extra filters for normalizing
type FilterExtra struct {
	Price *PriceFilterExtra
	LotSize *LotSizeFilterExtra
	// TODO: add more extra filters
}

// PriceFilterExtra defines extra parameters for normalizing price 
type PriceFilterExtra struct {
	fMaxPrice float64
	fMinPrice float64
	fTickSize float64

	fMinPriceMultiply float64
	nPrecision int
}

// LotSizeFilterExtra defines extra parameters for normalizing quantity
type LotSizeFilterExtra struct {
	fMaxQuantity  float64
	fMinQuantity float64
	fStepSize    float64

	fMinQuantityMultiply float64 
	nPrecision	 int
}

// MustParseFloat parses the float, panic if error
func MustParseFloat(str string) float64 {
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		eLog.Error(err)
		panic(err)
	}

	if val <= 0 {
		panic("zero float value")
	}

	return val
}

// GetPrecison returns the precision of string float
func GetPrecison(str string) int {
	switch str {
	case "1.00000000":
		return 0
	case "0.10000000":
		return 1
	case "0.01000000":
		return 2
	case "0.00100000":
		return 3
	case "0.00010000":
		return 4
	case "0.00001000":
		return 5
	case "0.00000100":
		return 6
	case "0.00000010":
		return 7
	case "0.00000001":
		return 8
	default:
		return 8
	}
}



