package uniswap

import (
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/huahuayu/go-eth-util/dex/backend/uniswap/entity"
	v2 "github.com/huahuayu/go-eth-util/dex/backend/uniswap/v2"
	v3 "github.com/huahuayu/go-eth-util/dex/backend/uniswap/v3"
	"github.com/huahuayu/go-eth-util/dex/consts"
	"github.com/shopspring/decimal"
)

type IUniswap interface {
	// SpotPrice returns the price of baseToken/quoteToken, it indicates how much of the quote token is needed to purchase one unit of the base token.
	SpotPrice(opts *bind.CallOpts, baseToken, quoteToken string, middleToken ...string) (*decimal.Decimal, error)
	// PairFor returns the pair address of baseToken and quoteToken.
	PairFor(baseToken, quoteToken string, feeRate ...int) (string, error)
	// WatchPairCreated watches the event of PairCreated.
	WatchPairCreated() (event.Subscription, chan *entity.PairCreated, error)
	// FilterPairCreated returns the event of PairCreated with block range.
	FilterPairCreated(fromBlock uint64, toBlock *uint64) ([]*entity.PairCreated, error)
	// Liquidity returns the liquidity of the pair.
	Liquidity(opts *bind.CallOpts, baseToken, quoteToken string) (*decimal.Decimal, *decimal.Decimal, error)
}

func NewUniswap(ethClient *ethclient.Client, version string, factory string) (IUniswap, error) {
	switch version {
	case consts.UniswapV2:
		return v2.NewUniswap(ethClient, factory)
	case consts.UniswapV3:
		return v3.NewUniswap(ethClient, factory)
	default:
		return nil, errors.New("unsupported uniswap version")
	}
}
