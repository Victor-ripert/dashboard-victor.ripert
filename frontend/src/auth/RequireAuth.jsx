import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "./useAuth";

// Renders its children only for a logged-in user, otherwise sends them to /login
export default function RequireAuth({ children }) {
  const { user, checking } = useAuth();
  const location = useLocation();

  if (checking) {
    return <p className="grid min-h-screen place-items-center text-sm text-slate-400">Loading…</p>;
  }
  if (!user) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }
  return children;
}
