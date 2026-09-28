import express from "express";

const app = express();
const PORT = process.env.PORT || 4000;

// Metadata the gateway fetches to build the aggregated /about.json
const ABOUT = {
  name: "weather",
  widgets: [
    {
      name: "current_weather",
      description: "Sky overview, temperature and precipitation for a city",
      params: [{ name: "city", type: "string" }],
    },
  ],
};

app.get("/about.json", (req, res) => {
  res.json(ABOUT);
});

const WEATHER_CODES = {
  0: "Clear sky",
  1: "Mainly clear",
  2: "Partly cloudy",
  3: "Overcast",
  45: "Fog",
  61: "Slight rain",
  63: "Moderate rain",
  65: "Heavy rain",
  71: "Slight snow",
  95: "Thunderstorm",
};

async function geocode(city) {
  const url = `https://geocoding-api.open-meteo.com/v1/search?name=${encodeURIComponent(city)}&count=1`;
  const res = await fetch(url);
  const data = await res.json();
  if (!data.results?.length) throw new Error(`City not found: ${city}`);
  const { latitude, longitude, name, country } = data.results[0];
  return { latitude, longitude, name, country };
}

// GET /widgets/current_weather?city=Paris
app.get("/widgets/current_weather", async (req, res) => {
  const { city } = req.query;
  if (!city) return res.status(400).json({ error: "Missing required param: city" });

  try {
    const location = await geocode(city);
    const forecastUrl = `https://api.open-meteo.com/v1/forecast?latitude=${location.latitude}&longitude=${location.longitude}&current=temperature_2m,precipitation,weather_code`;
    const forecast = await (await fetch(forecastUrl)).json();

    res.json({
      city: `${location.name}, ${location.country}`,
      temperature_celsius: forecast.current.temperature_2m,
      precipitation_mm: forecast.current.precipitation,
      sky: WEATHER_CODES[forecast.current.weather_code] ?? "Unknown",
      updated_at: forecast.current.time,
    });
  } catch (err) {
    res.status(502).json({ error: err.message });
  }
});

app.listen(PORT, () => {
  console.log(`Weather service listening on port ${PORT}`);
});
