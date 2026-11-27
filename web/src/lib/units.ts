import i18n from "i18next";

import type { Unit } from "./api";

// The server stores grams, metres and seconds; the phone converts for display and input.
const gramsPerLb = 453.59237;

/** Weight typed in unit, in grams; null when blank, undefined when not a number ≥ 0. */
export function parseWeight(text: string, unit: Unit): number | null | undefined {
  const n = parseNumber(text);
  if (n === null || n === undefined) return n;
  return Math.round(n * (unit === "kg" ? 1000 : gramsPerLb));
}

/** n in the UI language: "6,505.5" in English, "6 505,5" in French. Fields pass grouping false, so what
 * they show parses back (parseNumber takes "," or "."). */
export function formatNumber(n: number, opts: Intl.NumberFormatOptions = {}, grouping = true): string {
  const lang = i18n.language === "fr" ? "fr" : "en";
  return new Intl.NumberFormat(lang, { maximumFractionDigits: 2, useGrouping: grouping, ...opts }).format(n);
}

/** Grams as a number in unit: kg to 0.01, lb to 0.5. */
export const weightIn = (g: number, unit: Unit) => (unit === "kg" ? Math.round(g / 10) / 100 : Math.round((g / gramsPerLb) * 2) / 2);

/** Grams shown in unit, kg to 0.01 and lb to 0.5; field true for an input's value. */
export const formatWeight = (g: number, unit: Unit, field = false) => formatNumber(weightIn(g, unit), {}, !field);

/** "8" or "8-12" as a rep range; undefined when it isn't one. */
export function parseReps(text: string): { min: number | null; max: number | null } | undefined {
  const t = text.trim();
  if (!t) return { min: null, max: null };
  const m = /^(\d+)\s*(?:[-–]\s*(\d+))?$/.exec(t);
  if (!m) return undefined;
  const min = Number(m[1]);
  const max = m[2] === undefined ? min : Number(m[2]);
  return min <= max ? { min, max } : undefined;
}

export function formatReps(min: number | null | undefined, max: number | null | undefined): string {
  if (min == null && max == null) return "";
  if (min == null || max == null || min === max) return String(min ?? max);
  return `${min}–${max}`;
}

/** "1:30" or "90" (seconds) as seconds; null when blank, undefined when invalid. */
export function parseDuration(text: string): number | null | undefined {
  const t = text.trim();
  if (!t) return null;
  const m = /^(?:(\d+):)?(?:(\d+):)?(\d+)$/.exec(t);
  if (!m) return undefined;
  const parts = [m[1], m[2], m[3]].filter((p) => p !== undefined).map(Number);
  return parts.reduce((s, p) => s * 60 + p, 0);
}

/** Seconds as m:ss, or h:mm:ss from an hour. */
export function formatDuration(s: number): string {
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = String(s % 60).padStart(2, "0");
  return h ? `${h}:${String(m).padStart(2, "0")}:${sec}` : `${m}:${sec}`;
}

/** Kilometres typed, in metres. */
export function parseKm(text: string): number | null | undefined {
  const n = parseNumber(text);
  return n === null || n === undefined ? n : Math.round(n * 1000);
}

/** Metres as km; field true for an input's value. */
export const formatKm = (m: number, field = false) => formatNumber(Math.round(m) / 1000, { maximumFractionDigits: 3 }, !field);

/** A decimal number ≥ 0, with a dot or a comma; null when blank. */
function parseNumber(text: string): number | null | undefined {
  const t = text.trim().replace(",", ".");
  if (!t) return null;
  if (!/^\d*\.?\d+$|^\d+\.$/.test(t)) return undefined;
  return Number(t);
}
