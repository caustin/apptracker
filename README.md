# AppTracker

A personal job-search tracker. Log the positions you pursue, the recruiters and interviewers you talk to, and every email, call, interview, and follow-up along the way. Upload the PDF resumes you send out and attach one to each position. A dashboard shows your pipeline as a stage funnel, weekly activity, progress against your application goals, and upcoming follow-ups.

## Requirements

- Node.js 20+

## Develop

```sh
npm install
npm run dev
```

Open http://localhost:5173. The API and the web app reload on change.

## Build & run

```sh
npm run build
npm start
```

Open http://localhost:3001.

Data is stored in `data/` — `apptracker.db` (SQLite) plus uploaded resume PDFs in `data/resumes/`. Back it up by copying that folder.
