import { Dumbbell } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@kipit-labs/kipitidy/chart";
import { EmptyState } from "@kipit-labs/kipitidy/empty-state";
import { Section } from "@kipit-labs/kipitidy/section";
import { Segmented, SegmentedItem } from "@kipit-labs/kipitidy/segmented";

import type { ExerciseKind, ExerciseProgress, ExerciseSession, PersonalRecord, RecordKind, Unit } from "@/lib/api";
import { chartKinds, chartValue, formatRecord } from "@/lib/records";
import { setLabels, setText } from "@/lib/sets";
import { formatDuration, formatNumber, formatWeight } from "@/lib/units";
import { currentLanguage } from "@/i18n";

import { RecordBadge, RecordMark } from "./RecordBadge";

const shortDate = (iso: string) => new Date(iso).toLocaleDateString(currentLanguage(), { day: "numeric", month: "short" });
const longDate = (iso: string) => new Date(iso).toLocaleDateString(currentLanguage(), { dateStyle: "medium" });

export function NoHistory() {
  const { t } = useTranslation();
  return <EmptyState icon={Dumbbell} title={t("records.noHistory")} description={t("records.noHistoryDescription")} />;
}

/** A workout-by-workout line of one of the exercise's bests. */
export function ProgressChart({ kind, sessions, unit }: { kind: ExerciseKind; sessions: ExerciseSession[]; unit: Unit }) {
  const { t } = useTranslation();
  const kinds = chartKinds(kind).filter((rk) => sessions.some((s) => s.best[rk] != null));
  const [picked, setPicked] = useState<RecordKind>();
  const rk = picked && kinds.includes(picked) ? picked : kinds[0];
  if (!rk) return null;
  const data = sessions
    .filter((s) => s.best[rk] != null)
    .reverse()
    .map((s) => ({ date: s.startedAt, raw: s.best[rk]!, value: chartValue(rk, s.best[rk]!, unit) }));
  return (
    <Section title={t("records.chartTitle")}>
      {kinds.length > 1 && (
        <Segmented size="sm" value={rk} onValueChange={(v) => v && setPicked(v as RecordKind)} className="w-full">
          {kinds.map((k) => (
            <SegmentedItem key={k} value={k} className="flex-1">
              {t(`records.short.${k}`)}
            </SegmentedItem>
          ))}
        </Segmented>
      )}
      <ChartContainer
        config={{ value: { label: t(`records.kinds.${rk}`), color: "var(--primary)" } }}
        className="aspect-auto h-48 w-full"
        role="img"
        aria-label={t(`records.kinds.${rk}`)}
      >
        <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid vertical={false} />
          <XAxis dataKey="date" tickFormatter={shortDate} tickLine={false} axisLine={false} minTickGap={24} />
          <YAxis
            width={44}
            tickLine={false}
            axisLine={false}
            domain={["auto", "auto"]}
            tickFormatter={(v: number) => (rk === "duration" ? formatDuration(v) : formatNumber(v, { maximumFractionDigits: 1 }))}
          />
          <ChartTooltip
            content={
              <ChartTooltipContent
                labelFormatter={(_, p) => longDate(p[0]?.payload.date)}
                formatter={(_v, _n, item) => (
                  <span className="flex w-full justify-between gap-3">
                    <span className="text-muted-foreground">{t(`records.short.${rk}`)}</span>
                    <span className="font-medium tabular-nums">{formatRecord(rk, item.payload.raw, unit)}</span>
                  </span>
                )}
              />
            }
          />
          <Line dataKey="value" type="monotone" stroke="var(--color-value)" strokeWidth={2} dot={{ r: 3 }} />
        </LineChart>
      </ChartContainer>
    </Section>
  );
}

/** The exercise's current bests, then its rep records. */
export function Records({ kind, progress, unit }: { kind: ExerciseKind; progress: ExerciseProgress; unit: Unit }) {
  const { t } = useTranslation();
  if (!progress.records.length) return <NoHistory />;
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-2">
        {progress.records.map((r) => (
          <div key={r.kind} className="rounded-xl border bg-card p-3">
            <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <RecordMark />
              {t(`records.kinds.${r.kind}`)}
            </p>
            <p className="mt-1 text-lg font-semibold tabular-nums">{formatRecord(r.kind, r.value, unit)}</p>
            <p className="text-xs text-muted-foreground">
              {[detail(r, kind, unit), longDate(r.startedAt)].filter(Boolean).join(" · ")}
            </p>
          </div>
        ))}
      </div>
      {progress.repRecords.length > 0 && (
        <Section title={t("records.repRecords")} description={t("records.repRecordsDescription")}>
          <table className="w-full text-sm">
            <thead className="text-xs text-muted-foreground">
              <tr>
                <th className="pb-1 text-left font-normal">{t("records.reps")}</th>
                <th className="pb-1 text-right font-normal">{t("records.weight", { unit })}</th>
                <th className="pb-1 text-right font-normal">{t("records.date")}</th>
              </tr>
            </thead>
            <tbody className="tabular-nums">
              {progress.repRecords.map((r) => (
                <tr key={r.reps} className="border-t">
                  <td className="py-1.5">{r.reps}</td>
                  <td className="py-1.5 text-right font-medium">{formatWeight(r.weightG, unit)}</td>
                  <td className="py-1.5 text-right text-muted-foreground">{longDate(r.startedAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Section>
      )}
    </div>
  );
}

/** The set a record was set with, when it says more than the value: nothing for a workout total or a lone value. */
function detail(r: PersonalRecord, kind: ExerciseKind, unit: Unit): string {
  if (r.set < 0 || (r.weightG == null && kind !== "distance_duration")) return "";
  return setText(r, kind, unit, true);
}

/** Every workout the exercise was done in, newest first, with the records it beat. */
export function History({ kind, sessions, unit }: { kind: ExerciseKind; sessions: ExerciseSession[]; unit: Unit }) {
  const { t } = useTranslation();
  if (!sessions.length) return <NoHistory />;
  const letters = { normal: "", warmup: t("sets.letters.warmup"), drop: t("sets.letters.drop"), failure: t("sets.letters.failure") };
  return (
    <ul className="space-y-2">
      {sessions.map((s) => {
        const labels = setLabels(
          s.sets.map((x) => x.type),
          letters,
        );
        return (
          <li key={s.workoutId}>
            <Section title={s.title} description={longDate(s.startedAt)}>
              {s.records.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {s.records.map((r) => (
                    <RecordBadge key={r.kind} kind={r.kind} value={r.value} unit={unit} />
                  ))}
                </div>
              )}
              <ol className="space-y-1 text-sm tabular-nums">
                {s.sets.map((x, i) => (
                  <li key={i} className="flex gap-3">
                    <span className="w-5 text-center text-muted-foreground">{labels[i]}</span>
                    <span>{setText(x, kind, unit, true)}</span>
                    {s.records.some((r) => r.set === i) && <RecordMark />}
                  </li>
                ))}
              </ol>
            </Section>
          </li>
        );
      })}
    </ul>
  );
}
