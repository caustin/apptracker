import { useCallback, useEffect, useState } from "react";
import { api } from "./api";
import type { Db } from "./types";
import { Dashboard } from "./components/Dashboard";
import { Positions } from "./components/Positions";
import { People } from "./components/People";
import { Resumes } from "./components/Resumes";
import { GoalsPage } from "./components/Goals";

const TABS = ["Dashboard", "Positions", "People", "Resumes", "Goals"] as const;
type Tab = (typeof TABS)[number];
const TAB_SET = new Set<string>(TABS as readonly string[]);

function tabFromHash(hash: string): Tab {
  try {
    const raw = decodeURIComponent(hash.replace(/^#/, ""));
    return TAB_SET.has(raw) ? (raw as Tab) : "Dashboard";
  } catch {
    return "Dashboard";
  }
}

export function App() {
  const [db, setDb] = useState<Db | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTabState] = useState<Tab>(() => tabFromHash(location.hash));
  const setTab = (t: Tab) => {
    setTabState(t);
    location.hash = t;
  };

  useEffect(() => {
    const onHashChange = () => setTabState(tabFromHash(location.hash));
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  const refresh = useCallback(() => {
    api
      .loadAll()
      .then(setDb)
      .catch((e) => setError(String(e)));
  }, []);

  useEffect(refresh, [refresh]);

  return (
    <div className="app">
      <header className="topbar">
        <div className="topbar-inner">
          <div className="wordmark">
            <span className="wordmark-dot" />
            AppTracker
            <span className="wordmark-sub">job search ledger</span>
          </div>
          <nav className="nav">
            {TABS.map((t) => (
              <button
                key={t}
                className={`nav-tab ${tab === t ? "active" : ""}`}
                onClick={() => setTab(t)}
              >
                {t}
              </button>
            ))}
          </nav>
        </div>
      </header>

      <main className="content">
        {error && (
          <div className="error-banner">
            Could not reach the API server — {error}
          </div>
        )}
        {!db && !error && <div className="loading">Loading…</div>}
        {db && tab === "Dashboard" && <Dashboard db={db} onNavigate={setTab} />}
        {db && tab === "Positions" && <Positions db={db} onChange={refresh} />}
        {db && tab === "People" && <People db={db} onChange={refresh} />}
        {db && tab === "Resumes" && <Resumes db={db} onChange={refresh} />}
        {db && tab === "Goals" && <GoalsPage db={db} onChange={refresh} />}
      </main>
    </div>
  );
}
