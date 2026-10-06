import { createElement, useEffect, useState } from "react";
import { Cloud, CloudFog, CloudLightning, CloudRain, CloudSnow, Loader2, Search, Sun, SunMedium } from "lucide-react";
import { weatherApi } from "../../api/services";

function iconFor(sky = "") {
  const text = sky.toLowerCase();
  if (text.includes("thunder")) return CloudLightning;
  if (text.includes("rain")) return CloudRain;
  if (text.includes("snow")) return CloudSnow;
  if (text.includes("fog")) return CloudFog;
  if (text.includes("partly")) return SunMedium;
  if (text.includes("clear")) return Sun;
  return Cloud;
}

export default function WeatherWidget({ defaultCity = "Paris" }) {
  const [query, setQuery] = useState(defaultCity);
  const [city, setCity] = useState(defaultCity);
  const [weather, setWeather] = useState(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  // Fetch the weather each time the searched city changes
  useEffect(() => {
    let ignore = false;
    weatherApi
      .getCurrentWeather(city)
      .then((data) => {
        if (ignore) return;
        setWeather(data);
        setError("");
      })
      .catch((err) => {
        if (!ignore) setError(err.message);
      })
      .finally(() => {
        if (!ignore) setLoading(false);
      });
    // Ignore the answer if another city was searched in the meantime
    return () => { ignore = true; };
  }, [city]);

  function search(event) {
    event.preventDefault();
    const next = query.trim();
    if (!next || next === city) return;
    setLoading(true);
    setCity(next);
  }

  return (
    <div className="flex h-full flex-col justify-between text-xs">
      <form onSubmit={search} className="flex gap-1.5">
        <label className="relative flex-1">
          <span className="sr-only">City</span>
          <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-500" />
          <input
            className="field pl-8"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search a city"
          />
        </label>
        <button disabled={loading} className="grid h-8 min-w-10 place-items-center rounded-md bg-sky-400 px-3 text-xs font-bold text-slate-950 hover:bg-sky-300 disabled:opacity-60">
          {loading ? <Loader2 size={13} className="animate-spin" /> : "OK"}
        </button>
      </form>
      {error && <p className="mt-1 text-[11px] text-amber-300" role="alert">{error}</p>}

      {weather && (
        <div className="my-2 flex items-center justify-between">
          <div>
            <p className="text-xs text-slate-400">{weather.city}</p>
            <p className="mono text-2xl font-bold light-text">{weather.temperature_celsius}°C</p>
            <p className="text-xs text-slate-400">{weather.sky}</p>
            <p className="text-[11px] text-slate-500">Precipitation: {weather.precipitation_mm} mm</p>
          </div>
          {createElement(iconFor(weather.sky), { size: 44, strokeWidth: 1.4, className: "text-amber-300" })}
        </div>
      )}
    </div>
  );
}
