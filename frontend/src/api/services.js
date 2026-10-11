import { request } from "./client";

export const tradingApi = {
  // TODO: Match these paths with the Gin gateway contract.
  getBalance: () => request("/api/trading/balance"),
  searchAssets: (query) => request(`/api/trading/assets?search=${encodeURIComponent(query)}`),
  createOrder: (order) => request("/api/trading/orders", { method: "POST", body: order }),
};

export const weatherApi = {
  getCurrentWeather: (city) => request(`/api/weather/widgets/current_weather?city=${encodeURIComponent(city)}`),
};

export const githubApi = {
  getActivity: (repo) => request(`/api/github/activity?repo=${encodeURIComponent(repo)}`),
  connect: () => request("/api/github/connect"),
  disconnect: () => request("/api/github/connection", { method: "DELETE" }),
};
