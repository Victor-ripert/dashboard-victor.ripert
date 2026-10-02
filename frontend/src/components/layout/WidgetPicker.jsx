import { Check, Plus, X } from "lucide-react";

export default function WidgetPicker({ open, widgets, visible, onToggle, onClose }) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/55" role="dialog" aria-modal="true" aria-label="Choose widgets">
      <button className="absolute inset-0" onClick={onClose} aria-label="Close widget picker" />
      <aside className="relative h-full w-full max-w-sm border-l border-white/10 bg-panel p-5 shadow-2xl">
        <div className="flex items-center justify-between"><div><p className="label">Dashboard</p><h2 className="mt-1 text-xl font-bold">Manage widgets</h2></div><button className="icon-btn" onClick={onClose} aria-label="Close"><X size={20} /></button></div>
        <p className="mt-3 text-sm text-slate-500">Each widget represents one microservice and owns its controls and display state.</p>
        <div className="mt-6 space-y-2">{widgets.map(({ id, title, icon: Icon }) => { const selected = visible.includes(id); return <button key={id} onClick={() => onToggle(id)} className="flex min-h-14 w-full items-center gap-3 rounded-lg border border-white/8 bg-white/[0.025] px-4 text-left hover:bg-white/[0.055]"><Icon size={19} className="text-sky-400" /><span className="font-semibold">{title}</span><span className={`ml-auto grid size-7 place-items-center rounded-md ${selected ? "bg-emerald-400 text-slate-950" : "border border-white/10 text-slate-500"}`}>{selected ? <Check size={16} /> : <Plus size={16} />}</span></button>; })}</div>
      </aside>
    </div>
  );
}
