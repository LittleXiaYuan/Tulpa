package main

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBacktestUsesNextOpenAndAppliesCosts(t *testing.T) {
	candles := []Candle{
		{Time: 0, Open: 1, Close: 1},
		{Time: 60000, Open: 1, Close: 2},
		{Time: 120000, Open: 2, Close: 3},
		{Time: 180000, Open: 4, Close: 5},
		{Time: 240000, Open: 6, Close: 7},
	}
	cfg := strategy{kind: "sma", fast: 2, slow: 3, fee: .001, slippage: .002}
	result := backtest(candles, time.Minute, cfg)
	if len(result.Ledger) != 1 || result.Trades != 1 {
		t.Fatalf("expected one trade, got %+v", result)
	}
	trade := result.Ledger[0]
	if trade.EntryTime != candles[3].Time || trade.ExitTime != 300000 || trade.Reason != "window_end" {
		t.Fatalf("unexpected fills: %+v", trade)
	}
	expected := 100 * (7*.998*.999/(4*1.002*1.001) - 1)
	for name, got := range map[string]float64{"strategy": result.Return, "benchmark": result.Benchmark, "trade": trade.Return} {
		if math.Abs(got-expected) > 1e-9 {
			t.Errorf("%s return = %f, want %f", name, got, expected)
		}
	}
}

func TestMarketAcceptsStrategyParameters(t *testing.T) {
	path := "/api/market?instrument=BTC-USDT&bar=1H&mode=demo&strategy=ema&fast=8&slow=21&feeBps=12&slippageBps=4"
	w := httptest.NewRecorder()
	market(w, httptest.NewRequest(http.MethodGet, path, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("got %d: %s", w.Code, w.Body)
	}
	var output Market
	if err := json.Unmarshal(w.Body.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	want := backtest(output.Candles, time.Hour, strategy{kind: "ema", fast: 8, slow: 21, fee: .0012, slippage: .0004})
	if output.Result.Error != "" || output.Result.Return != want.Return || output.Result.Trades != len(output.Result.Ledger) {
		t.Fatalf("unexpected EMA result: %+v", output.Result)
	}
	w = httptest.NewRecorder()
	market(w, httptest.NewRequest(http.MethodGet, "/api/market?instrument=BTC-USDT&bar=1H&mode=demo&fast=26&slow=26", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for equal periods, got %d", w.Code)
	}
}
