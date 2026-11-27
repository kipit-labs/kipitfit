import { describe, expect, it } from "vitest";

import { chartKinds, chartValue, formatRecord } from "./records";

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
