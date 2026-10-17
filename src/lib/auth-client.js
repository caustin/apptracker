// A small client for /api/v1/auth/*. It keeps the names the app used with
// Better Auth (useSession, signIn.email, signUp.email, signOut), and the same
// result shape: { data, error }.
import { useSyncExternalStore } from "react";
import { request, json, setUnauthorizedHandler } from "../api";

// The current session: { data: { user } | null, isPending }.
let state = { data: null, isPending: true };
const listeners = new Set();

function setState(next) {
  state = next;
  listeners.forEach((l) => l());
}

const signedIn = (user) => setState({ data: { user }, isPending: false });
const signedOut = () => setState({ data: null, isPending: false });

async function loadSession() {
  try {
    const { user } = await request("/auth/session");
    signedIn(user);
  } catch {
    signedOut();
  }
}

// Any 401 from the API means the session is gone.
setUnauthorizedHandler(() => {
  if (state.data) signedOut();
});

let loaded = false;
function subscribe(listener) {
  listeners.add(listener);
  if (!loaded) {
    loaded = true;
    loadSession();
  }
  return () => listeners.delete(listener);
}

export function useSession() {
  return useSyncExternalStore(subscribe, () => state);
}

async function authCall(path, body) {
  try {
    const { user } = await request(path, json("POST", body));
    signedIn(user);
    return { data: { user }, error: null };
  } catch (err) {
    return { data: null, error: { message: err.message, status: err.status } };
  }
}

export const signIn = {
  email: ({ email, password }) => authCall("/auth/login", { email, password }),
};

export const signUp = {
  email: ({ email, password, name }) =>
    authCall("/auth/signup", { email, password, name }),
};

export async function signOut() {
  try {
    await request("/auth/logout", { method: "POST" });
  } finally {
    signedOut();
  }
}
