import { MoreVertical, X } from "lucide-react";

export default function Widget({ id, title, icon: Icon, status, size = "span-3 row-2", onHide, children, footer }) {
  return (
    <article className={`widget ${size}`} data-widget={id}>
      <header className="widget-header">
        <Icon size={15} className="text-sky-400" aria-hidden="true" />
        <h2 className="light-text text-xs font-bold text-slate-100">{title}</h2>
        {status && (
          <span className="ml-1 flex items-center gap-1 text-[10px] text-slate-500">
            <span className="size-1 rounded-full bg-emerald-400" />
            {status}
          </span>
        )}
        <div className="ml-auto flex items-center gap-0.5">
          <button className="icon-btn size-6 text-slate-400" aria-label={`Options pour ${title}`}>
            <MoreVertical size={13} />
          </button>
          <button className="icon-btn size-6 text-slate-400" onClick={() => onHide(id)} aria-label={`Masquer ${title}`}>
            <X size={13} />
          </button>
        </div>
      </header>
      <div className="widget-body">{children}</div>
      {footer && <footer className="border-t border-white/8 px-3 py-1.5 text-[10px] text-slate-500 truncate">{footer}</footer>}
    </article>
  );
}
