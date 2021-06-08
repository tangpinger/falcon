package pixiu 

// Price defines the price for sampling
type SamplePrice struct {
	Tick   uint64
	Symbol string
	Price  float64
	Start  string
}

// Epoch
type Epoch struct {
	Symbol string
	Slots  []Slot
}

const (
	RISE = 1
	DRAW = 0
	FALL = -1
)

// Slot represents the change of price
type Slot struct {
	Tick      uint64
	Price     float64
	Direction int32
}

const (
	BUY_ORDER  = "buy" 
	SELL_ORDER = "sell"
)

// Order defines the symbols to sell/buy
type Order struct {
	Type    string
	Symbols []string
}
