import * as React from "react";

type TerminalLaunchReplicaProps = {
  className?: string;
  commandTail?: string;
};

const logo = [
  "          ",
  "▄▄▄▄▄▄▄▄▄ ",
  "▀▀▀▀▀████ ",
  "   ▄███▀  ",
  " ▄███▀    ",
  "█████████ ",
  "          ",
] as const;

export function TerminalLaunchReplica({
  className,
  commandTail = "run ./cmd/zeb tui",
}: TerminalLaunchReplicaProps) {
  let index = 0;

  return (
    <section className={["zeb-terminal", className].filter(Boolean).join(" ")} aria-label="Zeb terminal launch preview">
      <style>{styles}</style>
      <div className="zeb-terminal-chrome" aria-hidden="true">
        <span />
        <span />
        <span />
      </div>
      <pre className="zeb-terminal-screen">
        <span className="zeb-prompt">›</span> <span className="zeb-go">go</span>{" "}
        <span className="zeb-command-path">{commandTail}</span>
        {"\n\n"}
        <span className="zeb-muted">   charging Zeb</span>
        {"\n\n"}
        <span className="zeb-mark" aria-hidden="true">
          <span className="zeb-scan" />
          {logo.map((line, row) => (
            <React.Fragment key={`${row}-${line}`}>
              {[...line].map((glyph, col) => {
                if (glyph === " ") {
                  return " ";
                }
                const cellIndex = index;
                index += 1;
                return (
                  <span
                    key={`${row}-${col}`}
                    className="zeb-cell"
                    data-kind={glyph === "█" ? "mass" : glyph === "▀" ? "edge" : "cap"}
                    style={{ "--i": cellIndex } as React.CSSProperties}
                  >
                    {glyph}
                  </span>
                );
              })}
              {"\n"}
            </React.Fragment>
          ))}
        </span>
        {"\n"}
        <span className="zeb-muted">   Welcome to Zeb, the Zebra Link CLI</span>
        {"\n"}
        <span className="zeb-title">   Zeb</span>
        {"\n"}
        <span className="zeb-muted">   Welcome to Zeb, the Zebra Link CLI. Press q to quit.</span>
        {"\n"}
        <span className="zeb-actions">   auth  space  spec  status  links</span>
      </pre>
    </section>
  );
}

const styles = `
.zeb-terminal {
  --zeb-terminal: #242a33;
  --zeb-terminal-deep: #1f2530;
  --zeb-border: rgba(255, 255, 255, 0.16);
  --zeb-ink: #f3f4f6;
  --zeb-muted: #8d9099;
  --zeb-accent: #ff74d4;
  --zeb-accent-2: #7dd7ff;
  --zeb-command: #82d9ff;

  overflow: hidden;
  border: 1px solid var(--zeb-border);
  border-radius: 8px;
  background: var(--zeb-terminal);
  color: var(--zeb-ink);
  box-shadow: 0 20px 80px rgba(0, 0, 0, 0.32);
}

.zeb-terminal-chrome {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 34px;
  padding: 0 14px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.09);
  background: var(--zeb-terminal-deep);
}

.zeb-terminal-chrome span {
  width: 11px;
  height: 11px;
  border-radius: 999px;
}

.zeb-terminal-chrome span:nth-child(1) {
  background: #ff5f57;
}

.zeb-terminal-chrome span:nth-child(2) {
  background: #ffbd2e;
}

.zeb-terminal-chrome span:nth-child(3) {
  background: #28c840;
}

.zeb-terminal-screen {
  margin: 0;
  padding: 34px 42px 38px;
  min-height: 430px;
  font-family:
    "SFMono-Regular", "SF Mono", Menlo, Consolas, "Liberation Mono",
    monospace;
  font-size: 18px;
  line-height: 1.45;
  white-space: pre;
}

.zeb-prompt,
.zeb-title {
  color: var(--zeb-accent);
  font-weight: 800;
}

.zeb-go {
  color: #c3de83;
}

.zeb-command-path {
  color: var(--zeb-ink);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.zeb-muted {
  color: var(--zeb-muted);
}

.zeb-actions {
  color: var(--zeb-command);
}

.zeb-mark {
  position: relative;
  display: inline-block;
}

.zeb-cell {
  display: inline-block;
  animation: zeb-terminal-pulse 3.8s cubic-bezier(0.45, 0, 0.2, 1) infinite;
  animation-delay: calc(var(--i) * 80ms);
}

.zeb-cell[data-kind="cap"] {
  color: #d8dade;
}

.zeb-cell[data-kind="mass"] {
  color: #f4f4f5;
}

.zeb-cell[data-kind="edge"] {
  color: #a8abb3;
}

.zeb-scan {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 4ch;
  background: linear-gradient(
    90deg,
    transparent,
    rgba(125, 215, 255, 0.32),
    rgba(255, 116, 212, 0.18),
    transparent
  );
  animation: zeb-terminal-scan 4.6s cubic-bezier(0.65, 0, 0.35, 1) infinite;
  pointer-events: none;
}

@keyframes zeb-terminal-pulse {
  0%,
  100% {
    opacity: 0.78;
    transform: translateY(0);
  }
  38% {
    color: var(--zeb-accent-2);
    opacity: 1;
    transform: translateY(-1px);
  }
  56% {
    color: var(--zeb-accent);
    opacity: 0.96;
  }
}

@keyframes zeb-terminal-scan {
  0% {
    transform: translateX(-5ch);
    opacity: 0;
  }
  18% {
    opacity: 1;
  }
  68% {
    transform: translateX(13ch);
    opacity: 0.65;
  }
  100% {
    transform: translateX(16ch);
    opacity: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .zeb-cell,
  .zeb-scan {
    animation: none;
  }
}
`;
