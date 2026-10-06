const KEY = "dashboard.token";

// Where the login token is kept between page loads.
// Replace this object to store it somewhere else (memory, sessionStorage...).
export const tokenStorage = {
  get() {
    try {
      return localStorage.getItem(KEY);
    } catch {
      return null;
    }
  },
  set(token) {
    try {
      localStorage.setItem(KEY, token);
    } catch {
      // Storage blocked (private mode...): the session lasts until reload
    }
  },
  clear() {
    try {
      localStorage.removeItem(KEY);
    } catch {
      // Nothing to clear
    }
  },
};
