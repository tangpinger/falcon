package plutus

import (
	glog "github.com/vjoke/falcon/pkg/log"
	"github.com/vjoke/falcon/venus/pkg/model"
)

var oLog = glog.RegisterScope("oracle", "oracle", 0)

// Oracle determines if it's right time for trading
type Oracle struct {
	arb            *Arbitrager
	windowLen      uint64
	slideDetect    bool
	symbolsLen     uint64
	symbols        []string
	buy_threshold  float64
	sell_threshold float64
	epochMap       map[string]*model.Epoch
	tick           uint64
	tickCount      uint64
}

// NewOracle creates a new oracle instance
func NewOracle(arb *Arbitrager) *Oracle {
	o := &Oracle{
		arb:            arb,
		slideDetect:    arb.config.Policy.Sample.SlideDetect,
		buy_threshold:  arb.config.Policy.Trigger.BuyThreshold,
		sell_threshold: arb.config.Policy.Trigger.SellThreshold,
		epochMap:       make(map[string]*model.Epoch),
		symbols:        arb.config.Policy.Symbols,
	}

	windowLen := arb.config.Policy.Sample.Window.Duration.Seconds() / arb.config.Policy.Sample.Interval.Duration.Seconds()
	o.windowLen = uint64(windowLen)
	o.symbolsLen = uint64(len(o.symbols))
	// Initialize the epoch map
	for _, symbol := range o.symbols {
		o.epochMap[symbol] = &model.Epoch{
			Symbol: symbol,
			Slots:  make([]model.Slot, o.windowLen),
		}
	}

	return o
}

// Run begins the process for check prices
func (o *Oracle) Run(stopCh <-chan struct{}) {
	oLog.Info("worker is running")

	for {
		select {
		case <-stopCh:
			oLog.Info("worker is stopped")
			return
		case sp := <-o.arb.priceChannel:
			oLog.Infof("got new price %v", sp)
			epoch, ok := o.epochMap[sp.Symbol]
			if !ok {
				oLog.Errorf("unknown symbol: %v", sp.Symbol)
				break
			}

			if sp.Tick < o.tick {
				oLog.Errorf("stale tick %v < %v, ignored", sp.Tick, o.tick)
				break
			} else if sp.Tick == o.tick {
				o.tickCount++
			} else {
				oLog.Infof("tick changed %v --> %v", o.tick, sp.Tick)
				o.tick = sp.Tick
				o.tickCount = 1
			}
			curSlot := &epoch.Slots[sp.Tick%o.windowLen]
			curSlot.Tick = sp.Tick
			curSlot.Price = sp.Price
			curSlot.Direction = model.DRAW
			// FIXME: Tick starts from 1, so it's not possibe to underflow
			prevTick := sp.Tick - 1
			prevSlot := &epoch.Slots[prevTick%o.windowLen]

			if prevSlot.Tick != prevTick {
				oLog.Warnf("previous slot: %v does not match: %v", prevSlot, prevTick)
				break
			}
			var legend string
			curSlot.Direction, legend = getPriceDirection(prevSlot.Price, curSlot.Price)
			oLog.Infof("current tick:%v, direction:%v", sp.Tick, legend)
			if o.isSlotMature() {
				riseGroup, fallGroup, otherGroup := o.groupSymbolsByDirection(o.tick)
				// FIXME: firstly, detect downtrend, then detect uptrend
				if float64(len(fallGroup))/float64(o.symbolsLen) >= o.sell_threshold {
					oLog.Infof("trigger sell orders with fall:%v/all:%v, threshod %v", len(fallGroup), o.symbolsLen, o.sell_threshold)
					o.arb.CreateOrders(&model.Order{Type: model.SELL_ORDER, Symbols: o.symbols})
				} else if float64(len(riseGroup))/float64(o.symbolsLen) >= o.buy_threshold {
					oLog.Infof("trigger buy orders with rise:%v/all:%v, threshod %v", len(riseGroup), o.symbolsLen, o.buy_threshold)
					if len(otherGroup) == 0 {
						oLog.Warnf("empty group for orders, do nothing!")
					} else {
						o.arb.CreateOrders(&model.Order{Type: model.BUY_ORDER, Symbols: otherGroup})
					}
				} else {
					oLog.Infof("not trigger sell/buy orders with fall:%v-rise:%v-all:%v, sell threshold:%v, buy threshold:%v", len(fallGroup), len(riseGroup), o.symbolsLen, o.sell_threshold, o.buy_threshold)
				}
			}
		}
	}
}

// isSlotMature check if the slot is mature for detecting
func (o *Oracle) isSlotMature() bool {
	if o.tickCount != o.symbolsLen {
		return false
	}

	if o.slideDetect {
		return o.tick >= o.windowLen
	} else {
		return 0 == (o.tick % o.windowLen)
	}
}

// getPriceDirection returns the price change direction and legend
func getPriceDirection(prevPrice, curPrice float64) (int32, string) {
	colorGreen := "\033[32m"
	colorRed := "\033[31m"
	
	if prevPrice < curPrice {
		return model.RISE, colorGreen + "↗"
	}

	if prevPrice > curPrice {
		return model.FALL, colorRed + "↘"
	}

	return model.DRAW, "-"
}

// groupSymbolsByDirection groups the symbols by continuous directions
func (o *Oracle) groupSymbolsByDirection(tick uint64) ([]string, []string, []string) {
	riseGroup := make([]string, 0, o.windowLen)
	fallGroup := make([]string, 0, o.windowLen)
	otherGroup := make([]string, 0, o.windowLen)

	for symbol, epoch := range o.epochMap {
		var totalScore int32 = 0
		var i uint64 = 0
		for ; i < o.windowLen; i++ {
			if tick < i {
				oLog.Warnf("slot tick skew, tick:%v, i:%v", tick, i)
				break
			}
			// FIXME: is algorithm ok?
			slot := epoch.Slots[(tick-i)%o.windowLen]
			if slot.Tick == (tick - i) {
				totalScore += slot.Direction
			} else {
				oLog.Warnf("slot tick mismatch, expected:%v, actual:%v", tick-i, slot.Tick)
				break
			}
		}

		if totalScore == int32(o.windowLen) {
			riseGroup = append(riseGroup, symbol)
		} else if totalScore == int32(-o.windowLen) {
			fallGroup = append(fallGroup, symbol)
		} else {
			otherGroup = append(otherGroup, symbol)
		}
	}

	return riseGroup, fallGroup, otherGroup
}
