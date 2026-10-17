// For buttons and selects that call the API outside a form. run(fn) wraps an
// async handler so a failure is kept in `error` (render it in a banner)
// instead of becoming an unhandled rejection the user never sees.
import { useState } from "react";

export function useAction() {
  const [error, setError] = useState(null);
  const run =
    (fn) =>
    async (...args) => {
      setError(null);
      try {
        await fn(...args);
      } catch (err) {
        setError(err?.message || String(err));
      }
    };
  return { error, run };
}
