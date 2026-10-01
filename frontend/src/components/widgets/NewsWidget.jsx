import { ArrowUpRight } from "lucide-react";
import { articles } from "../../data/demoData";

export default function NewsWidget() {
  return (
    <div className="flex h-full flex-col justify-between divide-y divide-white/8 overflow-hidden text-xs">
      {articles.slice(0, 3).map((article) => (
        <button
          key={article.title}
          className="group flex w-full items-start gap-2 py-2 text-left transition hover:bg-white/[0.02]"
        >
          <div className="min-w-0 flex-1">
            <p className="text-[10px] font-semibold text-sky-400">{article.source}</p>
            <p className="mt-0.5 line-clamp-1 text-xs leading-snug text-slate-300 group-hover:text-white">
              {article.title}
            </p>
          </div>
          <time className="shrink-0 text-[10px] text-slate-500">{article.time}</time>
          <ArrowUpRight size={13} className="shrink-0 text-slate-600 group-hover:text-sky-400" />
        </button>
      ))}
    </div>
  );
}
