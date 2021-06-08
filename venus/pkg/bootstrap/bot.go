package bootstrap

import (
	"github.com/vjoke/falcon/pkg/log"
	"github.com/vjoke/falcon/venus/pkg/plutus"
	"github.com/vjoke/falcon/venus/pkg/server"
)

// Bot contains the runtime configuration for the bot
type Bot struct {
	// TODO: add other components
	server server.Instance
	arb    plutus.Arbitrager
}

// NewBot creates a new bot instance based on the provided parameters
func NewBot(builder plutus.ArbitragerBuilder, initFuncs ...func(*Bot)) (*Bot, error) {
	b := &Bot{
		server: server.New(),
	}

	for _, fn := range initFuncs {
		fn(b)
	}

	err := b.initArbitrager(builder)
	if err != nil {
		log.Errorf("failed to init arbitrager: %v", err)
		return nil, err
	}

	return b, nil
}

func (b *Bot) initArbitrager(builder plutus.ArbitragerBuilder) error {
	arb, err := builder.Build()
	if err != nil {
		return err
	}

	b.arb = arb
	b.addStartFunc(func(stop <-chan struct{}) error {
		log.Infof("start arbitrager ...")
		b.arb.Start(stop)
		return nil
	})

	return nil
}

// Start starts all components of the pixiu bot
// Bot can be canceled at any time by closing the provided stop channel.
func (b *Bot) Start(stop <-chan struct{}) error {
	log.Info("start the bot ...")

	// Now start all of the components.
	if err := b.server.Start(stop); err != nil {
		return err
	}

	b.waitForShutdown(stop)

	log.Info("bot started")
	return nil
}

func (b *Bot) waitForShutdown(stop <-chan struct{}) {
	go func() {
		<-stop
		// TODO: shutdown all the comonents
		log.Info("bot shutdown")
	}()
}

// WaitUntilCompletion waits for everything marked as a "required termination" to complete.
// This should be called before exiting.
func (b *Bot) WaitUntilCompletion() {
	b.server.Wait()
}

// addStartFunc appends a function to be run. These are run synchronously in order,
// so the function should start a go routine if it needs to do anything blocking
func (b *Bot) addStartFunc(fn server.Component) {
	b.server.RunComponent(fn)
}

// addTerminatingStartFunc adds a function that should terminate before the serve shuts down
// This is useful to do cleanup activities
// This is does not guarantee they will terminate gracefully - best effort only
// Function should be synchronous; once it returns it is considered "done"
func (b *Bot) addTerminatingStartFunc(fn server.Component) {
	b.server.RunComponentAsyncAndWait(fn)
}
