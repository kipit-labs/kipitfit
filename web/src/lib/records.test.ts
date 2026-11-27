import { describe, expect, it } from "vitest";

import { readFileSync } from "node:fs";
import type { ExerciseKind, RecordKind, SetType } from "./api";
import { beaten, chartKinds, chartValue, e1rm, formatRecord, recordKinds, sessionBest } from "./records";

describe("formatRecord", () => {
  it("shows each kind in the user's unit", () => {
    expect(formatRecord("e1rm", 111000, "kg")).toBe("111.0 kg");
    expect(formatRecord("e1rm", 126750, "kg")).toBe("126.8 kg");
    expect(formatRecord("e1rm", 126750, "lb")).toBe("279.5 lb");
    expect(formatRecord("weight", 126750, "kg")).toBe("126.75 kg");
    expect(formatRecord("weight", 102058, "lb")).toBe("225 lb");
    expect(formatRecord("session_volume", 1975000, "kg")).toBe("1,975 kg");
    expect(formatRecord("reps", 12, "kg")).toBe("12");
    expect(formatRecord("duration", 90, "kg")).toBe("1:30");
    expect(formatRecord("distance", 5200, "lb")).toBe("5.2 km");
  });
});

describe("chart", () => {
  it("charts what a workout has one value of", () => {
    expect(chartKinds("weight_reps")).toEqual(["e1rm", "weight", "session_volume", "reps"]);
    expect(chartKinds("duration")).toEqual(["duration"]);
    expect(chartValue("weight", 102058, "lb")).toBe(225);
    expect(chartValue("distance", 5200, "kg")).toBe(5.2);
  });
});

// The cases internal/usecase/records/records_test.go runs too, so the phone badges what the server records. Read at run time:
// the Docker web build type-checks src without the Go tree.
type Case = {
  name: string;
  kind: ExerciseKind;
  sessions: { type: SetType; weightG?: number; reps?: number; durationS?: number; distanceM?: number }[][];
  beaten: Partial<Record<RecordKind, number>>[];
  bests: Partial<Record<RecordKind, number>>;
};
const cases: Case[] = JSON.parse(readFileSync(new URL("../../../internal/usecase/records/testdata/records.json", import.meta.url), "utf8"));
describe("records like the server", () => {
  it.each(cases.map((c) => [c.name, c] as const))("%s", (_, c) => {
    const kind = c.kind;
    let bests: Partial<Record<RecordKind, number>> | undefined;
    c.sessions.forEach((sets, i) => {
      const got = Object.fromEntries(beaten(kind, sets, bests).map((b) => [b.kind, b.value]));
      expect(got, `session ${i}`).toEqual(c.beaten[i]);
      for (const rk of recordKinds(kind)) {
        const b = sessionBest(rk, sets);
        if (b && (bests?.[rk] === undefined || b.value > bests[rk]!)) bests = { ...bests, [rk]: b.value };
      }
    });
    expect(bests ?? {}).toEqual(c.bests);
  });

  it("estimates a one-rep max like the server", () => {
    expect(e1rm(100000, 5)).toBe(116667);
    expect(e1rm(100000, 12)).toBe(140000);
    expect(e1rm(100000, 13)).toBeUndefined();
  });
});
