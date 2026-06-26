import { useState } from "react";
import { signIn, signUp } from "../lib/auth-client";

export function AuthScreen() {
  const [mode, setMode] = useState<"signin" | "signup">("signin");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      const res =
        mode === "signup"
          ? await signUp.email({ email, password, name: name || email })
          : await signIn.email({ email, password });
      if (res.error) setError(res.error.message || "Authentication failed");
    } catch (err) {
      setError(String(err));
    } finally {
      setBusy(false);
    }
  };

  const google = async () => {
    setError(null);
    setBusy(true);
    try {
      const res = await signIn.social({ provider: "google", callbackURL: "/" });
      if (res?.error) setError(res.error.message || "Google sign-in failed");
    } catch (err) {
      setError(String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="auth-screen">
      <div className="card form-card auth-card">
        <div className="wordmark">
          <span className="wordmark-dot" />
          AppTracker
        </div>
        <h2>{mode === "signup" ? "Create your account" : "Welcome back"}</h2>
        {error && <div className="error-banner">{error}</div>}
        <form onSubmit={submit}>
          {mode === "signup" && (
            <label className="field" htmlFor="auth-name">
              Name
              <input
                id="auth-name"
                name="name"
                data-testid="auth-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Ada Lovelace"
                autoComplete="name"
              />
            </label>
          )}
          <label className="field" htmlFor="auth-email">
            Email
            <input
              id="auth-email"
              name="email"
              data-testid="auth-email"
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
            />
          </label>
          <label className="field" htmlFor="auth-password">
            Password
            <input
              id="auth-password"
              name="password"
              data-testid="auth-password"
              type="password"
              required
              minLength={mode === "signup" ? 12 : undefined}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete={
                mode === "signup" ? "new-password" : "current-password"
              }
            />
          </label>
          <div className="form-actions">
            <button
              className="btn btn-primary"
              type="submit"
              disabled={busy}
              data-testid="auth-submit"
            >
              {busy
                ? "…"
                : mode === "signup"
                  ? "Sign up"
                  : "Sign in"}
            </button>
          </div>
        </form>
        <button className="btn" onClick={google} disabled={busy}>
          Continue with Google
        </button>
        <p className="muted">
          {mode === "signup" ? "Already have an account?" : "No account yet?"}{" "}
          <button
            className="link-btn"
            onClick={() => {
              setMode(mode === "signup" ? "signin" : "signup");
              setError(null);
            }}
          >
            {mode === "signup" ? "Sign in" : "Sign up"}
          </button>
        </p>
      </div>
    </div>
  );
}
