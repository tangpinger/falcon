package plutus 

// Arbitrager defines the interface for arbitrators
type Arbitrager interface {
	Start(stop <-chan struct{})
	Stop()
}

// ArbitragerBuilder defines interface for building an arbitrager
type ArbitragerBuilder interface {
	WithConfig(string) ArbitragerBuilder 
	Build()(Arbitrager, error)
} 