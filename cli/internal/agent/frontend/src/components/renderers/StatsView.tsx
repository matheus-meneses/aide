interface DailyData {
  date: string;
  agent: number;
  chat: number;
}

interface Summary {
  today_tokens: number;
  week_tokens: number;
  total_calls: number;
  avg_per_day: number;
}

interface Props {
  data: {
    summary: Summary;
    daily: DailyData[];
  };
}

function formatTokens(n: number): string {
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`;
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return `${n}`;
}

function dayLabel(dateStr: string): string {
  const d = new Date(dateStr + "T12:00:00");
  return d.toLocaleDateString("en", { weekday: "short" });
}

export function StatsView({ data }: Props) {
  if (!data?.summary) return null;

  const { summary, daily } = data;
  const maxTotal = Math.max(...(daily || []).map((d) => d.agent + d.chat), 1);

  const chartHeight = 160;
  const barWidth = 32;
  const gap = 12;
  const chartWidth = (daily?.length || 7) * (barWidth + gap);

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <StatCard label="Today" value={formatTokens(summary.today_tokens)} />
        <StatCard label="This week" value={formatTokens(summary.week_tokens)} />
        <StatCard label="Avg/day" value={formatTokens(summary.avg_per_day)} />
        <StatCard label="Calls (7d)" value={`${summary.total_calls}`} />
      </div>

      {daily && daily.length > 0 && (
        <div className="rounded-lg border bg-card p-4">
          <div className="text-xs text-muted-foreground mb-3 font-medium">
            Daily Token Usage (7 days)
          </div>
          <div className="flex items-end justify-center gap-1" style={{ height: chartHeight + 30 }}>
            <svg width={chartWidth} height={chartHeight + 24} className="overflow-visible">
              {daily.map((d, i) => {
                const total = d.agent + d.chat;
                const agentH = (d.agent / maxTotal) * chartHeight;
                const chatH = (d.chat / maxTotal) * chartHeight;
                const x = i * (barWidth + gap);
                const agentY = chartHeight - agentH;
                const chatY = agentY - chatH;

                return (
                  <g key={d.date}>
                    <rect
                      x={x}
                      y={agentY}
                      width={barWidth}
                      height={agentH}
                      rx={3}
                      className="fill-blue-500/80"
                    />
                    <rect
                      x={x}
                      y={chatY}
                      width={barWidth}
                      height={chatH}
                      rx={3}
                      className="fill-emerald-500/80"
                    />
                    <text
                      x={x + barWidth / 2}
                      y={chartHeight + 14}
                      textAnchor="middle"
                      className="fill-muted-foreground text-[10px]"
                    >
                      {dayLabel(d.date)}
                    </text>
                    {total > 0 && (
                      <text
                        x={x + barWidth / 2}
                        y={chatY - 4}
                        textAnchor="middle"
                        className="fill-muted-foreground text-[9px]"
                      >
                        {formatTokens(total)}
                      </text>
                    )}
                  </g>
                );
              })}
            </svg>
          </div>
          <div className="flex items-center justify-center gap-4 mt-3 text-[10px] text-muted-foreground">
            <span className="flex items-center gap-1">
              <span className="w-2.5 h-2.5 rounded-sm bg-blue-500/80 inline-block" /> Agent
            </span>
            <span className="flex items-center gap-1">
              <span className="w-2.5 h-2.5 rounded-sm bg-emerald-500/80 inline-block" /> Chat
            </span>
          </div>
        </div>
      )}
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-card p-3 text-center">
      <div className="text-lg font-semibold">{value}</div>
      <div className="text-[10px] text-muted-foreground">{label}</div>
    </div>
  );
}
