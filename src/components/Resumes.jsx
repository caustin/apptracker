import { useRef, useState } from "react";
import { api } from "../api";
import { FieldError, FormError, check, first, useForm } from "../lib/form";
import { useAction } from "../lib/use-action";
import { fmtDate, resumeFileUrl } from "../constants";

const MAX_RESUMES = 25;
const MAX_BYTES = 15 * 1024 * 1024;

// The same checks the server makes before it stores a file.
async function checkFile(file, count) {
  if (count >= MAX_RESUMES) {
    return `You already have ${MAX_RESUMES} resumes, the most allowed. Delete one first.`;
  }
  if (file.type !== "application/pdf" && !/\.pdf$/i.test(file.name)) {
    return `"${file.name}" isn't a PDF. Only PDF files can be uploaded.`;
  }
  if (file.size > MAX_BYTES) {
    return `"${file.name}" is larger than 15 MB.`;
  }
  const head = await file.slice(0, 5).text();
  if (head !== "%PDF-") {
    return `"${file.name}" doesn't look like a PDF file.`;
  }
  return null;
}

const validateName = (v) => ({
  name: first(check.required(v.name), check.maxLength(v.name, 200)),
});

export function Resumes({ db, onChange }) {
  const fileRef = useRef(null);
  const [pending, setPending] = useState(null); // the chosen File
  const [error, setError] = useState(null);
  const action = useAction();

  const choose = async (file) => {
    setError(null);
    const problem = await checkFile(file, db.resumes.length);
    if (problem) setError(problem);
    else setPending(file);
  };

  const del = action.run(async (id, name, usedBy) => {
    const warning =
      usedBy > 0
        ? `Delete "${name}"? It is attached to ${usedBy} position${usedBy === 1 ? "" : "s"}, which will be unlinked.`
        : `Delete "${name}"?`;
    if (!confirm(warning)) return;
    await api.remove("resumes", id);
    onChange();
  });

  return (
    <div>
      <div className="page-head">
        <h1>Resumes</h1>
        <button
          className="btn btn-primary"
          disabled={!!pending}
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

      {(error || action.error) && (
        <div className="error-banner" role="alert">
          {error || action.error}
        </div>
      )}

      {pending && (
        <UploadForm
          file={pending}
          onCancel={() => setPending(null)}
          onUploaded={() => {
            setPending(null);
            onChange();
          }}
        />
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

function UploadForm({ file, onCancel, onUploaded }) {
  const form = useForm(
    { name: file.name.replace(/\.pdf$/i, "") },
    validateName,
  );

  const upload = async ({ name }) => {
    await api.uploadResume(file, name.trim());
    onUploaded();
  };

  return (
    <form
      ref={form.ref}
      className="card form-card"
      noValidate
      onSubmit={form.submit(upload)}
    >
      <h2>Label this resume</h2>
      <p className="muted small upload-filename">{file.name}</p>
      <FormError form={form} />
      <label className="field">
        Label *
        <input
          {...form.input("name")}
          autoFocus
          placeholder="e.g. Platform-focused v3"
        />
        <FieldError form={form} name="name" />
      </label>
      <div className="form-actions">
        <button type="button" className="btn" onClick={onCancel}>
          Cancel
        </button>
        <button type="submit" className="btn btn-primary" disabled={form.busy}>
          {form.busy ? "Uploading…" : "Upload"}
        </button>
      </div>
    </form>
  );
}

function ResumeRow({ resume, usedBy, onDelete, onChange }) {
  const [renaming, setRenaming] = useState(false);

  return (
    <div className="resume-row">
      <div className="position-main">
        <span className="resume-icon" aria-hidden>
          PDF
        </span>
        {renaming ? (
          <RenameForm
            resume={resume}
            onDone={(changed) => {
              setRenaming(false);
              if (changed) onChange();
            }}
          />
        ) : (
          <span className="position-company">{resume.name}</span>
        )}
      </div>
      <div className="position-meta">
        <span className="muted">uploaded {fmtDate(resume.createdAt)}</span>
        <span className="muted">
          {usedBy === 0
            ? "not attached"
            : `used by ${usedBy} position${usedBy === 1 ? "" : "s"}`}
        </span>
        <button className="link-btn" onClick={() => setRenaming(true)}>
          rename
        </button>
        <a className="link-btn" href={resumeFileUrl(resume.id)} download>
          download
        </a>
        <button className="link-btn danger" onClick={onDelete}>
          delete
        </button>
      </div>
    </div>
  );
}

// Enter saves (or shows what's wrong), Escape cancels. Leaving the field
// saves a valid change and otherwise drops it.
function RenameForm({ resume, onDone: done }) {
  const form = useForm({ name: resume.name }, validateName);
  const unchanged = form.values.name.trim() === resume.name;

  // Finish once: a blur can still arrive after Enter or Escape.
  const finished = useRef(false);
  const onDone = (changed) => {
    if (finished.current) return;
    finished.current = true;
    done(changed);
  };

  const save = async ({ name }) => {
    if (finished.current) return;
    if (name.trim() === resume.name) return onDone(false);
    await api.update("resumes", resume.id, { name: name.trim() });
    onDone(true);
  };

  const { onBlur, ...input } = form.input("name");
  return (
    <form
      ref={form.ref}
      className="rename-field"
      noValidate
      onSubmit={form.submit(save)}
    >
      <input
        {...input}
        className="rename-input"
        aria-label="Resume label"
        autoFocus
        onKeyDown={(e) => {
          if (e.key === "Escape") onDone(false);
        }}
        onBlur={() => {
          if (form.busy || finished.current) return;
          if (unchanged || form.error("name") || !form.values.name.trim()) {
            onDone(false);
          } else {
            form.submit(save)();
          }
        }}
      />
      <FieldError form={form} name="name" />
      {form.formError && (
        <span className="field-error" role="alert">
          {form.formError}
        </span>
      )}
    </form>
  );
}
