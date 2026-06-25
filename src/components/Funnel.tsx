import type { Position } from "../types";
import { STATUS_META } from "../types";

const STAGES = [
  { label: "Applied", min: 1 },
  { label: "Screening", min: 2 },
  { label: "Interviewing", min: 3 },
  { label: "Offer", min: 4 },
  { label: "Accepted", min: 5 },
];

const COLORS = ["#bcd4c2", "#94bda3", "#699f82", "#467f63", "#2e6b4f"];

export function Funnel({ positions }: { positions: Position[] }) {
  const counts = STAGES.map(
    (s) => positions.filter((p) => STATUS_META[p.status].order >= s.min).length,
  );
  const max = Math.max(...counts, 1);
  const leads = positions.filter((p) => p.status === "lead").length;
  const rejected = positions.filter((p) => p.status === "rejected").length;
  const withdrawn = positions.filter((p) => p.status === "withdrawn").length;

  if (counts[0] === 0) {
    return (
      <div className="empty-state">
        Nothing in the funnel yet — mark a position as applied and it will show
        up here.
        {leads > 0 && ` (${leads} lead${leads === 1 ? "" : "s"} waiting.)`}
      </div>
    );
  }

  const LABEL_W = 120;
  const CHART_W = 480;
  const BAND_H = 42;
  const GAP = 26;
  const cx = LABEL_W + CHART_W / 2;
  const width = (c: number) => 70 + (c / max) * (CHART_W - 70);
  const height = STAGES.length * BAND_H + (STAGES.length - 1) * GAP;

  return (
    <div>
      <svg
        viewBox={`0 0 ${LABEL_W + CHART_W} ${height}`}
        className="funnel"
        role="img"
        aria-label="Job search funnel by stage"
      >
        {STAGES.map((stage, i) => {
          const y = i * (BAND_H + GAP);
          const w = width(counts[i]);
          const dark = i >= 2;
          const pct =
            i > 0 && counts[i - 1] > 0
              ? Math.round((counts[i] / counts[i - 1]) * 100)
              : null;
          return (
            <g key={stage.label}>
              {i > 0 && (
                <>
                  <path
                    d={`M ${cx - width(counts[i - 1]) / 2} ${y - GAP}
                        L ${cx + width(counts[i - 1]) / 2} ${y - GAP}
                        L ${cx + w / 2} ${y} L ${cx - w / 2} ${y} Z`}
                    fill={COLORS[i]}
                    opacity="0.25"
                  />
                  {pct !== null && (
                    <text
                      x={cx + CHART_W / 2 - 4}
                      y={y - GAP / 2 + 4}
                      textAnchor="end"
                      className="funnel-pct"
                    >
                      {pct}% →
                    </text>
                  )}
                </>
              )}
              <rect
                x={cx - w / 2}
                y={y}
                width={w}
                height={BAND_H}
                rx="6"
                fill={COLORS[i]}
              />
              <text
                x={cx}
                y={y + BAND_H / 2 + 6}
                textAnchor="middle"
                className={`funnel-count ${dark ? "on-dark" : ""}`}
              >
                {counts[i]}
              </text>
              <text
                x={LABEL_W - 16}
                y={y + BAND_H / 2 + 5}
                textAnchor="end"
                className="funnel-label"
              >
                {stage.label}
              </text>
            </g>
          );
        })}
      </svg>
      <p className="funnel-footnote">
        {leads > 0 && (
          <span>
            {leads} lead{leads === 1 ? "" : "s"} not yet applied ·{" "}
          </span>
        )}
        {rejected + withdrawn > 0 ? (
          <span>
            {rejected + withdrawn} closed ({rejected} rejected, {withdrawn}{" "}
            withdrawn)
          </span>
        ) : (
          <span>no closed positions yet</span>
        )}
      </p>
    </div>
  );
}
