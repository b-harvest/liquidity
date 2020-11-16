package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/liquidity/x/liquidity/types"
)

func (k Keeper) SwapExecution(ctx sdk.Context, liquidityPoolBatch types.LiquidityPoolBatch) error {
	params := k.GetParams(ctx)
	pool, found := k.GetLiquidityPool(ctx, liquidityPoolBatch.PoolId)
	if !found {
		return types.ErrPoolNotExists
	}

	// get reserve Coin from the liquidity pool
	reserveCoins := k.GetReserveCoins(ctx, pool)
	reserveCoins.Sort()

	// get current pool pair and price
	X := reserveCoins[0].Amount.ToDec()
	Y := reserveCoins[1].Amount.ToDec()
	currentYPriceOverX := X.Quo(Y)

	denomX := reserveCoins[0].Denom
	denomY := reserveCoins[1].Denom

	// get All swap msgs from pool batch, and make orderMap
	swapMsgs := k.GetAllLiquidityPoolBatchSwapMsgs(ctx, liquidityPoolBatch)
	orderMap, XtoY, YtoX := types.GetOrderMap(swapMsgs, denomX, denomY)

	// make orderbook by sort orderMap
	orderBook := orderMap.SortOrderBook()

	// check orderbook validity and compute batchResult(direction, swapPrice, ..)
	fmt.Println("orderbook before batch")
	orderBookValidity := types.CheckValidityOrderBook(orderBook, currentYPriceOverX)
	result := types.ComputePriceDirection(X, Y, currentYPriceOverX, orderBook)
	fmt.Println("batch Result before", result)

	// find order match, calculate pool delta with the total x, y amount for the invariant check
	fmt.Println("before XtoY, YtoX", len(XtoY), len(YtoX))
	beforeXtoYLen := len(XtoY)
	beforeYtoXLen := len(YtoX)
	var matchResultXtoY, matchResultYtoX []types.MatchResult
	//var poolXDeltaXtoY, poolXDeltaYtoX, poolYDeltaYtoX, poolXdelta, poolYdelta  sdk.Int
	poolXdelta := sdk.ZeroInt()
	poolYdelta := sdk.ZeroInt()
	if result.MatchType != types.NoMatch {
		var poolXDeltaXtoY, poolXDeltaYtoX, poolYDeltaYtoX, poolYDeltaXtoY sdk.Int
		matchResultXtoY, _, poolXDeltaXtoY, poolYDeltaXtoY = types.FindOrderMatch(types.DirectionXtoY, XtoY, result.EX, result.SwapPrice, params.SwapFeeRate, ctx.BlockHeight())
		matchResultYtoX, _, poolXDeltaYtoX, poolYDeltaYtoX = types.FindOrderMatch(types.DirectionYtoX, YtoX, result.EY, result.SwapPrice, params.SwapFeeRate, ctx.BlockHeight())
		poolXdelta = poolXDeltaXtoY.Add(poolXDeltaYtoX)
		poolYdelta = poolYDeltaXtoY.Add(poolYDeltaYtoX)
	}

	//fmt.Println("mid XtoY, YtoX", len(XtoY), len(YtoX), len(matchResultXtoY), len(matchResultYtoX))
	XtoY, YtoX, X, Y, poolXdelta2, poolYdelta2, fractionalCntX, fractionalCntY := types.UpdateState(X, Y, XtoY, YtoX, matchResultXtoY, matchResultYtoX)

	lastPrice := X.Quo(Y)
	fmt.Println("lastPrice ", lastPrice)

	//fmt.Println(result, matchResultXtoY, matchResultYtoX, poolXdelta, poolYdelta, poolXdelta2, poolYdelta2)
	fmt.Println("result.SwapPrice, X, Y, currentYPriceOverX", result.SwapPrice, X, Y, currentYPriceOverX)
	//fmt.Println("after XtoY, YtoX", len(XtoY), len(YtoX), len(matchResultXtoY), len(matchResultYtoX))
	if beforeXtoYLen-len(matchResultXtoY)+fractionalCntX != len(XtoY){
		fmt.Println("!! match invariant Fail X")
		sdk.ZeroDec().Quo(sdk.ZeroDec()) // panic
	}
	if beforeYtoXLen-len(matchResultYtoX)+fractionalCntY != len(YtoX){
		fmt.Println("!! match invariant Fail Y")
		sdk.ZeroDec().Quo(sdk.ZeroDec()) // panic
	}

	totalAmtX := sdk.ZeroInt()
	totalAmtY := sdk.ZeroInt()

	for _, mr := range matchResultXtoY {
		fmt.Println("matchResultXtoY", mr)
		totalAmtX = totalAmtX.Sub(mr.MatchedAmt)
		totalAmtY = totalAmtY.Add(mr.ReceiveAmt)
	}

	invariantCheckX := totalAmtX
	invariantCheckY := totalAmtY

	totalAmtX = sdk.ZeroInt()
	totalAmtY = sdk.ZeroInt()

	for _, mr := range matchResultYtoX {
		fmt.Println("matchResultYtoX", mr)
		totalAmtY = totalAmtY.Sub(mr.MatchedAmt)
		totalAmtX = totalAmtX.Add(mr.ReceiveAmt)
	}

	invariantCheckX = invariantCheckX.Add(totalAmtX)
	invariantCheckY = invariantCheckY.Add(totalAmtY)

	invariantCheckX = invariantCheckX.Add(poolXdelta)  // TODO: compare with pooldelta2
	invariantCheckY = invariantCheckY.Add(poolYdelta)

	// print the invariant check and validity with swap, match result
	if invariantCheckX.IsZero() && invariantCheckY.IsZero() {
		fmt.Println("swap execution invariant check: True")
	} else {
		fmt.Println("swap execution invariant check: False", invariantCheckX, invariantCheckY)
		sdk.ZeroDec().Quo(sdk.ZeroDec()) // panic
	}

	if result.MatchType == 1 {
		fmt.Println("matchType: ", "ExactMatch")
	} else if result.MatchType == 2 {
		fmt.Println("matchType: ", "No Match")
	} else if result.MatchType == 3 {
		fmt.Println("matchType: ", "FractionalMatch")
	}

	fmt.Println("swapPrice: ", result.SwapPrice)
	fmt.Println("matchResultXtoY: ", matchResultXtoY)
	fmt.Println("matchResultYtoX: ", matchResultYtoX)
	fmt.Println("matched totalAmtX, totalAmtY", totalAmtX, totalAmtY)
	fmt.Println("poolXdelta, poolYdelta", poolXdelta, poolYdelta, poolXdelta2, poolYdelta2)

	if !poolXdelta.Equal(poolXdelta2) || !poolYdelta.Equal(poolYdelta2) {
		panic(poolXdelta)
	}

	// TODO: updateState, cancelEndOfLifeSpanOrders
	XtoY, YtoX = types.ClearOrders(XtoY, YtoX)

	orderMapExecuted, _, _ := types.GetOrderMap(append(XtoY, YtoX...), denomX, denomY)
	orderBookExecuted := orderMapExecuted.SortOrderBook()
	fmt.Println("orderbook after batch")
	orderBookValidity = types.CheckValidityOrderBook(orderBookExecuted, lastPrice)
	fmt.Println("after orderBookValidity", orderBookValidity)
	if !orderBookValidity {
		fmt.Println(orderBookValidity, "ErrOrderBookInvalidity", orderBookExecuted)
		panic(orderBookValidity)
		sdk.ZeroDec().Quo(sdk.ZeroDec())  // panic
		//return types.ErrOrderBookInvalidity
	}

	// TODO: updateState with escrow, emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSwap,
		),
	)
	return nil
}

