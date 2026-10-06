import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "./useAuth";

// Renders its children only for a visitor who is not logged in (login and register pages)
export default function GuestOnly({ children }) {
  const { user, checking } = useAuth();
  const location = useLocation();

  if (checking) {
    return <p className="grid min-h-screen place-items-center text-sm text-slate-400">Loading…</p>;
  }
  if (user) {
    return <Navigate to={location.state?.from?.pathname || "/"} replace />;
  }
  return children;
}
