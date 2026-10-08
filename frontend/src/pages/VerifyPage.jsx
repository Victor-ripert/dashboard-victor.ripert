import { useState } from "react";
import { Link, Navigate, useLocation, useSearchParams } from "react-router-dom";
import { Loader2 } from "lucide-react";
import AuthCard from "../components/auth/AuthCard";
import { useAuth } from "../auth/useAuth";
import { useCooldown } from "../hooks/useCooldown";

// Must match the gateway's resend cooldown
const RESEND_COOLDOWN_SECONDS = 60;

// Where a new user enters the 6-digit code emailed to them.
// Reached after registering (state.reason "registered") or after logging in
// with an unverified email (state.reason "unverified").
export default function VerifyPage() {
  const [params] = useSearchParams();
  const email = params.get("email") || "";
  const reason = useLocation().state?.reason;
  const { verify, resendCode } = useAuth();

  const [code, setCode] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState(reason === "unverified" ? "Your email isn't verified yet. Enter the code we sent you, or request a new one." : "");
  const [verifying, setVerifying] = useState(false);
  const [resending, setResending] = useState(false);
  // A code was just sent at registration: same wait as after a resend
  const [cooldown, startCooldown] = useCooldown(reason === "registered" ? RESEND_COOLDOWN_SECONDS : 0);

  if (!email) return <Navigate to="/register" replace />;

  async function submit(event) {
    event.preventDefault();
    setVerifying(true);
    setError("");
    setNotice("");
    try {
      // On success the user is logged in, and GuestOnly opens the dashboard
      await verify(email, code);
    } catch (err) {
      setError(err.message);
      setCode("");
      setVerifying(false);
    }
  }

  async function resend() {
    setResending(true);
    setError("");
    setNotice("");
    try {
      await resendCode(email);
      setNotice("A new code was sent. The previous one no longer works.");
      startCooldown(RESEND_COOLDOWN_SECONDS);
    } catch (err) {
      setError(err.message);
      if (err.status === 429) startCooldown(err.data?.retry_after ?? RESEND_COOLDOWN_SECONDS);
    } finally {
      setResending(false);
    }
  }

  return (
    <AuthCard>
      <form onSubmit={submit} className="widget gap-4 p-6">
        <div>
          <h1 className="light-text text-lg font-bold">Verify your email</h1>
          <p className="mt-1 text-xs text-slate-400">
            We sent a 6-digit code to <span className="light-text font-semibold text-slate-200">{email}</span>. It expires in 10 minutes.
          </p>
        </div>

        <label className="block">
          <span className="label">Verification code</span>
          <input className="field mono mt-1 h-12 text-center text-xl tracking-[0.5em]" required
            inputMode="numeric" autoComplete="one-time-code" pattern="\d{6}" maxLength={6} placeholder="000000"
            value={code} onChange={(event) => setCode(event.target.value.replace(/\D/g, ""))} />
        </label>

        {notice && <p className="text-xs text-emerald-400" role="status">{notice}</p>}
        {error && <p className="text-xs text-rose-400" role="alert">{error}</p>}

        <button disabled={verifying || code.length !== 6} className="flex h-10 items-center justify-center gap-2 rounded-md bg-sky-400 text-sm font-bold text-slate-950 hover:bg-sky-300 disabled:opacity-60">
          {verifying && <Loader2 size={16} className="animate-spin" />}
          Verify
        </button>

        <button type="button" onClick={resend} disabled={resending || cooldown > 0}
          className="text-xs font-semibold text-sky-400 hover:underline disabled:cursor-not-allowed disabled:text-slate-500 disabled:no-underline">
          {cooldown > 0 ? `Resend code in ${cooldown}s` : resending ? "Sending…" : "Resend code"}
        </button>

        <p className="text-center text-xs text-slate-400">
          Wrong email or too many attempts? <Link to="/register" className="font-semibold text-sky-400 hover:underline">Start over</Link>
        </p>
      </form>
    </AuthCard>
  );
}
