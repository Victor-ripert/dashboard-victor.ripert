import { useContext } from "react";
import { AuthContext } from "./authContext";

// Returns { user, checking, login, register, verify, resendCode, logout }
export function useAuth() {
  const auth = useContext(AuthContext);
  if (!auth) throw new Error("useAuth must be used inside <AuthProvider>");
  return auth;
}
