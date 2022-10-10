package util

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/huahuayu/go-eth-util/contract/erc20"
	"math/big"
	"strings"
	"sync"
)

var (
	decimalCache = make(map[string]uint8)
	mu           sync.Mutex
)

func SortToken(tokenA, tokenB string) (token0 string, token1 string) {
	tokenABytes, _ := hex.DecodeString(strings.Replace(tokenA, "0x", "", -1))
	tokenBBytes, _ := hex.DecodeString(strings.Replace(tokenB, "0x", "", -1))
	if new(big.Int).SetBytes(tokenABytes).Cmp(new(big.Int).SetBytes(tokenBBytes)) < 0 {
		token0, token1 = tokenA, tokenB
	} else {
		token0, token1 = tokenB, tokenA
	}
	return
}

func TokenDecimal(client *ethclient.Client, token string) (int, error) {
	mu.Lock()
	decimal, ok := decimalCache[token]
	mu.Unlock()
	if ok {
		return int(decimal), nil
	}
	instance, err := erc20.NewIerc20(common.HexToAddress(token), client)
	if err != nil {
		return 0, err
	}
	decimal, err = instance.Decimals(nil)
	if err != nil {
		return 0, err
	}
	mu.Lock()
	decimalCache[token] = decimal
	mu.Unlock()
	return int(decimal), nil
}
