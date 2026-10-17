import { useCallback, useEffect, useRef, useState } from "react";
import { Navigate, NavLink, Route, Routes, useLocation } from "react-router";
import { api, API } from "./api";
import { useSession, signOut } from "./lib/auth-client";
import { AuthScreen } from "./components/Auth";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { Dashboard } from "./components/Dashboard";
import { Positions, PositionRoute } from "./components/Positions";
import { People } from "./components/People";
import { Resumes } from "./components/Resumes";
import { GoalsPage } from "./components/Goals";

const TABS = [
  { to: "/", label: "Dashboard" },
  { to: "/positions", label: "Positions" },
  { to: "/people", label: "People" },
  { to: "/resumes", label: "Resumes" },
  { to: "/goals", label: "Goals" },
];

export function App() {
  const { data: session, isPending } = useSession();
  const location = useLocation();

  if (isPending) return <div className="loading">Loading…</div>;

  if (!session) {
    if (location.pathname !== "/login") {
      return <Navigate to="/login" replace state={{ from: location }} />;
    }
    return <AuthScreen />;
  }
  if (location.pathname === "/login") {
    const from = location.state?.from;
    return <Navigate to={from ? from.pathname + from.search : "/"} replace />;
  }
  return <SignedIn session={session} />;
}

function SignedIn({ session }) {
  const location = useLocation();
  const [db, setDb] = useState(null);
  const [error, setError] = useState(null);
  const sessionRef = useRef(session);
  sessionRef.current = session;
  const refreshSeqRef = useRef(0);

  const refresh = useCallback(() => {
    const current = session;
    const seq = ++refreshSeqRef.current;
    setError(null);
    api
      .loadAll()
      .then((next) => {
        if (sessionRef.current !== current) return; // session changed mid-flight
        if (refreshSeqRef.current !== seq) return; // superseded by a newer refresh
        setDb(next);
        setError(null);
      })
      .catch((e) => {
        if (sessionRef.current !== current) return;
        if (refreshSeqRef.current !== seq) return;
        setDb(null); // don't keep showing stale/partial data behind the banner
        setError(String(e));
      });
  }, [session]);

  // (Re)load whenever the signed-in user changes.
  useEffect(() => {
    refresh();
  }, [refresh]);

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
              <NavLink
                key={t.to}
                to={t.to}
                end={t.to === "/"}
                className={({ isActive }) =>
                  `nav-tab ${isActive ? "active" : ""}`
                }
              >
                {t.label}
              </NavLink>
            ))}
          </nav>
          <div className="topbar-actions">
            <a className="btn" href={`${API}/export`}>
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
          <ErrorBoundary key={location.pathname}>
            <Routes>
              <Route path="/" element={<Dashboard db={db} />} />
              <Route
                path="/positions"
                element={<Positions db={db} onChange={refresh} />}
              />
              <Route
                path="/positions/:id"
                element={<PositionRoute db={db} onChange={refresh} />}
              />
              <Route
                path="/people"
                element={<People db={db} onChange={refresh} />}
              />
              <Route
                path="/resumes"
                element={<Resumes db={db} onChange={refresh} />}
              />
              <Route
                path="/goals"
                element={<GoalsPage db={db} onChange={refresh} />}
              />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </ErrorBoundary>
        )}
      </main>
    </div>
  );
}
