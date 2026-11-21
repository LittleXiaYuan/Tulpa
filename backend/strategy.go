package main

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"
)

type strategy struct {
	kind          string
	fast, slow    int
	fee, slippage float64
}

type Trade struct {
	EntryTime  int64   `json:"entryTime"`
	ExitTime   int64   `json:"exitTime"`
	EntryPrice float64 `json:"entryPrice"`
	ExitPrice  float64 `json:"exitPrice"`
	Return     float64 `json:"return"`
	Reason     string  `json:"reason"`
}

type Result struct {
	Return     float64   `json:"return"`
	Benchmark  float64   `json:"benchmark"`
	Drawdown   float64   `json:"drawdown"`
	Trades     int       `json:"trades"`
	Ledger     []Trade   `json:"ledger"`
	Equity     []float64 `json:"equity"`
	Signal     string    `json:"signal"`
	SignalTime int64     `json:"signalTime"`
	Error      string    `json:"error,omitempty"`
}

func boundedInt(q url.Values, key string, fallback, min, max int) (int, error) {
	raw := q.Get(key)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < min || value > max {
		return 0, fmt.Errorf("%s must be an integer from %d to %d", key, min, max)
	}
	return value, nil
}

func parseStrategy(q url.Values) (strategy, error) {
	cfg := strategy{kind: q.Get("strategy")}
	if cfg.kind == "" {
		cfg.kind = "sma"
	}
	if cfg.kind != "sma" && cfg.kind != "ema" {
		return cfg, fmt.Errorf("strategy must be sma or ema")
	}
	var err error
	if cfg.fast, err = boundedInt(q, "fast", 12, 2, 99); err != nil {
		return cfg, err
	}
	if cfg.slow, err = boundedInt(q, "slow", 26, 3, 100); err != nil {
		return cfg, err
	}
	if cfg.fast >= cfg.slow {
		return cfg, fmt.Errorf("fast period must be smaller than slow period")
	}
	fee, err := boundedInt(q, "feeBps", 10, 0, 500)
	if err != nil {
		return cfg, err
	}
	slippage, err := boundedInt(q, "slippageBps", 0, 0, 500)
	if err != nil {
		return cfg, err
	}
	cfg.fee, cfg.slippage = float64(fee)/10000, float64(slippage)/10000
	return cfg, nil
}

// The first defined average is seeded with the SMA at period-1.
func movingAverage(candles []Candle, period int, kind string) []float64 {
	out := make([]float64, len(candles))
	var sum float64
	for _, c := range candles[:period] {
		sum += c.Close
	}
	out[period-1] = sum / float64(period)
	alpha := 2 / float64(period+1)
	for i := period; i < len(candles); i++ {
		if kind == "ema" {
			out[i] = out[i-1] + alpha*(candles[i].Close-out[i-1])
		} else {
			sum += candles[i].Close - candles[i-period].Close
			out[i] = sum / float64(period)
		}
	}
	return out
}

func backtest(candles []Candle, interval time.Duration, cfg strategy) Result {
	if len(candles) <= cfg.slow {
		return Result{Error: fmt.Sprintf("Need at least %d confirmed candles for this strategy.", cfg.slow+1)}
	}
	fast, slow := movingAverage(candles, cfg.fast, cfg.kind), movingAverage(candles, cfg.slow, cfg.kind)
	last := len(candles) - 1
	r := Result{Equity: make([]float64, 0, len(candles)), Ledger: make([]Trade, 0), Signal: "cash", SignalTime: candles[last].Time + interval.Milliseconds()}
	if fast[last] > slow[last] {
		r.Signal = "long"
	}

	cash, units, peak := 1.0, 0.0, 1.0
	var entry Trade
	var entryEquity float64
	buyPrice := func(price float64) float64 { return price * (1 + cfg.slippage) }
	sellPrice := func(price float64) float64 { return price * (1 - cfg.slippage) }
	sell := func(price float64, timestamp int64, reason string) {
		entry.ExitTime, entry.ExitPrice, entry.Reason = timestamp, sellPrice(price), reason
		cash = units * entry.ExitPrice * (1 - cfg.fee)
		units = 0
		entry.Return = 100 * (cash/entryEquity - 1)
		r.Ledger = append(r.Ledger, entry)
	}

	for i, c := range candles {
		if i >= cfg.slow {
			wantLong := fast[i-1] > slow[i-1]
			if wantLong && units == 0 {
				entry = Trade{EntryTime: c.Time, EntryPrice: buyPrice(c.Open)}
				entryEquity = cash
				units = cash / (entry.EntryPrice * (1 + cfg.fee))
				cash = 0
			} else if !wantLong && units > 0 {
				sell(c.Open, c.Time, "signal")
			}
		}
		if i == last && units > 0 {
			sell(c.Close, r.SignalTime, "window_end")
		}
		equity := cash + units*c.Close
		peak = math.Max(peak, equity)
		r.Drawdown = math.Max(r.Drawdown, 100*(1-equity/peak))
		r.Equity = append(r.Equity, equity)
	}
	r.Trades = len(r.Ledger)
	r.Return = 100 * (cash - 1)
	startCost := buyPrice(candles[cfg.slow].Open) * (1 + cfg.fee)
	endProceeds := sellPrice(candles[last].Close) * (1 - cfg.fee)
	r.Benchmark = 100 * (endProceeds/startCost - 1)
	return r
}
