package uniswap

import (
	"context"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/huahuayu/go-eth-util/dex/consts"
	"testing"
)

func newClient() *ethclient.Client {
	ethClient, err := ethclient.Dial("ws://your_node:8546")
	if err != nil {
		panic(err)
	}
	return ethClient
}

func TestUniswap_WatchPairCreatedV2(t *testing.T) {
	ethClient := newClient()
	uniswapV2, err := NewUniswap(ethClient, consts.UniswapV2, "0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f")
	if err != nil {
		t.Error(err)
		return
	}
	sub, ch, err := uniswapV2.WatchPairCreated()
	if err != nil {
		t.Error(err)
		return
	}
	defer sub.Unsubscribe()
	for {
		select {
		case err := <-sub.Err():
			t.Error(err)
			return
		case event := <-ch:
			t.Log(event)
		}
	}
}

func TestUniswap_WatchPairCreatedV3(t *testing.T) {
	ethClient := newClient()
	uniswapV3, err := NewUniswap(ethClient, consts.UniswapV3, "0x1f98431c8ad98523631ae4a59f267346ea31f984")
	if err != nil {
		t.Error(err)
		return
	}
	sub, ch, err := uniswapV3.WatchPairCreated()
	if err != nil {
		t.Error(err)
		return
	}
	defer sub.Unsubscribe()
	for {
		select {
		case err := <-sub.Err():
			t.Error(err)
			return
		case event := <-ch:
			t.Log(event)
		}
	}
}

func TestUniswap_FilterPairCreatedV2(t *testing.T) {
	ethClient := newClient()
	uniswapV2, err := NewUniswap(ethClient, consts.UniswapV2, "0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f")
	if err != nil {
		t.Error(err)
		return
	}
	number, err := ethClient.BlockNumber(context.TODO())
	if err != nil {
		t.Error(err)
		return
	}
	events, err := uniswapV2.FilterPairCreated(number-1000, nil)
	if err != nil {
		t.Error(err)
		return
	}
	for _, event := range events {
		t.Log(event)
	}
}

func TestUniswap_FilterPairCreatedV3(t *testing.T) {
	ethClient := newClient()
	uniswapV3, err := NewUniswap(ethClient, consts.UniswapV3, "0x1f98431c8ad98523631ae4a59f267346ea31f984")
	if err != nil {
		t.Error(err)
		return
	}
	number, err := ethClient.BlockNumber(context.TODO())
	if err != nil {
		t.Error(err)
		return
	}
	events, err := uniswapV3.FilterPairCreated(number-10000, nil)
	if err != nil {
		t.Error(err)
		return
	}
	for _, event := range events {
		t.Log(event)
	}
}
