import { useMemo, useState } from "react";
import { ArrowDown, ArrowUp, Search, Wallet } from "lucide-react";
import { assets } from "../../data/demoData";

export default function TradingWidget() {
  const [side, setSide] = useState("buy");
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState(assets[0]);
  const [quantity, setQuantity] = useState("1");
  const [message, setMessage] = useState("");
  const filtered = assets.filter((asset) => `${asset.symbol} ${asset.name}`.toLowerCase().includes(query.toLowerCase()));
  const total = useMemo(() => selected.price * Math.max(Number(quantity) || 0, 0), [selected, quantity]);

  function choose(asset) { setSelected(asset); setQuery(""); setMessage(""); }
  function submit(event) {
    event.preventDefault();
    setMessage(`${side === "buy" ? "Buy" : "Sell"} order prepared: ${quantity || 0} ${selected.symbol}`);
  }

  return (
    <form onSubmit={submit} className="flex h-full flex-col justify-between text-xs">
      <div className="grid grid-cols-2 gap-2 border-b border-white/8 pb-2">
        <div>
          <p className="label">Balence</p>
          <p className="mono text-sm font-semibold light-text">$38,500.00</p>
        </div>
        <div className="text-right">
          <p className="label">Portfolio</p>
          <p className="mono text-sm font-semibold light-text">$124,820.40</p>
        </div>
      </div>

      <div className="mt-2 grid grid-cols-2 gap-1 rounded bg-white/[0.035] p-0.5">
        <button
          type="button"
          onClick={() => setSide("buy")}
          className={`h-7 rounded text-xs font-bold transition ${
            side === "buy" ? "bg-emerald-400 text-slate-950 shadow-sm" : "text-slate-400 hover:text-white"
          }`}
        >
          <ArrowDown className="mr-0.5 inline" size={13} />Buy
        </button>
        <button
          type="button"
          onClick={() => setSide("sell")}
          className={`h-7 rounded text-xs font-bold transition ${
            side === "sell" ? "bg-rose-400 text-slate-950 shadow-sm" : "text-slate-400 hover:text-white"
          }`}
        >
          <ArrowUp className="mr-0.5 inline" size={13} />Sell
        </button>
      </div>

      <div className="relative mt-2">
        <label htmlFor="asset-search" className="label">Actif</label>
        <Search size={13} className="absolute left-2.5 top-[27px] text-slate-500" />
        <input
          id="asset-search"
          className="field mt-1 pl-8"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={`${selected.symbol} — ${selected.name}`}
          autoComplete="off"
        />
        {query && (
          <div className="absolute z-20 mt-1 max-h-36 w-full overflow-auto rounded-md border border-white/10 bg-[#12202a] p-1 shadow-xl">
            {filtered.length ? (
              filtered.map((asset) => (
                <button
                  type="button"
                  key={asset.symbol}
                  onClick={() => choose(asset)}
                  className="flex h-8 w-full items-center justify-between rounded px-2 text-left hover:bg-white/[0.06]"
                >
                  <span>
                    <strong>{asset.symbol}</strong>
                    <span className="ml-1 text-[10px] text-slate-500">{asset.name}</span>
                  </span>
                  <span className="mono text-xs">${asset.price.toLocaleString()}</span>
                </button>
              ))
            ) : (
              <p className="p-2 text-xs text-slate-500">Can't place order, not enough money</p>
            )}
          </div>
        )}
      </div>

      <div className="mt-2 grid grid-cols-2 gap-2">
        <label>
          <span className="label">Volume</span>
          <input className="field mono mt-1" type="number" min="0" step="0.01" value={quantity} onChange={(event) => setQuantity(event.target.value)}
          />
        </label>
        <div>
          <p className="label">Price/title (before sprade)</p>
          <div className="soft-bg mt-1 flex h-8 items-center justify-between rounded-md border border-white/8 bg-white/[0.025] px-2">
            <span className="mono light-text text-xs">${selected.price.toLocaleString()}</span>
            <span className={`text-[10px] font-semibold ${selected.change >= 0 ? "text-emerald-400" : "text-rose-400"}`}>
              {selected.change > 0 ? "+" : ""}{selected.change}%
            </span>
          </div>
        </div>
      </div>

      <div className="mt-2 flex items-center justify-between rounded-md border border-white/8 bg-white/[0.025] px-2.5 py-1.5">
        <span className="text-xs text-slate-400">Total price</span>
        <strong className="mono light-text text-xs">${total.toLocaleString(undefined, { maximumFractionDigits: 2 })}</strong>
      </div>

      <button
        type="submit"
        className={`mt-2 flex h-8 w-full items-center justify-center rounded-md text-xs font-bold text-slate-950 transition active:scale-[.99] ${
          side === "buy" ? "bg-emerald-400 hover:bg-emerald-300" : "bg-rose-400 hover:bg-rose-300"
        }`}
      >
        <Wallet className="mr-1.5 inline" size={14} />
        {"Place an order"}
      </button>

      {message && (
        <p className="mt-2 rounded bg-sky-400/10 p-1.5 text-[11px] text-sky-300" role="status">
          {message}
        </p>
      )}
    </form>
  );
}
