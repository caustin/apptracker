export const API = "/api/v1";

// Called when any request comes back 401 (the session expired or was ended
// elsewhere). App registers a handler that drops the session and shows /login.
let onUnauthorized = () => {};
export const setUnauthorizedHandler = (fn) => {
  onUnauthorized = fn;
};

export class ApiError extends Error {
  constructor(status, message) {
    super(message);
    this.status = status;
  }
}

// Fetches JSON; non-2xx responses throw an ApiError carrying the server's
// {"error": "…"} message.
export async function request(path, init) {
  const res = await fetch(API + path, { credentials: "same-origin", ...init });
  if (!res.ok) {
    let message = `${res.status} ${res.statusText}`;
    try {
      const body = await res.json();
      if (body && body.error) message = body.error;
    } catch {
      // not JSON; keep the status line
    }
    if (res.status === 401) onUnauthorized();
    throw new ApiError(res.status, message);
  }
  return res.json();
}

export const json = (method, body) => ({
  method,
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify(body),
});

export const api = {
  loadAll: async () => {
    const [positions, people, events, resumes, goals] = await Promise.all([
      request("/positions"),
      request("/people"),
      request("/events"),
      request("/resumes"),
      request("/goals"),
    ]);
    return { positions, people, events, resumes, goals };
  },
  uploadResume: (file, name) => {
    const fd = new FormData();
    fd.append("file", file);
    if (name) fd.append("name", name);
    return request("/resumes", { method: "POST", body: fd });
  },
  create: (resource, body) => request(`/${resource}`, json("POST", body)),
  // PUT is a full replace: send every field.
  update: (resource, id, body) =>
    request(`/${resource}/${id}`, json("PUT", body)),
  remove: (resource, id) => request(`/${resource}/${id}`, { method: "DELETE" }),
  saveGoals: (g) => request("/goals", json("PUT", g)),
};
