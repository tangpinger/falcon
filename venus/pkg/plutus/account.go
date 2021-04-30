package plutus

import (
	"fmt"
	"context"
	"strconv"

	"github.com/adshao/go-binance/v2"
	glog "github.com/vjoke/falcon/pkg/log"
)

var accntLog = glog.RegisterScope("account", "account", 0)

// Account hold info for an account
type Account struct {
	arb     *Arbitrager
	client  *binance.Client
}

// NewAccount creates a new account instance
func NewAccount(arb *Arbitrager) *Account {
	a := &Account{
		arb:     arb,
		client:  binance.NewClient(arb.config.Exchange.ApiKey, arb.config.Exchange.SecretKey),
	}
	
	return a
}

// GetAccount gets account info from api server
func (a *Account) GetAccount() (*binance.Account, error) {
	account, err := a.client.NewGetAccountService().Do(context.Background())
	if err != nil {
		return nil, err
	}

	a.removeZeroBalances(account)
	accntLog.Infof("%+v", account)

	return account, nil
}

// GetBalance gets balance for an asset
func (a *Account) GetBalance(asset string) (float64, float64, error) {
	account, err := a.GetAccount()
	if err != nil {
		accntLog.Error(err)
		return 0, 0, err
	}

	for _, b := range account.Balances {
		if b.Asset == asset {
			return a.parseBalance(&b)	
		}
	}

	return 0, 0, fmt.Errorf("found no asset %v", asset)
}

// GetBalanceMap gets balance map for all the non-zero asset
func (a *Account) GetBalanceMap() (map[string]float64, error) {
	m := make(map[string]float64)
	account, err := a.GetAccount()
	if err != nil {
		accntLog.Error(err)
		return m, err
	}

	for _, b := range account.Balances {
		free, locked, err := a.parseBalance(&b)
		if err != nil {
			accntLog.Error(err)
			continue
		} 
		m[b.Asset] = free + locked
	}	

	accntLog.Debugf("balance map is %v", m)
	return m, nil
}

// removeZeroBalances removes zero balances 
func (a *Account) removeZeroBalances(account *binance.Account) {
	balances := make([]binance.Balance, 0, len(account.Balances))
	for _, b := range account.Balances {
		free, locked, err := a.parseBalance(&b)
		if err == nil && (free + locked) > 0 {
			balances = append(balances, b)
		}
	}

	account.Balances = balances
}

// parseBalance converts string value to float value
func (a *Account) parseBalance(b *binance.Balance)(float64, float64, error) {
	free, err := strconv.ParseFloat(b.Free, 64)
	if err != nil {
		accntLog.Errorf("Convert free balance %v error: %v", b.Free, err)
		return 0, 0, err
	}

	locked, err := strconv.ParseFloat(b.Locked, 64)
	if err != nil {
		accntLog.Errorf("Convert locked balance %v, error: %v", b.Locked, err)
		return 0, 0, err
	}

	return free, locked, nil
}

