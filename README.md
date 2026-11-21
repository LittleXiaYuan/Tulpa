# Tulpa Terminal

A local quantitative research dashboard built with Go, Next.js, and the public OKX v5 market-data API.

## Run locally

Requirements: Go 1.23 or later and Node.js 20.9 or later.

Run the API in one terminal:

```sh
cd backend
go run .
```

Run the dashboard in another:

```sh
cd web
npm ci
npm run dev
```

Open http://localhost:3000. The dashboard starts in **Demo** with synthetic OHLCV data. Select **Live OKX** to fetch public spot data. No API keys are required. Next.js proxies requests to the Go API at `http://127.0.0.1:8080` by default.

- `LISTEN_ADDR` changes the Go bind address.
- `OKX_BASE_URL` changes the OKX API origin.
- `TULPA_API_URL` changes the backend origin used by Next.js. Set it before starting development or building for production.
- `GET /healthz` checks the Go process. `GET /api/market?instrument=BTC-USDT&bar=1H&mode=live` retrieves market data and strategy results.

The live view polls every 15 seconds and retries after 30 seconds on failure. It labels feed errors and retains the last successful snapshot for the selected market and interval.

## Strategy model

Select BTC, ETH, SOL, or XRP against USDT, and a 15-minute, hourly, four-hour, or UTC daily interval. The Strategy lab supports SMA and EMA trend rules with configurable fast and slow periods. A fast average above the slow average targets a long position. An equal or lower value targets cash. Signals use confirmed closes and fills use the next bar's open.

Strategy and buy-and-hold start after the same warm-up, use the same per-side fee and adverse slippage settings, and close positions at the final close. The defaults are SMA 12/26, 10 bps fees, and zero slippage. The equity curve marks open positions at bar close. The trade ledger shows simulated fills, net per-trade return, and exit reasons. The latest signal describes the target for the next open.

The market API accepts optional query parameters `strategy=sma|ema`, `fast=2–99`, `slow=3–100`, `feeBps=0–500`, and `slippageBps=0–500`. Use integer values and a fast period smaller than the slow period.

The dashboard uses public market data and simulated execution. There are no account or order endpoints.

[OKX v5 API documentation](https://www.okx.com/docs-v5/en/#order-book-trading-market-data)
