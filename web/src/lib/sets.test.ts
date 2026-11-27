import { describe, expect, it } from "vitest";

import { fieldsOf, setLabels, setText } from "./sets";

describe("sets", () => {
  it("numbers normal sets and letters the others", () => {
    const letters = { normal: "", warmup: "W", drop: "D", failure: "F" };
    expect(setLabels(["warmup", "normal", "normal", "drop", "failure", "normal"], letters)).toEqual(["W", "1", "2", "D", "F", "3"]);
  });
  it("gives each kind its fields", () => {
    expect(fieldsOf("reps_only")).toEqual({ weight: false, reps: true, distance: false, duration: false });
    expect(fieldsOf("distance_duration")).toEqual({ weight: false, reps: false, distance: true, duration: true });
  });
  it("writes a set in a line", () => {
    expect(setText({ weightG: 100000, reps: 5 }, "weight_reps", "kg")).toBe("100 × 5");
    expect(setText({ weightG: 100000, reps: 5 }, "weight_reps", "kg", true)).toBe("100 kg × 5");
    expect(setText({ weightG: 20000, reps: 8 }, "weighted_bodyweight", "kg", true)).toBe("+20 kg × 8");
    expect(setText({ reps: 12 }, "reps_only", "lb", true)).toBe("× 12");
    expect(setText({ distanceM: 5000, durationS: 1500 }, "distance_duration", "kg")).toBe("5 km · 25:00");
  });
});
