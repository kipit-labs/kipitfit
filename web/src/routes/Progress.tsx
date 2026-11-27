import { useQuery } from "@tanstack/react-query";
import { CalendarCheck, ChartLine, ChevronLeft, ChevronRight, Dumbbell, SearchX, Timer } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { cn } from "cn";
import { Button } from "@kipit-labs/kipitidy/button";
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@kipit-labs/kipitidy/chart";
import { EmptyState } from "@kipit-labs/kipitidy/empty-state";
import { Loading } from "@kipit-labs/kipitidy/loading";
import { Section } from "@kipit-labs/kipitidy/section";
import { Segmented, SegmentedItem } from "@kipit-labs/kipitidy/segmented";

import { api, type Unit, type WorkoutStats } from "@/lib/api";
import { useMe } from "@/lib/auth";
import { errorMessage } from "@/lib/errors";
import { byDay, dayKey, dayOf, muscleSets, statsFrom, weekOf, weeks, type Week } from "@/lib/stats";
import { formatNumber, formatWeight, weightIn } from "@/lib/units";
import { currentLanguage } from "@/i18n";

import { PageTitle } from "./Layout";

type Metric = "workouts" | "durationS" | "volumeG";

/** The Progress tab: this week against the last, twelve weeks in a chart, the calendar, sets per muscle. */
export function Progress() {
  const { t } = useTranslation();
  const { unit } = useMe();
  // One "now" for the page, so the weeks, the calendar and the query agree.
  const [now] = useState(() => new Date());
  const stats = useQuery({ queryKey: ["progress", "stats", dayKey(now)], queryFn: () => api.stats(statsFrom(now)) });
  return (
    <>
      <PageTitle>{t("nav.progress")}</PageTitle>
      {stats.isPending ? (
        <Loading className="py-12" />
      ) : stats.isError ? (
        <EmptyState icon={SearchX} title={errorMessage(stats.error)} action={<Button onClick={() => stats.refetch()}>{t("common.retry")}</Button>} />
      ) : stats.data.length === 0 ? (
        <EmptyState icon={ChartLine} title={t("progress.emptyTitle")} description={t("progress.emptyDescription")} />
      ) : (
        <div className="space-y-4">
          <ThisWeek stats={stats.data} now={now} unit={unit} />
          <WeeksChart stats={stats.data} now={now} unit={unit} />
          <Calendar stats={stats.data} now={now} />
          <Muscles stats={stats.data} now={now} />
        </div>
      )}
    </>
  );
}

/** "45 min", "3 h 05": a week's time at a glance. */
function durationText(s: number) {
  const m = Math.round(s / 60);
  // No-break spaces: "3 h 20" stays on one line.
  return m < 60 ? `${m}\u00a0min` : `${Math.floor(m / 60)}\u00a0h\u00a0${String(m % 60).padStart(2, "0")}`;
}

function metricText(m: Metric, w: Week, unit: Unit) {
  if (m === "durationS") return durationText(w.durationS);
  if (m === "volumeG") return `${formatWeight(w.volumeG, unit)} ${unit}`;
  return String(w.workouts);
}

function ThisWeek({ stats, now, unit }: { stats: WorkoutStats[]; now: Date; unit: Unit }) {
  const { t } = useTranslation();
  const [last, current] = weeks(stats, now, 2);
  const facts: [Metric, typeof Dumbbell, string][] = [
    ["workouts", CalendarCheck, t("progress.workouts")],
    ["durationS", Timer, t("live.duration")],
    ["volumeG", Dumbbell, t("live.volume")],
  ];
  return (
    <Section title={t("progress.thisWeek")}>
      <dl className="grid grid-cols-3 gap-2">
        {facts.map(([m, Icon, label]) => (
          <div key={m}>
            <dt className="flex items-center gap-1 text-xs text-muted-foreground">
              <Icon className="size-3.5 shrink-0" aria-hidden />
              {label}
            </dt>
            <dd className="text-lg font-semibold tabular-nums">{metricText(m, current, unit)}</dd>
            <dd className="text-xs text-muted-foreground tabular-nums">{t("progress.lastWeek", { value: metricText(m, last, unit) })}</dd>
          </div>
        ))}
      </dl>
    </Section>
  );
}

function WeeksChart({ stats, now, unit }: { stats: WorkoutStats[]; now: Date; unit: Unit }) {
  const { t } = useTranslation();
  const [metric, setMetric] = useState<Metric>("workouts");
  const short = (d: Date) => d.toLocaleDateString(currentLanguage(), { day: "numeric", month: "short" });
  const data = weeks(stats, now, 12).map((w) => ({
    week: w,
    label: short(w.start),
    // Hours and kg/lb, so the axis reads in round numbers.
    value: metric === "durationS" ? w.durationS / 3600 : metric === "volumeG" ? weightIn(w.volumeG, unit) : w.workouts,
  }));
  return (
    <Section title={t("progress.weeks")}>
      <Segmented size="sm" value={metric} onValueChange={(v) => v && setMetric(v as Metric)} className="w-full">
        <SegmentedItem value="workouts" className="flex-1">
          {t("progress.workouts")}
        </SegmentedItem>
        <SegmentedItem value="durationS" className="flex-1">
          {t("live.duration")}
        </SegmentedItem>
        <SegmentedItem value="volumeG" className="flex-1">
          {t("live.volume")}
        </SegmentedItem>
      </Segmented>
      <ChartContainer
        config={{ value: { label: t(`progress.${metric === "workouts" ? "workouts" : metric === "durationS" ? "hours" : "volume"}`, { unit }), color: "var(--primary)" } }}
        className="aspect-auto h-48 w-full"
        role="img"
        aria-label={t("progress.weeks")}
      >
        <BarChart data={data} margin={{ top: 8, right: 0, left: 0, bottom: 0 }}>
          <CartesianGrid vertical={false} />
          <XAxis dataKey="label" tickLine={false} axisLine={false} minTickGap={16} />
          <YAxis
            width={44}
            tickLine={false}
            axisLine={false}
            allowDecimals={metric !== "workouts"}
            tickFormatter={(v: number) => formatNumber(v, { maximumFractionDigits: 1 })}
          />
          <ChartTooltip
            cursor={false}
            content={
              <ChartTooltipContent
                labelFormatter={(_, p) => t("progress.weekOf", { date: short(p[0]?.payload.week.start) })}
                formatter={(_v, _n, item) => <span className="font-medium tabular-nums">{metricText(metric, item.payload.week, unit)}</span>}
              />
            }
          />
          <Bar dataKey="value" fill="var(--color-value)" radius={4} />
        </BarChart>
      </ChartContainer>
    </Section>
  );
}

/** A month of training days; back up to the first month the stats cover. */
function Calendar({ stats, now }: { stats: WorkoutStats[]; now: Date }) {
  const { t } = useTranslation();
  const [back, setBack] = useState(0);
  const lang = currentLanguage();
  const month = new Date(now.getFullYear(), now.getMonth() - back, 1);
  const first = weekOf(month);
  const days = byDay(stats);
  const cells = Array.from({ length: 42 }, (_, i) => new Date(first.getFullYear(), first.getMonth(), first.getDate() + i)).filter(
    (d, i) => i < 35 || d.getMonth() === month.getMonth(),
  );
  const today = dayOf(now).getTime();
  const weekdays = cells.slice(0, 7).map((d) => d.toLocaleDateString(lang, { weekday: "narrow" }));
  const trained = [...days.keys()].filter((k) => k.startsWith(dayKey(month).slice(0, 8))).length;
  return (
    <Section
      title={month.toLocaleDateString(lang, { month: "long", year: "numeric" })}
      description={t("progress.trainingDays", { count: trained })}
      actions={
        <div className="flex gap-1">
          <Button variant="ghost" size="icon-sm" aria-label={t("progress.previousMonth")} disabled={back >= 12} onClick={() => setBack(back + 1)}>
            <ChevronLeft />
          </Button>
          <Button variant="ghost" size="icon-sm" aria-label={t("progress.nextMonth")} disabled={back === 0} onClick={() => setBack(back - 1)}>
            <ChevronRight />
          </Button>
        </div>
      }
    >
      <div className="grid grid-cols-7 gap-1 text-center text-sm">
        {weekdays.map((d, i) => (
          <span key={i} className="pb-1 text-xs text-muted-foreground" aria-hidden>
            {d}
          </span>
        ))}
        {cells.map((d) => {
          const ws = days.get(dayKey(d));
          const out = d.getMonth() !== month.getMonth();
          const cls = cn(
            "mx-auto flex size-9 items-center justify-center rounded-full tabular-nums",
            out && "text-muted-foreground/50",
            d.getTime() === today && "ring-2 ring-primary",
            ws && !out && "bg-primary font-medium text-primary-foreground",
          );
          return ws && !out ? (
            <Link
              key={d.getTime()}
              to={`/workouts/${ws[0].id}`}
              className={cls}
              aria-label={t("progress.trainedOn", { date: d.toLocaleDateString(lang, { dateStyle: "full" }), count: ws.length })}
            >
              {d.getDate()}
            </Link>
          ) : (
            <span key={d.getTime()} className={cls}>
              {d.getDate()}
            </span>
          );
        })}
      </div>
    </Section>
  );
}

function Muscles({ stats, now }: { stats: WorkoutStats[]; now: Date }) {
  const { t } = useTranslation();
  const [days, setDays] = useState("7");
  const rows = muscleSets(stats, now, Number(days));
  const most = rows[0]?.sets ?? 1;
  return (
    <Section title={t("progress.muscles")} description={t("progress.musclesDescription")}>
      <Segmented size="sm" value={days} onValueChange={(v) => v && setDays(v)} className="w-full">
        <SegmentedItem value="7" className="flex-1">
          {t("progress.days", { count: 7 })}
        </SegmentedItem>
        <SegmentedItem value="30" className="flex-1">
          {t("progress.days", { count: 30 })}
        </SegmentedItem>
      </Segmented>
      {rows.length === 0 ? (
        <p className="py-4 text-center text-sm text-muted-foreground">{t("progress.noSets")}</p>
      ) : (
        <ul className="space-y-2">
          {rows.map((r) => (
            <li key={r.muscle} className="grid grid-cols-[7rem_1fr_2rem] items-center gap-2 text-sm">
              <span className="truncate">{t(`exercises.muscles.${r.muscle}`)}</span>
              <span className="h-2 rounded-full bg-muted">
                <span className="block h-2 rounded-full bg-primary" style={{ width: `${(r.sets / most) * 100}%` }} />
              </span>
              <span className="text-right font-medium tabular-nums">{r.sets}</span>
            </li>
          ))}
        </ul>
      )}
    </Section>
  );
}
