package v3

import (
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/huahuayu/go-eth-util/contract/erc20"
	. "github.com/huahuayu/go-eth-util/contract/uniswap/factory/v3"
	. "github.com/huahuayu/go-eth-util/contract/uniswap/pair/v3"
	"github.com/huahuayu/go-eth-util/dex/backend/uniswap/entity"
	"github.com/huahuayu/go-eth-util/dex/backend/util"
	"github.com/shopspring/decimal"
	"log"
	"math/big"
	"strings"
)

type UniswapV3 struct {
	EthClient *ethclient.Client
	Factory   *Factory
	Name      string
}

func NewUniswap(ethClient *ethclient.Client, factory string, name string) (*UniswapV3, error) {
	factoryContract, err := NewFactory(common.HexToAddress(factory), ethClient)
	if err != nil {
		return nil, err
	}
	return &UniswapV3{
		EthClient: ethClient,
		Factory:   factoryContract,
		Name:      name,
	}, nil
}

func (u *UniswapV3) GetName() string {
	return u.Name
}

func (u *UniswapV3) SpotPrice(opts *bind.CallOpts, baseToken, quoteToken string, middleToken ...string) ([]string, *decimal.Decimal, error) {
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

func (u *UniswapV3) WatchPairCreated() (event.Subscription, chan *entity.PairCreated, error) {
	pairCreated := make(chan *FactoryPoolCreated)
	res := make(chan *entity.PairCreated)
	sub, err := u.Factory.WatchPoolCreated(nil, pairCreated, nil, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	go func() {
		for {
			select {
			case err := <-sub.Err():
				log.Println("uniswapVr32 WatchPairCreated: ", err)
			case p := <-pairCreated:
				if p.Raw.Removed {
					continue
				}
				res <- &entity.PairCreated{
					BlockNumber: p.Raw.BlockNumber,
					Pair:        p.Pool,
					Token0:      p.Token0,
					Token1:      p.Token1,
					Fee:         p.Fee,
					TickSpacing: p.TickSpacing,
				}
			}
		}
	}()
	return sub, res, nil
}

func (u *UniswapV3) FilterPairCreated(fromBlock uint64, toBlock *uint64) ([]*entity.PairCreated, error) {
	iterator, err := u.Factory.FilterPoolCreated(&bind.FilterOpts{Start: fromBlock, End: toBlock}, nil, nil, nil)
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
			Pair:        iterator.Event.Pool,
			Token0:      iterator.Event.Token0,
			Token1:      iterator.Event.Token1,
			Fee:         iterator.Event.Fee,
			TickSpacing: iterator.Event.TickSpacing,
		})
	}
	return res, nil
}

func (u *UniswapV3) spotPrice(opts *bind.CallOpts, baseToken, quoteToken string) (string, *decimal.Decimal, error) {
	pair, err := u.PairFor(baseToken, quoteToken)
	if err != nil {
		return "", nil, err
	}
	pairContract, err := NewV3pair(common.HexToAddress(pair), u.EthClient)
	if err != nil {
		return "", nil, err
	}
	var sqrtPriceX96Big *big.Int
	res, err := pairContract.Slot0(opts)
	if err != nil {
		// try to get reserves from event
		if strings.Contains(err.Error(), "missing trie node") {
			blockNumber := opts.BlockNumber
			start := blockNumber.Uint64()
			if swapIterator, err := pairContract.FilterSwap(&bind.FilterOpts{Start: start, End: &start}, nil, nil); err == nil && swapIterator != nil {
				for swapIterator.Next() {
					sqrtPriceX96Big = swapIterator.Event.SqrtPriceX96
				}
			}
		} else {
			return "", nil, err
		}
	} else {
		sqrtPriceX96Big = res.SqrtPriceX96
	}
	baseTokenDecimal, err := util.TokenDecimal(u.EthClient, baseToken)
	if err != nil {
		return "", nil, err
	}
	quoteTokenDecimal, err := util.TokenDecimal(u.EthClient, quoteToken)
	if err != nil {
		return "", nil, err
	}
	token0, _ := util.SortToken(baseToken, quoteToken)
	//    sqrtPriceX96 = sqrt(price) * 2 ** 96
	//    sqrtPriceX96 / (2 ** 96) = sqrt(price)
	//    (sqrtPriceX96 / (2 ** 96)) ** 2 = price
	//    sqrtPriceX96 ** 2 / 2 ** 192 = price
	sqrtPriceX96 := decimal.NewFromBigInt(sqrtPriceX96Big, 0)
	price := sqrtPriceX96.Mul(sqrtPriceX96).Div(decimal.NewFromInt(2).Pow(decimal.NewFromInt(192)))
	if token0 == baseToken {
		price = price.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(baseTokenDecimal - quoteTokenDecimal))))
	} else {
		if price.String() == "0" {
			return "", nil, errors.New("price is zero")
		}
		price = price.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(quoteTokenDecimal - baseTokenDecimal))))
		price = decimal.NewFromInt(1).Div(price)
	}
	return pair, &price, nil
}

func (u *UniswapV3) PairFor(baseToken, quoteToken string, feeRate ...int) (string, error) {
	// specific fee rate
	if len(feeRate) > 0 {
		pool, err := u.Factory.GetPool(nil, common.HexToAddress(baseToken), common.HexToAddress(quoteToken), big.NewInt(int64(feeRate[0])))
		if err != nil || pool == common.HexToAddress("0x0") {
			return "", err
		}
		return pool.String(), nil
	}
	// try to find best fee rate pair
	var pair string
	for _, fee := range []int64{500, 3000, 10000} {
		pool, err := u.Factory.GetPool(nil, common.HexToAddress(baseToken), common.HexToAddress(quoteToken), big.NewInt(fee))
		if err != nil || pool == common.HexToAddress("0x0") {
			continue
		} else {
			pair = pool.String()
			break
		}
	}
	if pair == "" {
		return "", errors.New("no pair found")
	}
	return pair, nil
}

func (u *UniswapV3) Liquidity(opts *bind.CallOpts, baseToken, quoteToken string) (string, *decimal.Decimal, *decimal.Decimal, error) {
	pair, err := u.PairFor(baseToken, quoteToken)
	if err != nil {
		return "", nil, nil, err
	}
	baseTokenDecimal, err := util.TokenDecimal(u.EthClient, baseToken)
	if err != nil {
		return "", nil, nil, err
	}
	quoteTokenDecimal, err := util.TokenDecimal(u.EthClient, quoteToken)
	if err != nil {
		return "", nil, nil, err
	}
	var baseReserve, quoteReserve decimal.Decimal
	if erc20, _ := erc20.NewIerc20(common.HexToAddress(baseToken), u.EthClient); erc20 != nil {
		if balance, err := erc20.BalanceOf(opts, common.HexToAddress(pair)); balance != nil && err == nil {
			baseReserve = decimal.NewFromBigInt(balance, -int32(baseTokenDecimal))
		} else {
			return "", nil, nil, err
		}
	}
	if erc20, _ := erc20.NewIerc20(common.HexToAddress(quoteToken), u.EthClient); erc20 != nil {
		if balance, err := erc20.BalanceOf(opts, common.HexToAddress(pair)); balance != nil && err == nil {
			quoteReserve = decimal.NewFromBigInt(balance, -int32(quoteTokenDecimal))
		} else {
			return "", nil, nil, err
		}
	}
	return pair, &baseReserve, &quoteReserve, nil
}
