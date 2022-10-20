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
	"strings"
)

type UniswapV2 struct {
	EthClient *ethclient.Client
	Factory   *Factory
	Name      string
}

func NewUniswap(ethClient *ethclient.Client, factory string, name string) (*UniswapV2, error) {
	factoryContract, err := NewFactory(common.HexToAddress(factory), ethClient)
	if err != nil {
		return nil, err
	}
	return &UniswapV2{
		EthClient: ethClient,
		Factory:   factoryContract,
		Name:      name,
	}, nil
}

func (u *UniswapV2) GetName() string {
	return u.Name
}

func (u *UniswapV2) SpotPrice(opts *bind.CallOpts, baseToken, quoteToken string, middleToken ...string) ([]string, *decimal.Decimal, error) {
	if baseToken == quoteToken {
		p := decimal.NewFromInt(1)
		return nil, &p, nil
	}
	var router = make([]string, 0)
	if len(middleToken) == 0 {
		pair, price, err := u.spotPrice(opts, baseToken, quoteToken)
		if err != nil {
			return nil, nil, err
		} else {
			router = append(router, pair)
			return router, price, nil
		}
	}
	path := append(append([]string{baseToken}, middleToken...), quoteToken)
	price := decimal.NewFromInt(1)
	for i := 0; i < len(path)-1; i++ {
		pair, p, err := u.spotPrice(opts, path[i], path[i+1])
		if err != nil {
			return nil, nil, err
		}
		router = append(router, pair)
		price = price.Mul(*p)
	}
	return router, &price, nil
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

func (u *UniswapV2) spotPrice(opts *bind.CallOpts, baseToken, quoteToken string) (string, *decimal.Decimal, error) {
	pair, baseReserves, quoteReserves, err := u.Liquidity(opts, baseToken, quoteToken)
	if err != nil {
		return "", nil, err
	}
	price := quoteReserves.Div(*baseReserves)
	return pair, &price, nil
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

func (u *UniswapV2) Liquidity(opts *bind.CallOpts, baseToken, quoteToken string) (string, *decimal.Decimal, *decimal.Decimal, error) {
	p, err := u.PairFor(baseToken, quoteToken)
	if err != nil {
		return "", nil, nil, err
	}
	pairContract, err := NewPair(common.HexToAddress(p), u.EthClient)
	if err != nil {
		return "", nil, nil, err
	}
	reserves, err := pairContract.GetReserves(opts)
	if err != nil {
		// try to get reserves from event
		if strings.Contains(err.Error(), "missing trie node") {
			blockNumber := opts.BlockNumber
			start := blockNumber.Uint64()
			if pairSyncIterator, err := pairContract.FilterSync(&bind.FilterOpts{Start: start, End: &start}); err == nil && pairSyncIterator != nil {
				for pairSyncIterator.Next() {
					reserves = struct {
						Reserve0           *big.Int
						Reserve1           *big.Int
						BlockTimestampLast uint32
					}{Reserve0: pairSyncIterator.Event.Reserve0, Reserve1: pairSyncIterator.Event.Reserve1}
				}
			}
		} else {
			return "", nil, nil, err
		}
	}
	token0, _ := util.SortToken(baseToken, quoteToken)
	baseTokenDecimal, err := util.TokenDecimal(u.EthClient, baseToken)
	if err != nil {
		return "", nil, nil, err
	}
	quoteTokenDecimal, err := util.TokenDecimal(u.EthClient, quoteToken)
	if err != nil {
		return "", nil, nil, err
	}
	if token0 == baseToken {
		baseReserve := decimal.NewFromBigInt(reserves.Reserve0, int32(-1*baseTokenDecimal))
		quoteReserve := decimal.NewFromBigInt(reserves.Reserve1, int32(-1*quoteTokenDecimal))
		return p, &baseReserve, &quoteReserve, nil
	} else {
		baseReserve := decimal.NewFromBigInt(reserves.Reserve1, int32(-1*baseTokenDecimal))
		quoteReserve := decimal.NewFromBigInt(reserves.Reserve0, int32(-1*quoteTokenDecimal))
		return p, &baseReserve, &quoteReserve, nil
	}
}
