package cointop

import (
	"sync"
	"time"

	types "github.com/cdyfng/coind/cointop/common/api/types"
)

var coinslock sync.Mutex
var updatecoinsmux sync.Mutex

func (ct *Cointop) updateCoins() error {
	ct.debuglog("updateCoins()")
	coinslock.Lock()
	defer coinslock.Unlock()
	cachekey := ct.CacheKey("allCoinsSlugMap")

	var err error
	var allCoinsSlugMap map[string]types.Coin
	cached, found := ct.cache.Get(cachekey)
	_ = cached
	if found {
		// cache hit
		allCoinsSlugMap, _ = cached.(map[string]types.Coin)
		ct.debuglog("soft cache hit")
	}

	// cache miss
	if allCoinsSlugMap == nil {
		ct.debuglog("cache miss")
		ch := make(chan []types.Coin)
		err = ct.api.GetAllCoinData(ct.State.currencyConversion, ch)
		if err != nil {
			return err
		}

		for coins := range ch {
			go ct.processCoins(coins)
		}
	} else {
		ct.processCoinsMap(allCoinsSlugMap)
	}

	return nil
}

func (ct *Cointop) processCoinsMap(coinsMap map[string]types.Coin) {
	ct.debuglog("processCoinsMap()")
	var coins []types.Coin
	for _, v := range coinsMap {
		coins = append(coins, v)
	}

	ct.processCoins(coins)
}

func (ct *Cointop) processCoins(coins []types.Coin) {
	ct.debuglog("processCoins()")
	updatecoinsmux.Lock()
	defer updatecoinsmux.Unlock()

	ct.CacheAllCoinsSlugMap()

	for _, v := range coins {
		k := v.Name
		ilast, _ := ct.State.allCoinsSlugMap.Load(k)
		ct.State.allCoinsSlugMap.Store(k, &Coin{
			ID:               v.ID,
			Name:             v.Name,
			Symbol:           v.Symbol,
			Rank:             v.Rank,
			Price:            v.Price,
			Volume24H:        v.Volume24H,
			MarketCap:        v.MarketCap,
			AvailableSupply:  v.AvailableSupply,
			TotalSupply:      v.TotalSupply,
			PercentChange1H:  v.PercentChange1H,
			PercentChange24H: v.PercentChange24H,
			PercentChange7D:  v.PercentChange7D,
			PercentChange30D: v.PercentChange30D,
			PercentChange1Y:  v.PercentChange1Y,			
			LastUpdated:      v.LastUpdated,
		})
		if ilast != nil {
			last, _ := ilast.(*Coin)
			if last != nil {
				ivalue, _ := ct.State.allCoinsSlugMap.Load(k)
				l, _ := ivalue.(*Coin)
				l.Favorite = last.Favorite
				ct.State.allCoinsSlugMap.Store(k, l)
			}
		}
	}

	size := 0
	// NOTE: there's no Len method on sync.Map so need to manually count
	ct.State.allCoinsSlugMap.Range(func(key, value interface{}) bool {
		size++
		return true
	})

	if len(ct.State.allCoins) < size {
		list := []*Coin{}
		for _, v := range coins {
			k := v.Name
			icoin, _ := ct.State.allCoinsSlugMap.Load(k)
			coin, _ := icoin.(*Coin)
			list = append(list, coin)
		}
		ct.State.allCoins = append(ct.State.allCoins, list...)
	} else {
		// update list in place without changing order
		ct.State.allCoinsSlugMap.Range(func(key, value interface{}) bool {
			cm, _ := value.(*Coin)
			for k := range ct.State.allCoins {
				c := ct.State.allCoins[k]
				if c.ID == cm.ID {
					// TODO: improve this
					c.ID = cm.ID
					c.Name = cm.Name
					c.Symbol = cm.Symbol
					c.Rank = cm.Rank
					c.Price = cm.Price
					c.Volume24H = cm.Volume24H
					c.MarketCap = cm.MarketCap
					c.AvailableSupply = cm.AvailableSupply
					c.TotalSupply = cm.TotalSupply
					c.PercentChange1H = cm.PercentChange1H
					c.PercentChange24H = cm.PercentChange24H
					c.PercentChange7D = cm.PercentChange7D
					c.PercentChange30D = cm.PercentChange30D
					c.PercentChange1Y = cm.PercentChange1Y
					c.LastUpdated = cm.LastUpdated
					c.Favorite = cm.Favorite
				}
			}

			return true
		})
	}

	time.AfterFunc(10*time.Millisecond, func() {
		ct.sort(ct.State.sortBy, ct.State.sortDesc, ct.State.coins, true)
		ct.UpdateTable()
	})
}

func (ct *Cointop) getListCount() int {
	ct.debuglog("getListCount()")
	if ct.State.filterByFavorites {
		return len(ct.State.favorites)
	} else if ct.State.portfolioVisible {
		return len(ct.State.portfolio.Entries)
	} else {
		return len(ct.State.allCoins)
	}
}
