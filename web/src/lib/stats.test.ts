import { describe, expect, it } from "vitest";

import type { WorkoutStats } from "./api";
import { byDay, dayKey, muscleSets, statsFrom, weekOf, weeks } from "./stats";

// Local times, so the test holds in any time zone.
const at = (y: number, m: number, d: number, h = 18) => new Date(y, m - 1, d, h);
const workout = (start: Date, minutes: number, volumeG: number, muscles: Record<string, number>): WorkoutStats => ({
  id: start.toISOString(),
  startedAt: start.toISOString(),
  endedAt: new Date(start.getTime() + minutes * 60_000).toISOString(),
  volumeG,
  sets: 0,
  muscles,
});

const now = at(2026, 11, 27, 12); // a Friday
const stats = [
  workout(at(2026, 11, 2), 60, 1000, { chest: 3 }), // Monday 4 weeks back: out of 3 weeks
  workout(at(2026, 11, 16, 23), 45, 2000, { chest: 4, triceps: 2 }), // Monday two weeks back, late
  workout(at(2026, 11, 22, 9), 30, 500, { quadriceps: 5 }), // Sunday: last week
  workout(at(2026, 11, 24), 60, 3000, { chest: 2 }),
  workout(at(2026, 11, 27, 7), 90, 4000, { lats: 6 }),
];

describe("stats", () => {
  it("starts weeks on Monday, local", () => {
    expect(dayKey(weekOf(now))).toBe("2026-11-23");
    expect(dayKey(weekOf(at(2026, 11, 23, 0)))).toBe("2026-11-23");
    expect(dayKey(weekOf(at(2026, 11, 22, 23)))).toBe("2026-11-16");
    expect(dayKey(statsFrom(now))).toBe("2025-11-01");
  });

  it("sums the last weeks, oldest first", () => {
    const w = weeks(stats, now, 3);
    expect(w.map((x) => dayKey(x.start))).toEqual(["2026-11-09", "2026-11-16", "2026-11-23"]);
    expect(w.map((x) => x.workouts)).toEqual([0, 2, 2]);
    expect(w[1]).toMatchObject({ durationS: 75 * 60, volumeG: 2500 });
    expect(w[2]).toMatchObject({ durationS: 150 * 60, volumeG: 7000 });
  });

  it("groups workouts by local day", () => {
    const d = byDay(stats);
    expect(d.get("2026-11-16")).toHaveLength(1);
    expect(d.has("2026-11-17")).toBe(false);
  });

  it("counts sets per muscle over the last days", () => {
    expect(muscleSets(stats, now, 7)).toEqual([
      { muscle: "lats", sets: 6 },
      { muscle: "quadriceps", sets: 5 },
      { muscle: "chest", sets: 2 },
    ]);
    expect(muscleSets(stats, now, 30)[0]).toEqual({ muscle: "chest", sets: 9 });
  });
});
