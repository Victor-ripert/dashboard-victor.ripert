import { useCallback, useEffect, useMemo, useState } from "react";
import { authApi } from "../api/auth";
import { configureClient } from "../api/client";
import { AuthContext } from "./authContext";
import { tokenStorage } from "./tokenStorage";

export default function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  // True while a token saved by a previous visit is being checked
  const [checking, setChecking] = useState(() => tokenStorage.get() !== null);

  const logout = useCallback(() => {
    tokenStorage.clear();
    setUser(null);
  }, []);

  useEffect(() => {
    configureClient({ getToken: tokenStorage.get, onUnauthorized: logout });

    if (!tokenStorage.get()) return;
    authApi
      .me()
      .then(setUser)
      .catch(() => setUser(null))
      .finally(() => setChecking(false));
  }, [logout]);

  const startSession = useCallback(({ token, user }) => {
    tokenStorage.set(token);
    setUser(user);
  }, []);

  const login = useCallback(
    (email, password) => authApi.login(email, password).then(startSession),
    [startSession],
  );
  // Doesn't log in: the account must first be verified with the emailed code
  const register = useCallback((email, password) => authApi.register(email, password), []);
  const verify = useCallback(
    (email, code) => authApi.verify(email, code).then(startSession),
    [startSession],
  );
  const resendCode = useCallback((email) => authApi.resendCode(email), []);

  const value = useMemo(
    () => ({ user, checking, login, register, verify, resendCode, logout }),
    [user, checking, login, register, verify, resendCode, logout],
  );
  return <AuthContext value={value}>{children}</AuthContext>;
}
