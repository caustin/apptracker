import { useRef, useState } from "react";
import { api } from "../api";
import type { Db, Resume } from "../types";
import { fmtDate, resumeFileUrl } from "../types";

export function Resumes({ db, onChange }: { db: Db; onChange: () => void }) {
  const fileRef = useRef<HTMLInputElement>(null);
  const [pending, setPending] = useState<{ file: File; label: string } | null>(
    null,
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const choose = (file: File) => {
    setError(null);
    setPending({ file, label: file.name.replace(/\.pdf$/i, "") });
  };

  const upload = async () => {
    if (!pending) return;
    setBusy(true);
    setError(null);
    try {
      await api.uploadResume(pending.file, pending.label.trim());
      setPending(null);
      onChange();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };

  const del = async (id: number, name: string, usedBy: number) => {
    const warning =
      usedBy > 0
        ? `Delete "${name}"? It is attached to ${usedBy} position${usedBy === 1 ? "" : "s"}, which will be unlinked.`
        : `Delete "${name}"?`;
    if (!confirm(warning)) return;
    await api.remove("resumes", id);
    onChange();
  };

  return (
    <div>
      <div className="page-head">
        <h1>Resumes</h1>
        <button
          className="btn btn-primary"
          disabled={busy || !!pending}
          onClick={() => fileRef.current?.click()}
        >
          + Upload PDF
        </button>
        <input
          ref={fileRef}
          type="file"
          accept="application/pdf,.pdf"
          hidden
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) choose(f);
            e.target.value = "";
          }}
        />
      </div>

      {error && <div className="error-banner">{error}</div>}

      {pending && (
        <div className="card form-card">
          <h2>Label this resume</h2>
          <p className="muted small upload-filename">{pending.file.name}</p>
          <label className="field">
            Label
            <input
              value={pending.label}
              autoFocus
              onChange={(e) =>
                setPending({ ...pending, label: e.target.value })
              }
              onKeyDown={(e) => {
                if (e.key === "Enter" && pending.label.trim()) upload();
              }}
              placeholder="e.g. Platform-focused v3"
            />
          </label>
          <div className="form-actions">
            <button className="btn" onClick={() => setPending(null)}>
              Cancel
            </button>
            <button
              className="btn btn-primary"
              disabled={busy || !pending.label.trim()}
              onClick={upload}
            >
              {busy ? "Uploading…" : "Upload"}
            </button>
          </div>
        </div>
      )}

      {db.resumes.length === 0 && !pending && (
        <div className="empty-state big">
          No resumes yet. Upload the PDF versions you send out, give each a
          label, then attach one to each position.
        </div>
      )}

      <div className="position-list">
        {db.resumes.map((r) => {
          const usedBy = db.positions.filter((p) => p.resumeId === r.id).length;
          return (
            <ResumeRow
              key={r.id}
              resume={r}
              usedBy={usedBy}
              onDelete={() => del(r.id, r.name, usedBy)}
              onChange={onChange}
            />
          );
        })}
      </div>
    </div>
  );
}

function ResumeRow({
  resume,
  usedBy,
  onDelete,
  onChange,
}: {
  resume: Resume;
  usedBy: number;
  onDelete: () => void;
  onChange: () => void;
}) {
  const [name, setName] = useState<string | null>(null); // non-null while renaming

  const rename = async () => {
    if (name && name.trim() && name.trim() !== resume.name) {
      await api.update("resumes", resume.id, { name: name.trim() });
      onChange();
    }
    setName(null);
  };

  return (
    <div className="resume-row">
      <div className="position-main">
        <span className="resume-icon" aria-hidden>
          PDF
        </span>
        {name === null ? (
          <span className="position-company">{resume.name}</span>
        ) : (
          <input
            className="rename-input"
            value={name}
            autoFocus
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") rename();
              if (e.key === "Escape") setName(null);
            }}
            onBlur={rename}
          />
        )}
      </div>
      <div className="position-meta">
        <span className="muted">uploaded {fmtDate(resume.createdAt)}</span>
        <span className="muted">
          {usedBy === 0
            ? "not attached"
            : `used by ${usedBy} position${usedBy === 1 ? "" : "s"}`}
        </span>
        <button className="link-btn" onClick={() => setName(resume.name)}>
          rename
        </button>
        <a
          className="link-btn"
          href={resumeFileUrl(resume.id)}
          target="_blank"
          rel="noreferrer"
        >
          view ↗
        </a>
        <button className="link-btn danger" onClick={onDelete}>
          delete
        </button>
      </div>
    </div>
  );
}
