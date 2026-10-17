import { useState } from "react";
import { ApiError } from "../api";
import { signIn, signUp } from "../lib/auth-client";
import { FieldError, FormError, check, first, useForm } from "../lib/form";

// Sign-up enforces the server's rules; sign-in only needs both fields filled
// (a wrong password gets the server's one generic message).
const validateAuth = (mode) => (f) =>
  mode === "signup"
    ? {
        name: check.maxLength(f.name, 200),
        email: first(check.required(f.email), check.email(f.email)),
        password: first(
          check.required(f.password),
          check.length(f.password, 12, 128),
        ),
      }
    : {
        email: first(check.required(f.email), check.email(f.email)),
        password: check.required(f.password),
      };

export function AuthScreen() {
  const [mode, setMode] = useState("signin");
  const form = useForm(
    { name: "", email: "", password: "" },
    validateAuth(mode),
  );

  const submit = async ({ name, email, password }) => {
    const res =
      mode === "signup"
        ? await signUp.email({ email, password, name: name || email })
        : await signIn.email({ email, password });
    if (!res.error) return;
    if (res.error.status === 409) {
      form.setFieldError("email", "An account with this email already exists");
      return;
    }
    // The form maps "field: message" 400s onto fields; anything else (such
    // as "invalid email or password") shows in the banner.
    throw new ApiError(
      res.error.status,
      res.error.message || "Authentication failed",
    );
  };

  const switchMode = () => {
    setMode(mode === "signup" ? "signin" : "signup");
    form.reset();
  };

  return (
    <div className="auth-screen">
      <div className="card form-card auth-card">
        <div className="wordmark">
          <span className="wordmark-dot" />
          AppTracker
        </div>
        <h2>{mode === "signup" ? "Create your account" : "Welcome back"}</h2>
        <FormError form={form} />
        <form ref={form.ref} noValidate onSubmit={form.submit(submit)}>
          {mode === "signup" && (
            <label className="field" htmlFor="auth-name">
              Name
              <input
                {...form.input("name")}
                id="auth-name"
                name="name"
                data-testid="auth-name"
                placeholder="Ada Lovelace"
                autoComplete="name"
              />
              <FieldError form={form} name="name" />
            </label>
          )}
          <label className="field" htmlFor="auth-email">
            Email
            <input
              {...form.input("email")}
              id="auth-email"
              name="email"
              data-testid="auth-email"
              type="email"
              autoComplete="email"
            />
            <FieldError form={form} name="email" />
          </label>
          <label className="field" htmlFor="auth-password">
            Password
            <input
              {...form.input("password")}
              id="auth-password"
              name="password"
              data-testid="auth-password"
              type="password"
              autoComplete={
                mode === "signup" ? "new-password" : "current-password"
              }
            />
            <FieldError form={form} name="password" />
            {mode === "signup" && !form.error("password") && (
              <span className="field-hint">12 to 128 characters</span>
            )}
          </label>
          <div className="form-actions">
            <button
              className="btn btn-primary"
              type="submit"
              disabled={form.busy}
              data-testid="auth-submit"
            >
              {form.busy ? "…" : mode === "signup" ? "Sign up" : "Sign in"}
            </button>
          </div>
        </form>
        <p className="muted">
          {mode === "signup" ? "Already have an account?" : "No account yet?"}{" "}
          <button type="button" className="link-btn" onClick={switchMode}>
            {mode === "signup" ? "Sign in" : "Sign up"}
          </button>
        </p>
      </div>
    </div>
  );
}
