package utils

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
)

func IsZeroAddress(addr common.Address) bool {
	for i := range addr {
		if i != 0 {
			return false
		}
	}
	return true
}

func ToEther(value *big.Int) float64 {
	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(18)))
	num, _ := decimal.NewFromString(value.String())
	result := num.Div(mul)
	f, _ := result.Float64()
	return f
}

func ToGWei(value *big.Int) float64 {
	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(9)))
	num, _ := decimal.NewFromString(value.String())
	result := num.Div(mul)
	f, _ := result.Float64()
	return f
}

func JsonString(value any) string {
	val, _ := json.Marshal(value)
	return string(val)
}
