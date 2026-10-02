import { useState } from "react";
import { Cloud, CloudRain, Search, Sun, SunMedium } from "lucide-react";
import { weatherByCity } from "../../data/demoData";

const icons = { partly: SunMedium, rain: CloudRain, sun: Sun, cloud: Cloud };

export default function WeatherWidget() {
  const [city, setCity] = useState("Paris");
  const [query, setQuery] = useState("Paris");
  const [error, setError] = useState("");
  const weather = weatherByCity[city];
  const WeatherIcon = icons[weather.icon];

  function search(event) {
    event.preventDefault();
    const match = Object.keys(weatherByCity).find((name) => name.toLowerCase() === query.trim().toLowerCase());
    if (!match) { setError("Demo cities: Paris, London, Barcelona or Tokyo."); return; }
    setCity(match); setError("");
    // TODO: Replace demo lookup with weatherApi.getByCity(query).
  }

  return (
    <div className="flex h-full flex-col justify-between text-xs">
      <form onSubmit={search} className="flex gap-1.5">
        <label className="relative flex-1">
          <span className="sr-only">Ville</span>
          <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-500" />
          <input
            className="field pl-8"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Rechercher une ville"
          />
        </label>
        <button className="h-8 rounded-md bg-sky-400 px-3 text-xs font-bold text-slate-950 hover:bg-sky-300">
          OK
        </button>
      </form>
      {error && <p className="mt-1 text-[11px] text-amber-300" role="alert">{error}</p>}
      
      <div className="my-2 flex items-center justify-between">
        <div>
          <p className="text-xs text-slate-400">{city}</p>
          <p className="mono text-2xl font-bold light-text">{weather.temperature}°C</p>
          <p className="text-xs text-slate-400">{weather.condition}</p>
        </div>
        <WeatherIcon size={44} strokeWidth={1.4} className="text-amber-300" />
      </div>
    </div>
  );
}
