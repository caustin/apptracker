// Inline form validation. The checks mirror the server's rules in
// internal/api/validate.go so most mistakes are caught before a request; the
// server stays the authority, and its 400 messages ("field: message; …") are
// mapped back onto the matching fields.
import { useId, useRef, useState } from "react";
import { ApiError } from "../api";

// Length caps in characters, as in internal/api/validate.go.
export const LIMITS = { short: 500, url: 2048, text: 50_000 };

// Each check takes the raw input string and returns an error message or null.
// Empty optional values pass; use `required` for mandatory fields.
export const check = {
  required: (v) => (String(v ?? "").trim() ? null : "Required"),

  httpUrl(v) {
    const s = String(v ?? "").trim();
    if (!s) return null;
    try {
      const u = new URL(s);
      return /^https?:$/.test(u.protocol) && u.host
        ? null
        : "Must be an http:// or https:// link";
    } catch {
      return "Enter a full link, like https://example.com";
    }
  },

  email(v) {
    const s = String(v ?? "").trim();
    if (!s) return null;
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(s) ? null : "Enter a valid email";
  },

  // YYYY-MM-DD that is a real calendar date.
  date(v) {
    const s = String(v ?? "").trim();
    if (!s) return null;
    const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
    const d = m && new Date(Date.UTC(+m[1], +m[2] - 1, +m[3]));
    return d && d.toISOString().slice(0, 10) === s
      ? null
      : "Enter a valid date";
  },

  // Money is whole dollars, zero or more.
  wholeDollars(v) {
    if (String(v ?? "").trim() === "") return null;
    const n = Number(v);
    return Number.isInteger(n) && n >= 0 ? null : "Whole dollars, 0 or more";
  },

  // Counts characters (not UTF-16 units), as the server does.
  length(v, min, max) {
    const n = [...String(v ?? "")].length;
    if (n < min) return `At least ${min} characters`;
    if (n > max) return `At most ${max} characters`;
    return null;
  },

  maxLength: (v, max) =>
    [...String(v ?? "").trim()].length > max
      ? `At most ${max} characters`
      : null,
};

// The first failing check, or null: first(check.required(v), check.date(v)).
export const first = (...results) => results.find(Boolean) ?? null;

const capitalize = (s) => s.charAt(0).toUpperCase() + s.slice(1);

// Splits a server 400 message into per-field errors for the given fields and
// a leftover message for the banner.
function splitServerError(err, fields) {
  if (!(err instanceof ApiError) || err.status !== 400) {
    return {
      fieldErrors: {},
      formError: capitalize(err?.message || String(err)),
    };
  }
  const fieldErrors = {};
  const rest = [];
  for (const part of err.message.split("; ")) {
    const m = /^(\w+): (.+)$/.exec(part);
    if (m && fields.includes(m[1])) fieldErrors[m[1]] = capitalize(m[2]);
    else rest.push(capitalize(part));
  }
  return { fieldErrors, formError: rest.length ? rest.join(". ") : null };
}

// useForm keeps string values for a form and validates them on every render.
// A field's error shows once it has been blurred or a submit was attempted,
// and updates as the user types from then on.
//
//   const form = useForm(initial, (v) => ({ company: check.required(v.company) }));
//   <form ref={form.ref} noValidate onSubmit={form.submit(save)}>
//     <input {...form.input("company")} /> <FieldError form={form} name="company" />
export function useForm(initial, validate) {
  const id = useId();
  const ref = useRef(null);
  const [values, setValues] = useState(initial);
  const [touched, setTouched] = useState({});
  const [submitted, setSubmitted] = useState(false);
  const [serverErrors, setServerErrors] = useState({});
  const [formError, setFormError] = useState(null);
  const [busy, setBusy] = useState(false);

  const errors = validate(values);
  const error = (name) =>
    serverErrors[name] ||
    ((submitted || touched[name]) && errors[name]) ||
    null;
  const errorId = (name) => `${id}-${name}-error`;

  const set = (name, value) => {
    setValues((v) => ({ ...v, [name]: value }));
    setServerErrors(({ [name]: _, ...rest }) => rest);
  };

  const input = (name) => {
    const err = error(name);
    return {
      value: values[name],
      onChange: (e) => set(name, e.target.value),
      onBlur: () => setTouched((t) => ({ ...t, [name]: true })),
      "aria-invalid": err ? true : undefined,
      "aria-describedby": err ? errorId(name) : undefined,
    };
  };

  const focusFirstInvalid = () =>
    requestAnimationFrame(() =>
      ref.current?.querySelector('[aria-invalid="true"]')?.focus(),
    );

  // Clears errors and touched state; replaces the values when given.
  const reset = (next) => {
    if (next) setValues(next);
    setTouched({});
    setSubmitted(false);
    setServerErrors({});
    setFormError(null);
  };

  // Shows a server-side problem on one field (e.g. an email already in use).
  const setFieldError = (name, message) => {
    setServerErrors((s) => ({ ...s, [name]: message }));
    focusFirstInvalid();
  };

  // Wraps a save function: validates first, then runs it, mapping a thrown
  // error onto fields or the banner.
  const submit = (fn) => async (e) => {
    e?.preventDefault();
    setSubmitted(true);
    setFormError(null);
    if (Object.values(errors).some(Boolean)) {
      focusFirstInvalid();
      return;
    }
    setBusy(true);
    try {
      await fn(values);
    } catch (err) {
      const split = splitServerError(err, Object.keys(values));
      setServerErrors(split.fieldErrors);
      setFormError(split.formError);
      focusFirstInvalid();
    } finally {
      setBusy(false);
    }
  };

  return {
    ref,
    values,
    set,
    input,
    error,
    errorId,
    submit,
    reset,
    setFieldError,
    busy,
    formError,
  };
}

export function FieldError({ form, name }) {
  const message = form.error(name);
  if (!message) return null;
  return (
    <span id={form.errorId(name)} className="field-error">
      {message}
    </span>
  );
}

export function FormError({ form }) {
  if (!form.formError) return null;
  return (
    <div className="error-banner" role="alert">
      {form.formError}
    </div>
  );
}
