import type { WorkoutStats } from "./api";

// The server sends workouts with UTC times; weeks and days are the phone's own, so a late workout lands on its local day.

const dayMs = 86_400_000;

/** Local midnight of d's day. */
export const dayOf = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate());

/** "2026-11-27" for d's local day. */
export const dayKey = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;

/** Local midnight of the Monday starting d's week. */
export function weekOf(d: Date): Date {
  const day = dayOf(d);
  return new Date(day.getFullYear(), day.getMonth(), day.getDate() - ((day.getDay() + 6) % 7));
}

/** What the Progress tab asks for: from the first day of the month a year back, for the calendar and the weeks. */
export const statsFrom = (now: Date) => new Date(now.getFullYear() - 1, now.getMonth(), 1);

export type Week = { start: Date; workouts: number; durationS: number; volumeG: number };

/** The last n weeks up to now's, oldest first, with what was done in each. */
export function weeks(stats: WorkoutStats[], now: Date, n: number): Week[] {
  const last = weekOf(now);
  const out: Week[] = Array.from({ length: n }, (_, i) => ({
    start: new Date(last.getFullYear(), last.getMonth(), last.getDate() - 7 * (n - 1 - i)),
    workouts: 0,
    durationS: 0,
    volumeG: 0,
  }));
  for (const w of stats) {
    const start = new Date(w.startedAt);
    const week = out.find((x) => x.start.getTime() === weekOf(start).getTime());
    if (!week) continue;
    week.workouts++;
    week.durationS += Math.round((Date.parse(w.endedAt) - start.getTime()) / 1000);
    week.volumeG += w.volumeG;
  }
  return out;
}

/** The workouts of each local day, by dayKey. */
export function byDay(stats: WorkoutStats[]): Map<string, WorkoutStats[]> {
  const out = new Map<string, WorkoutStats[]>();
  for (const w of stats) {
    const k = dayKey(new Date(w.startedAt));
    out.set(k, [...(out.get(k) ?? []), w]);
  }
  return out;
}

/** Working sets per muscle over the last days (today included), most first. */
export function muscleSets(stats: WorkoutStats[], now: Date, days: number): { muscle: string; sets: number }[] {
  const from = dayOf(now).getTime() - (days - 1) * dayMs;
  const sums: Record<string, number> = {};
  for (const w of stats) {
    if (Date.parse(w.startedAt) < from) continue;
    for (const [m, n] of Object.entries(w.muscles)) sums[m] = (sums[m] ?? 0) + n;
  }
  return Object.entries(sums)
    .map(([muscle, sets]) => ({ muscle, sets }))
    .sort((a, b) => b.sets - a.sets || a.muscle.localeCompare(b.muscle));
}
