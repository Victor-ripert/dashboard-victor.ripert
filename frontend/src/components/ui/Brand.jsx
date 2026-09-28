export default function Brand() {
  return (
    <div className="flex items-center gap-2" aria-label="Nexus Dashboard">
      <svg viewBox="0 0 32 32" className="size-7 text-sky-400" fill="none" aria-hidden="true">
        <rect x="4" y="4" width="9" height="9" rx="2" fill="currentColor" />
        <rect x="19" y="4" width="9" height="9" rx="2" fill="currentColor" opacity=".45" />
        <rect x="4" y="19" width="9" height="9" rx="2" fill="currentColor" opacity=".45" />
        <path d="M19 19h9v9h-9z" stroke="currentColor" strokeWidth="2" />
      </svg>
      <span className="text-sm font-bold tracking-wide">Dashboard</span>
    </div>
  );
}
