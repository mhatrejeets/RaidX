const TOKEN_KEY = "raidx.token";
const SESSION_KEY = "raidx.session";
const AUTH_ROUTES = {
  refresh: "/refresh",
  logout: "/logout",
  logoutAll: "/logout-all"
};

export function decodeJwtPayload(token) {
  if (!token) return null;
  if (typeof atob !== "function") return null;
  try {
    const parts = token.split(".");
    if (parts.length < 2) return null;
    const normalized = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const payload = JSON.parse(atob(normalized));
    return payload;
  } catch {
    return null;
  }
}

export function createAuthClient({ baseUrl = "", storage }) {
  if (!storage || !storage.getItem || !storage.setItem || !storage.removeItem) {
    throw new Error("storage adapter with getItem/setItem/removeItem is required");
  }

  async function getToken() {
    return storage.getItem(TOKEN_KEY);
  }

  async function setToken(token) {
    if (!token) {
      await storage.removeItem(TOKEN_KEY);
      return;
    }
    await storage.setItem(TOKEN_KEY, token);
  }

  async function clearToken() {
    await storage.removeItem(TOKEN_KEY);
    await storage.removeItem(SESSION_KEY);
  }

  async function setSessionMeta(meta) {
    await storage.setItem(SESSION_KEY, JSON.stringify(meta));
  }

  async function getSessionMeta() {
    const raw = await storage.getItem(SESSION_KEY);
    if (!raw) return null;
    try {
      return JSON.parse(raw);
    } catch {
      return null;
    }
  }

  async function login({ identifier, password }) {
    const response = await fetch(`${baseUrl}/login`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ identifier, password })
    });

    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(data.error || "Login failed");
    }

    if (data.token) {
      await setToken(data.token);
    }
    await setSessionMeta({
      userId: data.user_id || null,
      role: data.role || null,
      exp: data.expires || null
    });

    return data;
  }

  async function signup(payload) {
    const body = new URLSearchParams();
    Object.entries(payload || {}).forEach(([key, value]) => {
      if (value === undefined || value === null) return;
      body.append(key, String(value));
    });

    const response = await fetch(`${baseUrl}/signup`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body
    });

    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(data.error || "Signup failed");
    }

    return data;
  }

  async function refresh() {
    const response = await fetch(`${baseUrl}${AUTH_ROUTES.refresh}`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" }
    });

    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      await clearToken();
      throw new Error(data.error || "Token refresh failed");
    }

    if (data.token) {
      await setToken(data.token);
    }
    const existing = (await getSessionMeta()) || {};
    await setSessionMeta({
      userId: existing.userId || null,
      role: existing.role || null,
      exp: data.expires || existing.exp || null
    });

    return data;
  }

  async function logout() {
    const token = await getToken();
    await fetch(`${baseUrl}${AUTH_ROUTES.logout}`, {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {})
      }
    }).catch(() => null);
    await clearToken();
  }

  async function logoutAll() {
    const token = await getToken();
    const response = await fetch(`${baseUrl}${AUTH_ROUTES.logoutAll}`, {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {})
      }
    });

    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(data.error || "Logout-all failed");
    }

    await clearToken();
    return data;
  }

  async function apiFetch(path, options = {}) {
    const token = await getToken();
    const response = await fetch(`${baseUrl}${path}`, {
      ...options,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...(options.headers || {})
      }
    });

    if (response.status === 401) {
      await refresh();
      const nextToken = await getToken();
      return fetch(`${baseUrl}${path}`, {
        ...options,
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          ...(nextToken ? { Authorization: `Bearer ${nextToken}` } : {}),
          ...(options.headers || {})
        }
      });
    }

    return response;
  }

  async function getSessionSummary() {
    const token = await getToken();
    const sessionMeta = await getSessionMeta();
    const payload = decodeJwtPayload(token);
    return {
      token,
      userId: sessionMeta?.userId || payload?.user_id || null,
      role: sessionMeta?.role || payload?.role || null,
      exp: sessionMeta?.exp || payload?.exp || null
    };
  }

  return {
    getToken,
    setToken,
    clearToken,
    signup,
    login,
    refresh,
    logout,
    logoutAll,
    apiFetch,
    getSessionSummary
  };
}