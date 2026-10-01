export const assets = [
  { symbol: "AAPL", name: "Apple Inc.", price: 227.16, change: 1.24 },
  { symbol: "NVDA", name: "NVIDIA", price: 178.09, change: 2.81 },
  { symbol: "TSLA", name: "Tesla", price: 438.72, change: -0.64 },
  { symbol: "MSFT", name: "Microsoft", price: 511.41, change: 0.42 },
  { symbol: "BTC", name: "Bitcoin", price: 66384.12, change: 1.23 },
];

export const weatherByCity = {
  Paris: { temperature: 18, condition: "Partly cloudy", humidity: 62, wind: 11, icon: "partly" },
  London: { temperature: 14, condition: "Light rain", humidity: 78, wind: 17, icon: "rain" },
  Barcelona: { temperature: 24, condition: "Sunny", humidity: 48, wind: 9, icon: "sun" },
  Tokyo: { temperature: 21, condition: "Cloudy", humidity: 70, wind: 13, icon: "cloud" },
};

export const commits = [
  { message: "Add trading widget routes", hash: "a1b2c3d", time: "2 hours ago" },
  { message: "Fix gateway proxy timeout", hash: "d4e5f6g", time: "Yesterday" },
  { message: "Create weather service card", hash: "f7g8h9i", time: "2 days ago" },
];

export const articles = [
  { source: "TechWire", title: "Markets react to new semiconductor outlook", time: "2h" },
  { source: "Daily API", title: "Building resilient microservice gateways", time: "4h" },
  { source: "Dev Journal", title: "Designing useful dashboard widgets", time: "7h" },
  { source: "Systems Weekly", title: "Observability patterns for small services", time: "9h" },
];
