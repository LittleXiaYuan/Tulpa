package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Candle struct {
	Time   int64   `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}
type Market struct {
	Candles []Candle `json:"candles"`
	Price   float64  `json:"price"`
	Change  float64  `json:"change"`
	High    float64  `json:"high"`
	Low     float64  `json:"low"`
	Volume  float64  `json:"volume"`
	AsOf    int64    `json:"asOf"`
	Result  Result   `json:"result"`
}

var instruments = map[string]float64{"BTC-USDT": 67420, "ETH-USDT": 3520, "SOL-USDT": 168, "XRP-USDT": 0.58}
var bars = map[string]time.Duration{"15m": 15 * time.Minute, "1H": time.Hour, "4H": 4 * time.Hour, "1Dutc": 24 * time.Hour}
var client = &http.Client{Timeout: 7 * time.Second}
var origin = env("OKX_BASE_URL", "https://www.okx.com")
var cache sync.Map

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func okx[T any](ctx context.Context, path string, params url.Values) (T, error) {
	var response struct {
		Code string
		Msg  string
		Data T
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+"/api/v5/market/"+path+"?"+params.Encode(), nil)
	if err != nil {
		return response.Data, err
	}
	res, err := client.Do(req)
	if err != nil {
		return response.Data, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return response.Data, fmt.Errorf("OKX HTTP %d", res.StatusCode)
	}
	err = json.NewDecoder(res.Body).Decode(&response)
	if err == nil && response.Code != "0" {
		err = fmt.Errorf("OKX code %s: %s", response.Code, response.Msg)
	}
	return response.Data, err
}

func number(raw string) (float64, error) {
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 {
		return 0, fmt.Errorf("invalid market number %q", raw)
	}
	return n, nil
}

func live(ctx context.Context, instrument, bar string) (Market, error) {
	var m Market
	params := url.Values{"instId": {instrument}}
	tickers, err := okx[[]struct{ Last, Open24h, High24h, Low24h, VolCcy24h, Ts string }](ctx, "ticker", params)
	if err != nil {
		return m, err
	}
	if len(tickers) == 0 {
		return m, fmt.Errorf("empty OKX ticker")
	}
	t := tickers[0]
	values := []string{t.Last, t.Open24h, t.High24h, t.Low24h, t.VolCcy24h}
	parsed := make([]float64, len(values))
	for i, v := range values {
		if parsed[i], err = number(v); err != nil {
			return m, err
		}
	}
	if parsed[0] <= 0 || parsed[1] <= 0 {
		return m, fmt.Errorf("invalid ticker price")
	}
	m.Price, m.Change, m.High, m.Low, m.Volume = parsed[0], 100*(parsed[0]/parsed[1]-1), parsed[2], parsed[3], parsed[4]
	m.AsOf, err = strconv.ParseInt(t.Ts, 10, 64)
	if err != nil {
		return m, err
	}
	params.Set("bar", bar)
	params.Set("limit", "121")
	rows, err := okx[[][]string](ctx, "candles", params)
	if err != nil {
		return m, err
	}
	seen := map[int64]bool{}
	for _, row := range rows {
		if len(row) < 9 {
			return m, fmt.Errorf("invalid OKX candle")
		}
		if row[8] != "1" {
			continue
		}
		ts, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			return m, err
		}
		if seen[ts] {
			continue
		}
		var v [5]float64
		for i, field := range []int{1, 2, 3, 4, 7} {
			v[i], err = number(row[field])
			if err != nil {
				return m, err
			}
		}
		if v[0] <= 0 || v[3] <= 0 || v[1] < math.Max(v[0], v[3]) || v[2] > math.Min(v[0], v[3]) || v[2] <= 0 {
			return m, fmt.Errorf("invalid candle prices")
		}
		seen[ts] = true
		m.Candles = append(m.Candles, Candle{ts, v[0], v[1], v[2], v[3], v[4]})
	}
	sort.Slice(m.Candles, func(i, j int) bool { return m.Candles[i].Time < m.Candles[j].Time })
	return m, nil
}

func demo(instrument, bar string) Market {
	step := bars[bar]
	end := time.Now().UTC().Truncate(step)
	rng := rand.New(rand.NewSource(int64(step/time.Minute) + int64(instruments[instrument]*100)))
	base := instruments[instrument]
	last := base * .94
	m := Market{AsOf: end.UnixMilli()}
	for i := 0; i < 120; i++ {
		open := last
		last = base * (.94 + float64(i)*.00063 + .013*math.Sin(float64(i)*.135) + .008*rng.NormFloat64())
		high := math.Max(open, last) * (1 + .004*rng.Float64())
		low := math.Min(open, last) * (1 - .004*rng.Float64())
		volume := (.35 + rng.Float64()) * 1.4e6
		m.Candles = append(m.Candles, Candle{end.Add(time.Duration(i-120) * step).UnixMilli(), open, high, low, last, volume})
	}
	// Use a dedicated hourly sample for the 24-hour figures at every chart interval.
	if bar != "1H" {
		hourly := demo(instrument, "1H")
		m.AsOf, m.Price, m.Change, m.High, m.Low, m.Volume = hourly.AsOf, hourly.Price, hourly.Change, hourly.High, hourly.Low, hourly.Volume
		ratio := m.Price / last
		for i := range m.Candles {
			c := &m.Candles[i]
			c.Open *= ratio
			c.High *= ratio
			c.Low *= ratio
			c.Close *= ratio
		}
		return m
	}
	m.Price = last
	m.Low = math.Inf(1)
	window := m.Candles[96:]
	m.Change = 100 * (last/window[0].Open - 1)
	for _, c := range window {
		m.High = math.Max(m.High, c.High)
		m.Low = math.Min(m.Low, c.Low)
		m.Volume += c.Volume
	}
	return m
}

type snapshot struct {
	market  Market
	expires time.Time
}

func market(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	instrument, bar, mode := q.Get("instrument"), q.Get("bar"), q.Get("mode")
	if instruments[instrument] == 0 || bars[bar] == 0 || (mode != "live" && mode != "demo") {
		http.Error(w, "Invalid instrument, interval, or mode", http.StatusBadRequest)
		return
	}
	cfg, err := parseStrategy(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var m Market
	if mode == "demo" {
		m = demo(instrument, bar)
	} else {
		key := instrument + " " + bar
		if cached, ok := cache.Load(key); ok && time.Now().Before(cached.(snapshot).expires) {
			m = cached.(snapshot).market
		} else {
			var err error
			m, err = live(r.Context(), instrument, bar)
			if err != nil {
				log.Printf("market %s: %v", key, err)
				http.Error(w, "OKX market data is unavailable. Retry or switch to Demo.", http.StatusBadGateway)
				return
			}
			if len(m.Candles) < 28 {
				http.Error(w, "OKX returned too few confirmed candles.", http.StatusBadGateway)
				return
			}
			cache.Store(key, snapshot{m, time.Now().Add(10 * time.Second)})
		}
	}
	m.Result = backtest(m.Candles, bars[bar], cfg)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(m); err != nil {
		log.Print(err)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/market", market)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok\n")) })
	server := &http.Server{Addr: env("LISTEN_ADDR", "127.0.0.1:8080"), Handler: mux, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 20 * time.Second}
	log.Printf("Go market API listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
