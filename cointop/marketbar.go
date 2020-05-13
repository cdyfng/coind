package cointop

import (
	"fmt"
	"math"
	"time"

	types "github.com/cdyfng/coind/cointop/common/api/types"
	"github.com/cdyfng/coind/cointop/common/color"
	"github.com/cdyfng/coind/cointop/common/filecache"
	"github.com/cdyfng/coind/cointop/common/humanize"
	"github.com/cdyfng/coind/cointop/common/pad"
)

// MarketbarView is structure for marketbar view
type MarketbarView struct {
	*View
}

// NewMarketbarView returns a new marketbar view
func NewMarketbarView() *MarketbarView {
	return &MarketbarView{NewView("marketbar")}
}

func (ct *Cointop) updateMarketbar() error {
	ct.debuglog("updateMarketbar()")
	if ct.Views.Marketbar.Backing() == nil {
		return nil
	}

	maxX := ct.width()
	logo := "❯❯❯cointop"
	if ct.colorschemeName == "cointop" {
		logo = fmt.Sprintf("%s%s%s%s", color.Green("❯"), color.Cyan("❯"), color.Green("❯"), color.Cyan("cointop"))
	}
	var content string

	if ct.State.portfolioVisible {
		total := ct.getPortfolioTotal()
		totalstr := humanize.Commaf(total)
		if !(ct.State.currencyConversion == "BTC" || ct.State.currencyConversion == "ETH" || total < 1) {
			total = math.Round(total*1e2) / 1e2
			totalstr = humanize.Commaf2(total)
		}

		timeframe := ct.State.selectedChartRange
		chartname := ct.selectedCoinName()
		var charttitle string
		if chartname == "" {
			chartname = "Portfolio"
			charttitle = ct.colorscheme.MarketBarLabelActive(chartname)
		} else {
			charttitle = fmt.Sprintf("Portfolio - %s", ct.colorscheme.MarketBarLabelActive(chartname))
		}

		var percentChange24H float64
		for _, p := range ct.getPortfolioSlice() {
			n := ((p.Balance / total) * p.PercentChange24H)
			if math.IsNaN(n) {
				continue
			}
			percentChange24H += n
		}

		color24h := ct.colorscheme.MarketbarSprintf()
		arrow := ""
		if percentChange24H > 0 {
			color24h = ct.colorscheme.MarketbarChangeUpSprintf()
			arrow = "▲"
		}
		if percentChange24H < 0 {
			color24h = ct.colorscheme.MarketbarChangeDownSprintf()
			arrow = "▼"
		}

		chartInfo := ""
		if !ct.State.hideChart {
			chartInfo = fmt.Sprintf(
				"[ Chart: %s %s ] ",
				charttitle,
				timeframe,
			)
		}

		content = fmt.Sprintf(
			"%sTotal Portfolio Value: %s • 24H: %s",
			chartInfo,
			ct.colorscheme.MarketBarLabelActive(fmt.Sprintf("%s%s", ct.currencySymbol(), totalstr)),
			color24h(fmt.Sprintf("%.2f%%%s", percentChange24H, arrow)),
		)
	} else {
		var market types.GlobalMarketData
		var err error
		cachekey := ct.CacheKey("market")
		cached, found := ct.cache.Get(cachekey)

		if found {
			// cache hit
			var ok bool
			market, ok = cached.(types.GlobalMarketData)
			if ok {
				ct.debuglog("soft cache hit")
			}
		}

		if market.TotalMarketCapUSD == 0 {
			market, err = ct.api.GetGlobalMarketData(ct.State.currencyConversion)
			if err != nil {
				filecache.Get(cachekey, &market)
			}

			ct.cache.Set(cachekey, market, 10*time.Second)
			go func() {
				filecache.Set(cachekey, market, 24*time.Hour)
			}()
		}

		timeframe := ct.State.selectedChartRange
		chartname := ct.selectedCoinName()
		if chartname == "" {
			chartname = "Global"
		}

		chartInfo := ""
		if !ct.State.hideChart {
			chartInfo = fmt.Sprintf(
				"[ Chart: %s %s ] ",
				ct.colorscheme.MarketBarLabelActive(chartname),
				timeframe,
			)
		}

		content = fmt.Sprintf(
			"%sGlobal ▶ Market Cap: %s • 24H Volume: %s • BTC Dominance: %.2f%%",
			chartInfo,
			fmt.Sprintf("%s%s", ct.currencySymbol(), humanize.Commaf0(market.TotalMarketCapUSD)),
			fmt.Sprintf("%s%s", ct.currencySymbol(), humanize.Commaf0(market.Total24HVolumeUSD)),
			market.BitcoinPercentageOfMarketCap,
		)
	}

	content = fmt.Sprintf("%s %s", logo, content)
	content = pad.Right(content, maxX, " ")
	content = ct.colorscheme.Marketbar(content)

	ct.Update(func() error {
		if ct.Views.Marketbar.Backing() == nil {
			return nil
		}

		ct.Views.Marketbar.Backing().Clear()
		fmt.Fprintln(ct.Views.Marketbar.Backing(), content)
		return nil
	})

	return nil
}
