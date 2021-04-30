package bootstrap

// PixiuArgs provids all of the configuration parameters for pixiu service
type PixiuArgs struct {
	ConfigFile string
}

func NewPixiuArgs(initFuncs ...func(*PixiuArgs)) *PixiuArgs {
	p := &PixiuArgs{}

	// Apply defaults 
	p.applyDefaults()

	// Apply custom init functions
	for _, fn := range initFuncs {
		fn(p)
	}

	return p
}

// Apply default value to PixiuArgs
func (p *PixiuArgs)applyDefaults() {
	p.ConfigFile = "./config.toml"
}