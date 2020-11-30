package types

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto"
)

// Get denom pair alphabetical ordered
func AlphabeticalDenomPair(denom1, denom2 string) (resDenom1, resDenom2 string) {
	if denom1 > denom2 {
		return denom2, denom1
	} else {
		return denom1, denom2
	}
}

// GetPoolReserveAcc returns the poor account for the provided poolKey (reserve denoms + poolType)
func GetPoolReserveAcc(poolKey string) sdk.AccAddress {
	return sdk.AccAddress(crypto.AddressHash([]byte(poolKey)))
}

// TODO: tmp denom rule, TBD
func GetPoolCoinDenom(reserveAcc sdk.AccAddress) string {
	return reserveAcc.String()
}

// TODO: check is poolcoin or not when poolcoin denom rule fixed
//func IsPoolCoin(coin sdk.Coin) bool {
//}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func EqualApprox(a , b sdk.Dec) bool {
	fmt.Println(a.Quo(b))
	fmt.Println(a.Quo(b).Sub(sdk.OneDec()))
	fmt.Println(a.Quo(b).Sub(sdk.OneDec()).Abs())
	if a.Quo(b).Sub(sdk.OneDec()).Abs().LT(sdk.NewDecWithPrec(1, 10)){
		return true
	} else {
		return false
	}
}