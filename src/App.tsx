import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "./api";
import type { Db } from "./types";
import { useSession, signOut } from "./lib/auth-client";
import { AuthScreen } from "./components/Auth";
import { ErrorBoundary } from "./components/ErrorBoundary";
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
  const { data: session, isPending } = useSession();
  const [db, setDb] = useState<Db | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTabState] = useState<Tab>(() => tabFromHash(location.hash));
  const sessionRef = useRef(session);
  sessionRef.current = session;
  const setTab = (t: Tab) => {
    setTabState(t);
    if (location.hash.replace(/^#/, "") !== t) location.hash = t;
  };

  useEffect(() => {
    const onHashChange = () => setTabState(tabFromHash(location.hash));
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  const refresh = useCallback(() => {
    if (!session) return;
    const current = session;
    setError(null);
    api
      .loadAll()
      .then((next) => {
        if (sessionRef.current !== current) return; // session changed mid-flight
        setDb(next);
        setError(null);
      })
      .catch((e) => {
        if (sessionRef.current !== current) return;
        setDb(null); // don't keep showing stale/partial data behind the banner
        setError(String(e));
      });
  }, [session]);

  // (Re)load whenever the signed-in user changes; clear data on sign-out.
  useEffect(() => {
    if (session) refresh();
    else setDb(null);
  }, [session, refresh]);

  if (isPending) return <div className="loading">Loading…</div>;
  if (!session) return <AuthScreen />;

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
          <div className="topbar-actions">
            <a className="btn" href="/api/export">
              Export
            </a>
            <button className="btn" onClick={() => signOut()}>
              Sign out
            </button>
          </div>
        </div>
      </header>

      <main className="content">
        {error && (
          <div className="error-banner">
            Could not reach the API server — {error}
          </div>
        )}
        {!db && !error && <div className="loading">Loading…</div>}
        {db && (
          <ErrorBoundary key={tab}>
            {tab === "Dashboard" && <Dashboard db={db} onNavigate={setTab} />}
            {tab === "Positions" && <Positions db={db} onChange={refresh} />}
            {tab === "People" && <People db={db} onChange={refresh} />}
            {tab === "Resumes" && <Resumes db={db} onChange={refresh} />}
            {tab === "Goals" && <GoalsPage db={db} onChange={refresh} />}
          </ErrorBoundary>
        )}
      </main>
    </div>
  );
}
