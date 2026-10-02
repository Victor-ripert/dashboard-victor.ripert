# Microservice Widget Dashboard

This version follows a widget model: **one interactive card per microservice**. It is intentionally closer to the attached reference than a traditional finance dashboard.

## Run

```bash
npm install
npm run dev
```

The Vite server proxies `/api/*` to the Gin gateway at `http://localhost:8080`.

## Included service cards

- Trading: balance, asset search/selection, buy/sell mode, quantity, estimated total.
- Weather: city search and weather details.
- GitHub activity: contribution grid and recent commits.
- Service health: gateway and microservice status.
- News feed: compact article list.
- Notes: local interactive project notes.
- Widget picker: hide and restore cards with `Add widget`.

All displayed values are explicit demo data. API calls are isolated in `src/api/services.js`, with TODO comments where routes must be matched to the Gin gateway.

## Recommended gateway routes

```text
GET  /api/trading/balance
GET  /api/trading/assets?search=AAPL
POST /api/trading/orders
GET  /api/weather?city=Paris
GET  /api/github/activity
GET  /api/health
```

## Add a microservice card

1. Create `src/components/widgets/MyServiceWidget.jsx`.
2. Add its gateway calls to `src/api/services.js`.
3. Add one entry to `widgetDefinitions` in `src/App.jsx`.
4. Choose its grid size, for example `span-3 row-2` or `span-6 row-3`.
