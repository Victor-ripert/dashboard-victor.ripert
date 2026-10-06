import { useState } from "react";
import { CloudSun, TrendingUp } from "lucide-react";
import Header from "../components/layout/Header";
import WidgetPicker from "../components/layout/WidgetPicker";
import Widget from "../components/ui/Widget";
import TradingWidget from "../components/widgets/TradingWidget";
import WeatherWidget from "../components/widgets/WeatherWidget";
import { useAuth } from "../auth/useAuth";

const widgetDefinitions = [
  { id: "trading", title: "Trading", icon: TrendingUp, size: "span-4 row-2", component: TradingWidget},
  { id: "weather", title: "Weather", icon: CloudSun, size: "span-3 row-2", component: WeatherWidget},
];

export default function DashboardPage() {
  const { user, logout } = useAuth();
  const [dark, setDark] = useState(true);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [visible, setVisible] = useState(widgetDefinitions.map((widget) => widget.id));

  function toggleTheme() {
    const next = !dark;
    setDark(next);
    document.documentElement.classList.toggle("dark", next);
  }
  function toggleWidget(id) { setVisible((items) => items.includes(id) ? items.filter((item) => item !== id) : [...items, id]); }
  function hideWidget(id) { setVisible((items) => items.filter((item) => item !== id)); }

  return (
    <div className="min-h-screen bg-slate-100 text-slate-900 dark:bg-shell dark:text-slate-100">
      <Header dark={dark} user={user} onToggleTheme={toggleTheme} onOpenWidgets={() => setPickerOpen(true)} onLogout={logout} />
      <main className="mx-auto max-w-[1800px] p-3 sm:p-5">
        {visible.length ? <section className="dashboard-grid" aria-label="Service widgets">{widgetDefinitions.filter((widget) => visible.includes(widget.id)).map(({ component: Component, ...widget }) => <Widget key={widget.id} {...widget} onHide={hideWidget}><Component /></Widget>)}</section> : <section className="grid min-h-[60vh] place-items-center rounded-lg border border-dashed border-white/10"><div className="text-center"><p className="text-lg font-bold">No widgets displayed</p><button onClick={() => setPickerOpen(true)} className="mt-3 rounded-md bg-sky-400 px-4 py-2 text-sm font-bold text-slate-950">Add a widget</button></div></section>}
      </main>
      <WidgetPicker open={pickerOpen} widgets={widgetDefinitions} visible={visible} onToggle={toggleWidget} onClose={() => setPickerOpen(false)} />
    </div>
  );
}
