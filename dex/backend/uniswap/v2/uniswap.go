package v2

import (
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	. "github.com/huahuayu/go-eth-util/contract/uniswap/factory/v2"
	. "github.com/huahuayu/go-eth-util/contract/uniswap/pair/v2"
	"github.com/huahuayu/go-eth-util/dex/backend/uniswap/entity"
	"github.com/huahuayu/go-eth-util/dex/backend/util"
	"github.com/shopspring/decimal"
	"log"
	"math/big"
)

type UniswapV2 struct {
	EthClient *ethclient.Client
	Factory   *Factory
}

func NewUniswap(ethClient *ethclient.Client, factory string) (*UniswapV2, error) {
	factoryContract, err := NewFactory(common.HexToAddress(factory), ethClient)
	if err != nil {
		return nil, err
	}
	return &UniswapV2{
		EthClient: ethClient,
		Factory:   factoryContract,
	}, nil
}

func (u *UniswapV2) SpotPrice(opts *bind.CallOpts, baseToken, quoteToken string, middleToken ...string) (*decimal.Decimal, error) {
	if baseToken == quoteToken {
		p := decimal.NewFromInt(1)
		return &p, nil
	}
	if len(middleToken) == 0 {
		return u.spotPrice(opts, baseToken, quoteToken)
	}
	path := append(append([]string{baseToken}, middleToken...), quoteToken)
	price := decimal.NewFromInt(1)
	for i := 0; i < len(path)-1; i++ {
		p, err := u.spotPrice(opts, path[i], path[i+1])
		if err != nil {
			return nil, err
		}
		price = price.Mul(*p)
	}
	return &price, nil
}

func (u *UniswapV2) WatchPairCreated() (event.Subscription, chan *entity.PairCreated, error) {
	pairCreated := make(chan *FactoryPairCreated)
	res := make(chan *entity.PairCreated)
	sub, err := u.Factory.WatchPairCreated(nil, pairCreated, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	go func() {
		for {
			select {
			case err := <-sub.Err():
				log.Println("uniswapV2 WatchPairCreated: ", err)
			case p := <-pairCreated:
				if p.Raw.Removed {
					continue
				}
				res <- &entity.PairCreated{
					BlockNumber: p.Raw.BlockNumber,
					Pair:        p.Pair,
					Token0:      p.Token0,
					Token1:      p.Token1,
				}
			}
		}
	}()
	return sub, res, nil
}

func (u *UniswapV2) FilterPairCreated(fromBlock uint64, toBlock *uint64) ([]*entity.PairCreated, error) {
	iterator, err := u.Factory.FilterPairCreated(&bind.FilterOpts{Start: fromBlock, End: toBlock}, nil, nil)
	if err != nil {
		return nil, err
	}
	var res []*entity.PairCreated
	for iterator.Next() {
		if iterator.Event.Raw.Removed {
			continue
		}
		res = append(res, &entity.PairCreated{
			BlockNumber: iterator.Event.Raw.BlockNumber,
			Pair:        iterator.Event.Pair,
			Token0:      iterator.Event.Token0,
			Token1:      iterator.Event.Token1,
		})
	}
	return res, nil
}

func (u *UniswapV2) spotPrice(opts *bind.CallOpts, baseToken, quoteToken string) (*decimal.Decimal, error) {
	p, err := u.PairFor(baseToken, quoteToken)
	if err != nil {
		return nil, err
	}
	pairContract, err := NewPair(common.HexToAddress(p), u.EthClient)
	if err != nil {
		return nil, err
	}
	reserves, err := pairContract.GetReserves(opts)
	if err != nil {
		return nil, err
	}
	baseTokenDecimal, err := util.TokenDecimal(u.EthClient, baseToken)
	if err != nil {
		return nil, err
	}
	quoteTokenDecimal, err := util.TokenDecimal(u.EthClient, quoteToken)
	if err != nil {
		return nil, err
	}
	token0, _ := util.SortToken(baseToken, quoteToken)
	if token0 == baseToken {
		return calculatePrice(reserves.Reserve0, reserves.Reserve1, int32(baseTokenDecimal), int32(quoteTokenDecimal))
	} else {
		return calculatePrice(reserves.Reserve1, reserves.Reserve0, int32(baseTokenDecimal), int32(quoteTokenDecimal))
	}
}

func (u *UniswapV2) PairFor(baseToken, quoteToken string, feeRate ...int) (string, error) {
	pair, err := u.Factory.GetPair(nil, common.HexToAddress(baseToken), common.HexToAddress(quoteToken))
	if err != nil {
		return "", err
	}
	if pair == common.HexToAddress("0x0000000000000000000000000000000000000000") {
		return "", errors.New("pair not found")
	}
	return pair.String(), nil
}

func calculatePrice(baseTokenReserve, quoteTokenReserve *big.Int, baseTokenDecimals, quoteTokenDecimals int32) (*decimal.Decimal, error) {
	if decimal.NewFromBigInt(baseTokenReserve, -baseTokenDecimals) == decimal.Zero {
		return nil, errors.New("base token reserve is zero")
	}
	p := decimal.NewFromBigInt(quoteTokenReserve, -quoteTokenDecimals).Div(decimal.NewFromBigInt(baseTokenReserve, -baseTokenDecimals))
	return &p, nil
}
