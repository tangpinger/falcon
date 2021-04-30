package model

import (
	"fmt"
	"time"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type configTestSuite struct {
	suite.Suite
	configFile string
}

func TestConfig(t *testing.T) {
	suite.Run(t, new(configTestSuite))
}

func (c *configTestSuite) SetupTest() {
	c.configFile = "../../../config/pixiu.toml"
}
	
func (c *configTestSuite) TestLoadFromFile() {
	conf, err := LoadConfigFromFile(c.configFile)
	assert.Nil(c.T(), err)
	fmt.Println(conf.Exchange.Name)
	assert.Equal(c.T(), conf.Policy.Sample.Interval.Duration, time.Minute)
	assert.Equal(c.T(), conf.Policy.Sample.Window.Duration, time.Minute * 5)
} 