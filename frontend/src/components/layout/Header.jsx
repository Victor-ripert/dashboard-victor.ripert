import { Bell, Moon, Plus, Sun } from "lucide-react";
import Brand from "../ui/Brand";

export default function Header({ dark, onToggleTheme, onOpenWidgets }) {
  return (
    <header className="sticky top-0 z-40 border-b border-white/8 bg-shell/90 backdrop-blur-xl dark:bg-shell/90">
      <nav className="mx-auto flex items-center justify-between px-6 py-4">
        <div className="flex min-h-10 items-center rounded-md bg-white/[0.025] px-3" aria-label="Dashboard">
          <Brand />
        </div>
        <div className="flex items-center gap-3">
          <button className="icon-btn border border-white/8" onClick={onToggleTheme} aria-label="Toggle color theme">{dark ? <Sun size={18} /> : <Moon size={18} />}</button>
          <button className="icon-btn border border-white/8" aria-label="Notifications"><Bell size={18} /></button>
          <button onClick={onOpenWidgets} className="inline-flex min-h-10 items-center gap-2 rounded-md border border-white/10 bg-white/[0.035] px-3 text-sm font-semibold text-slate-300 hover:bg-white/[0.07] hover:text-white"><Plus size={17} /> <span className="hidden sm:inline">Add widget</span></button>
          <button className="grid size-10 place-items-center rounded-full bg-sky-400 text-sm font-bold text-slate-950" aria-label="User profile">AM</button>
        </div>
      </nav>
    </header>
  );
}
