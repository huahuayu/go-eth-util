package entity

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type PairCreated struct {
	BlockNumber uint64
	Pair        common.Address
	Token0      common.Address
	Token1      common.Address
	Token2      common.Address // for curve
	Token3      common.Address // for curve
	Fee         *big.Int       // uniswapV3 only
	TickSpacing *big.Int       // uniswapV3 only
}
