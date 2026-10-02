import { GitCommitHorizontal } from "lucide-react";
import { commits } from "../../data/demoData";

export default function GithubWidget() {
  return (
    <div className="flex h-full flex-col justify-between text-xs">
      <div>
        <div className="flex items-center justify-between pb-1">
          <span className="text-[10px] text-slate-500">128 contributions ce mois</span>
          <span className="text-[10px] text-emerald-400">Actif</span>
        </div>
        <div className="grid grid-cols-12 gap-1 py-1">
          {Array.from({ length: 36 }, (_, i) => (
            <span
              key={i}
              className={`h-2.5 rounded-[2px] ${
                i % 7 === 0 ? "bg-emerald-300" : i % 3 === 0 ? "bg-emerald-500/70" : "bg-emerald-950/70"
              }`}
            />
          ))}
        </div>
      </div>
      <div className="space-y-1 pt-1">
        {commits.slice(0, 2).map((commit) => (
          <div key={commit.hash} className="grid grid-cols-[auto_1fr_auto] items-center gap-1.5 rounded px-1 py-0.5 text-[11px]">
            <GitCommitHorizontal size={13} className="text-emerald-400" />
            <span className="truncate text-slate-300">{commit.message}</span>
            <time className="text-[10px] text-slate-600">{commit.time}</time>
          </div>
        ))}
      </div>
    </div>
  );
}
