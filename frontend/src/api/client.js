// Shared HTTP client: every call to the gateway goes through request().

export class ApiError extends Error {
  constructor(status, message) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

// Filled by the auth module, so this file doesn't depend on how auth works
const config = {
  getToken: () => null,
  onUnauthorized: () => {},
};

export function configureClient(options) {
  Object.assign(config, options);
}

export async function request(path, { method = "GET", body, auth = true } = {}) {
  const headers = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const token = auth ? config.getToken() : null;
  if (token) headers.Authorization = `Bearer ${token}`;

  let response;
  try {
    response = await fetch(path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, "Cannot reach the server");
  }

  const data = await response.json().catch(() => null);
  if (!response.ok) {
    // The token was refused (expired or invalid): let the auth module log out
    if (response.status === 401 && token) config.onUnauthorized();
    throw new ApiError(response.status, data?.error || `Request failed with status ${response.status}`);
  }
  return data;
}
