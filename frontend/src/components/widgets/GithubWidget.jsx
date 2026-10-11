import { ExternalLink, GitCommitHorizontal, Loader2, Search } from "lucide-react";
import { useEffect, useState } from "react";
import { githubApi } from "../../api/services";

const repositoryPattern = /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;

function formatCommitDate(date) {
  return new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric" }).format(new Date(date));
}

export default function GithubWidget({ defaultRepository = "" }) {
  const [query, setQuery] = useState(defaultRepository);
  const [repository, setRepository] = useState(defaultRepository);
  const [activity, setActivity] = useState(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(Boolean(defaultRepository));
  const [connecting, setConnecting] = useState(false);
  const [connected, setConnected] = useState(() => new URLSearchParams(window.location.search).get("github") === "connected");

  useEffect(() => {
    if (!repository) return undefined;

    let ignore = false;
    githubApi
      .getActivity(repository)
      .then((data) => {
        if (!ignore) setActivity(data);
      })
      .catch((err) => {
        if (!ignore) {
          setActivity(null);
          setError(err.message || "Unable to load GitHub activity");
        }
      })
      .finally(() => {
        if (!ignore) setLoading(false);
      });

    return () => { ignore = true; };
  }, [repository]);

  function submit(event) {
    event.preventDefault();
    const next = query.trim();
    if (!next) {
      setError("Enter a repository first");
      return;
    }
    if (!repositoryPattern.test(next)) {
      setError("Use the owner/repository format");
      return;
    }
    if (next === repository) return;
    setError("");
    setActivity(null);
    setLoading(true);
    setRepository(next);
  }

  async function connectGithub() {
    setConnecting(true);
    setError("");
    try {
      const { url } = await githubApi.connect();
      window.location.assign(url);
    } catch (err) {
      setError(err.message || "Unable to connect GitHub");
      setConnecting(false);
    }
  }

  async function disconnectGithub() {
    setConnecting(true);
    setError("");
    try {
      await githubApi.disconnect();
      setConnected(false);
    } catch (err) {
      setError(err.message || "Unable to disconnect GitHub");
    } finally {
      setConnecting(false);
    }
  }

  const commits = activity?.commits || [];
  const counts = activity?.counts || { today: 0, this_week: 0 };

  return (
    <div className="flex h-full flex-col text-xs">
      <form onSubmit={submit} className="flex gap-1.5">
        <label className="relative flex-1">
          <span className="sr-only">GitHub repository</span>
          <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-500" />
          <input
            className="field pl-8"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="owner/repository"
            autoComplete="off"
          />
        </label>
        <button disabled={loading} className="grid h-8 min-w-10 place-items-center rounded-md bg-sky-400 px-3 text-xs font-bold text-slate-950 hover:bg-sky-300 disabled:opacity-60">
          {loading ? <Loader2 size={13} className="animate-spin" /> : "Load"}
        </button>
      </form>

      <div className="mt-2 flex items-center justify-between gap-2">
        <span className="truncate text-[10px] text-slate-500">{connected ? "GitHub connected" : "Public repositories"}</span>
        <button type="button" onClick={connected ? disconnectGithub : connectGithub} disabled={connecting} className="text-[10px] font-semibold text-sky-300 hover:text-sky-200 disabled:opacity-60">
          {connecting ? "Working..." : connected ? "Disconnect" : "Connect GitHub"}
        </button>
      </div>

      {error && <p className="mt-2 text-[11px] text-amber-300" role="alert">{error}</p>}

      {loading && <p className="mt-4 flex items-center gap-2 text-slate-500"><Loader2 size={13} className="animate-spin" />Loading activity...</p>}

      {!loading && activity && (
        <>
          <div className="mt-3 grid grid-cols-2 gap-2 border-b border-white/8 pb-2">
            <div>
              <p className="label">Today</p>
              <p className="mono text-sm font-semibold light-text">{counts.today}</p>
            </div>
            <div className="text-right">
              <p className="label">This week</p>
              <p className="mono text-sm font-semibold light-text">{counts.this_week}</p>
            </div>
          </div>

          {commits.length ? (
            <div className="mt-2 space-y-1 overflow-auto">
              {commits.map((commit) => (
                <a key={commit.sha} href={commit.url} target="_blank" rel="noreferrer" className="grid grid-cols-[auto_1fr_auto] items-center gap-1.5 rounded px-1 py-1 hover:bg-white/[0.04]">
                  <GitCommitHorizontal size={13} className="text-emerald-400" />
                  <span className="min-w-0">
                    <span className="block truncate text-[11px] text-slate-300">{commit.message.split("\n")[0]}</span>
                    <span className="block truncate text-[10px] text-slate-500">{commit.author}</span>
                  </span>
                  <span className="flex items-center gap-1 text-[10px] text-slate-600">
                    <time dateTime={commit.date}>{formatCommitDate(commit.date)}</time>
                    <ExternalLink size={11} />
                  </span>
                </a>
              ))}
            </div>
          ) : (
            <p className="mt-4 text-slate-500">No commits found for this repository.</p>
          )}
        </>
      )}

      {!loading && !activity && !error && (
        <p className="mt-4 text-slate-500">Enter a repository to see recent activity.</p>
      )}
    </div>
  );
}
