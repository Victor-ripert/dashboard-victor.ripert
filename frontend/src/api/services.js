async function request(path, options = {}) {
  const response = await fetch(path, {
    headers: { "Content-Type": "application/json", ...options.headers },
    ...options,
  });
  if (!response.ok) throw new Error(`Request failed with status ${response.status}`);
  return response.json();
}

export const tradingApi = {
  // TODO: Match these paths with the Gin gateway contract.
  getBalance: () => request("/api/trading/balance"),
  searchAssets: (query) => request(`/api/trading/assets?search=${encodeURIComponent(query)}`),
  createOrder: (order) => request("/api/trading/orders", { method: "POST", body: JSON.stringify(order) }),
};

export const weatherApi = {
  // TODO: Match this path with the weather microservice route exposed by Gin.
  getByCity: (city) => request(`/api/weather?city=${encodeURIComponent(city)}`),
};

export const githubApi = {
  // TODO: Replace with the route exposed by the future GitHub microservice.
  getActivity: () => request("/api/github/activity"),
};
