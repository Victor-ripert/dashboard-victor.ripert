import { useState } from "react";
import { Loader2 } from "lucide-react";
import AuthCard from "./AuthCard";

// Email + password form shared by the login and register pages
export default function AuthForm({ title, submitLabel, passwordAutoComplete, passwordHint, minPasswordLength, onSubmit, footer }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(event) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      await onSubmit(email, password);
    } catch (err) {
      setError(err.message);
      setLoading(false);
    }
  }

  return (
    <AuthCard>
      <form onSubmit={submit} className="widget gap-4 p-6">
        <h1 className="light-text text-lg font-bold">{title}</h1>

        <label className="block">
          <span className="label">Email</span>
          <input className="field mt-1 h-10 text-sm" type="email" autoComplete="email" required
            value={email} onChange={(event) => setEmail(event.target.value)} />
        </label>

        <label className="block">
          <span className="label">Password</span>
          <input className="field mt-1 h-10 text-sm" type="password" autoComplete={passwordAutoComplete} required
            minLength={minPasswordLength} maxLength={72}
            value={password} onChange={(event) => setPassword(event.target.value)} />
          {passwordHint && <span className="mt-1 block text-[11px] text-slate-500">{passwordHint}</span>}
        </label>

        {error && <p className="text-xs text-rose-400" role="alert">{error}</p>}

        <button disabled={loading} className="flex h-10 items-center justify-center gap-2 rounded-md bg-sky-400 text-sm font-bold text-slate-950 hover:bg-sky-300 disabled:opacity-60">
          {loading && <Loader2 size={16} className="animate-spin" />}
          {submitLabel}
        </button>

        <p className="text-center text-xs text-slate-400">{footer}</p>
      </form>
    </AuthCard>
  );
}
