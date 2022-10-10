package dex

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/huahuayu/go-eth-util/dex/consts"
	"math/big"
	"testing"
)

const (
	WETH      = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	USDC      = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	UNI       = "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984"
	v2Factory = "0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f"
	v3Factory = "0x1F98431c8aD98523631AE4a59f267346ea31F984"
)

func TestDex_SpotPrice(t *testing.T) {
	ethClient, err := ethclient.Dial("https://mainnet.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161")
	if err != nil {
		t.Fatal(err)
	}
	uniswapV2, err := NewDex(ethClient, consts.Uniswap, consts.UniswapV2, v2Factory)
	// UNI -> WETH -> USDC
	priceV2, err := uniswapV2.SpotPrice(nil, UNI, USDC, WETH)
	if err != nil {
		t.Error(err)
	}
	t.Log(priceV2)
	number, err := ethClient.BlockNumber(context.TODO())
	if err != nil {
		t.Error()
	}
	uniswapV3, err := NewDex(ethClient, consts.Uniswap, consts.UniswapV3, v3Factory)
	// uni history price in -100 block
	priceV3, err := uniswapV3.SpotPrice(&bind.CallOpts{BlockNumber: big.NewInt(int64(number - 100))}, UNI, USDC, WETH)
	if err != nil {
		t.Error(err)
	}
	t.Log(priceV3)
}
