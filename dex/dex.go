package dex

import (
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/huahuayu/go-eth-util/dex/backend/uniswap"
	"github.com/huahuayu/go-eth-util/dex/consts"
	"github.com/shopspring/decimal"
)

type IDex interface {
	// GetName returns dex name
	GetName() string
	// SpotPrice returns the spot price of baseToken/quoteToken, it indicates how much of the quote token is needed to purchase one unit of the base token.
	// If middleToken is not empty, it indicates the price of baseToken/middleTokens/quoteToken.
	// Get history spot price by specify blockNumber in call opts , if blockNumber is nil, it will return the latest spot price.
	// returns pair router, spot price, error
	SpotPrice(opts *bind.CallOpts, baseToken, quoteToken string, middleTokens ...string) ([]string, *decimal.Decimal, error)
	// Liquidity returns the pair name and liquidity of the pair.
	Liquidity(opts *bind.CallOpts, baseToken, quoteToken string) (string, *decimal.Decimal, *decimal.Decimal, error)
}

func NewDex(ethClient *ethclient.Client, protocol string, version string, param any) (IDex, error) {
	switch protocol {
	case consts.Uniswap:
		config, ok := param.(string)
		if !ok {
			return nil, errors.New("invalid param")
		}
		return uniswap.NewUniswap(ethClient, version, config)
	default:
		return nil, errors.New("unsupported protocol")
	}
}
