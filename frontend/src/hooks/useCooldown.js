import { useCallback, useEffect, useState } from "react";

// Countdown in seconds, e.g. to disable a button for a while.
// Returns [secondsLeft, start(seconds)].
export function useCooldown(initialSeconds = 0) {
  const [endsAt, setEndsAt] = useState(() => (initialSeconds > 0 ? Date.now() + initialSeconds * 1000 : 0));
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (endsAt <= Date.now()) return;
    const timer = setInterval(() => {
      const current = Date.now();
      setNow(current);
      if (current >= endsAt) clearInterval(timer);
    }, 250);
    return () => clearInterval(timer);
  }, [endsAt]);

  const start = useCallback((seconds) => {
    const current = Date.now();
    setNow(current);
    setEndsAt(current + seconds * 1000);
  }, []);

  return [Math.max(0, Math.ceil((endsAt - now) / 1000)), start];
}
