package plutus

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type exchangeTestSuite struct {
	suite.Suite
}

func TestExchange(t *testing.T) {
	suite.Run(t, new(exchangeTestSuite))
}

func (e *exchangeTestSuite) SetupTest() {
	
}
	
func (e *exchangeTestSuite) TestNormalizeQuantity() {
	// TODO:
} 