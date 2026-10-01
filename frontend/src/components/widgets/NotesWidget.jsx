import { useState } from "react";

export default function NotesWidget() {
  const [text, setText] = useState("Connecter trading à /api/trading/orders\nAjouter endpoint météo au gateway");
  return (
    <div className="flex h-full flex-col justify-between text-xs">
      <textarea
        value={text}
        onChange={(event) => setText(event.target.value)}
        className="h-24 w-full flex-1 resize-none rounded-md border border-white/8 bg-white/[0.025] p-2 text-xs leading-5 text-slate-300 outline-none focus:border-sky-400/50"
        aria-label="Notes du projet"
      />
      <p className="mt-1 text-[10px] text-slate-500">Sauvegardé localement.</p>
    </div>
  );
}
