import { request } from "./client";

export const authApi = {
  // Creates an unverified account and emails it a code. No token: see verify.
  register: (email, password) =>
    request("/api/auth/register", { method: "POST", body: { email, password }, auth: false }),
  // Returns { token, user } when the code is right
  verify: (email, code) =>
    request("/api/auth/verify", { method: "POST", body: { email, code }, auth: false }),
  resendCode: (email) =>
    request("/api/auth/resend-code", { method: "POST", body: { email }, auth: false }),
  login: (email, password) =>
    request("/api/auth/login", { method: "POST", body: { email, password }, auth: false }),
  me: () => request("/api/auth/me"),
};
