# Microservice Widget Dashboard

This version follows a widget model: **one interactive card per microservice**. It is intentionally closer to the attached reference than a traditional finance dashboard.

## Run

```bash
npm install
npm run dev
```

The Vite server proxies `/api/*` to the Gin gateway at `http://localhost:8080`. Set `GATEWAY_URL` (see `.env.example`) to use another address.

## Included service cards

- Trading: balance, asset search/selection, buy/sell mode, quantity, estimated total.
- Weather: city search and weather details.
- GitHub activity: contribution grid and recent commits.
- Service health: gateway and microservice status.
- News feed: compact article list.
- Notes: local interactive project notes.
- Widget picker: hide and restore cards with `Add widget`.

The weather card uses real data from the gateway. The other cards still use demo data from `src/data/demoData.js`.

## Code structure

```text
src/
├── api/
│   ├── client.js        # request(): the only place that calls fetch, adds the token, turns errors into ApiError
│   ├── auth.js          # register / login / me
│   └── services.js      # one object per microservice (weatherApi, tradingApi...)
├── auth/
│   ├── AuthProvider.jsx # holds the logged-in user, exposes login / register / logout
│   ├── useAuth.js       # hook to read the auth state from any component
│   ├── authContext.js   # the React context shared by the two files above
│   ├── tokenStorage.js  # where the token is kept (localStorage)
│   ├── RequireAuth.jsx  # route guard: logged-in users only
│   └── GuestOnly.jsx    # route guard: logged-out visitors only
├── components/auth/AuthForm.jsx  # email + password form used by both auth pages
└── pages/               # LoginPage, RegisterPage, DashboardPage
```

Routes: `/login`, `/register`, and `/` (the dashboard, login required).

## Gateway routes

```text
GET  /api/trading/balance
GET  /api/trading/assets?search=AAPL
POST /api/trading/orders
GET  /api/weather/widgets/current_weather?city=Paris   (implemented)
GET  /api/github/activity
GET  /api/health
```

## Add a microservice card

1. Create `src/components/widgets/MyServiceWidget.jsx`.
2. Add its gateway calls to `src/api/services.js`.
3. Add one entry to `widgetDefinitions` in `src/pages/DashboardPage.jsx`.
4. Choose its grid size, for example `span-3 row-2` or `span-6 row-3`.
