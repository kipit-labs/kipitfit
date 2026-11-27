import {
  ArrowDown,
  ArrowUp,
  Check,
  ChevronDown,
  Gauge,
  Link2,
  Link2Off,
  MoreVertical,
  Plus,
  Replace,
  Trash2,
} from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, Navigate, useNavigate } from "react-router";
import { cn } from "cn";
import { Button } from "@kipit-labs/kipitidy/button";
import { ConfirmDialog } from "@kipit-labs/kipitidy/confirm-dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@kipit-labs/kipitidy/dropdown-menu";
import { Input } from "@kipit-labs/kipitidy/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@kipit-labs/kipitidy/select";

import type { Unit } from "@/lib/api";
import { useMe } from "@/lib/auth";
import { defaultRestS } from "@/lib/routines";
import { fieldsOf, setLabels, weightSign } from "@/lib/sets";
import { formatDuration, formatKm, formatWeight, parseDuration, parseKm, parseWeight } from "@/lib/units";
import { doneSets, hint, volumeOf, type Live as LiveState, type LiveExercise, type LiveSet, type LoggedSet } from "@/live/model";
import { unlockAudio } from "@/live/signal";
import { dispatch, loadPrevious, setLive, useLive } from "@/live/store";
import { useNow } from "@/live/useNow";

import { CellInput } from "./CellInput";
import { ExerciseThumb } from "./ExerciseMedia";
import { ExercisePicker } from "./ExercisePicker";
import { FinishSheet } from "./FinishSheet";
import { SetTypeMenu } from "./SetType";

const rpeChoices = [6, 6.5, 7, 7.5, 8, 8.5, 9, 9.5, 10];

/** The live workout: the screen used at the gym. */
export function Live() {
  const me = useMe();
  const live = useLive(me.id);
  // Browsers play sound only after a tap; the rest timer's beep needs one.
  useEffect(() => {
    addEventListener("pointerdown", unlockAudio, { once: true });
    return () => removeEventListener("pointerdown", unlockAudio);
  }, []);
  if (!live) return <Navigate to="/workout" replace />;
  return <Screen live={live} unit={me.unit} />;
}

function Screen({ live, unit }: { live: LiveState; unit: Unit }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const now = useNow();
  const [adding, setAdding] = useState(false);
  const [swapping, setSwapping] = useState<number | null>(null);
  const [finishing, setFinishing] = useState(false);
  const title = live.routine?.name ?? t("live.defaultTitle");
  const exs = live.exercises;
  return (
    <>
      <div className="sticky top-0 z-10 -mx-4 mb-4 border-b bg-background/95 px-4 py-2 backdrop-blur">
        <div className="flex items-center gap-2">
          <Button variant="ghost" size="icon-sm" className="-ml-2" aria-label={t("live.minimize")} onClick={() => navigate("/workout")}>
            <ChevronDown />
          </Button>
          <h1 className="min-w-0 flex-1 truncate font-heading font-semibold">{title}</h1>
          <Button size="sm" onClick={() => setFinishing(true)}>
            {t("live.finish")}
          </Button>
        </div>
        <dl className="mt-1 grid grid-cols-3 text-center text-sm">
          <Stat label={t("live.duration")} value={formatDuration(Math.max(0, Math.floor((now - live.startedAt) / 1000)))} />
          <Stat label={t("live.volume")} value={`${formatWeight(volumeOf(live), unit)} ${unit}`} />
          <Stat label={t("live.sets")} value={String(doneSets(live))} />
        </dl>
      </div>

      {exs.length === 0 && <p className="py-8 text-center text-sm text-muted-foreground">{t("live.empty")}</p>}
      <ol>
        {exs.map((e, i) => (
          <ExerciseCard
            key={e.key}
            live={live}
            i={i}
            unit={unit}
            joinedAbove={!!e.superset && exs[i - 1]?.superset === e.superset}
            joinedBelow={!!e.superset && exs[i + 1]?.superset === e.superset}
            onSwap={() => setSwapping(i)}
          />
        ))}
      </ol>
      <Button variant="outline" size="lg" className="mt-4 w-full" onClick={() => setAdding(true)}>
        <Plus data-icon="inline-start" />
        {t("routines.addExercises")}
      </Button>
      <ConfirmDialog
        trigger={
          <Button variant="ghost" className="mt-4 w-full text-destructive">
            <Trash2 data-icon="inline-start" />
            {t("live.discard")}
          </Button>
        }
        title={t("live.discardTitle")}
        description={t("live.discardDescription")}
        confirmLabel={t("live.discardConfirm")}
        cancelLabel={t("common.cancel")}
        destructive
        onConfirm={() => {
          setLive(null);
          navigate("/workout");
        }}
      />

      <ExercisePicker
        open={adding}
        onOpenChange={setAdding}
        onPick={(exercises) => {
          dispatch({ type: "addExercises", exercises });
          void loadPrevious(exercises.map((e) => e.id));
        }}
      />
      <ExercisePicker
        single
        open={swapping !== null}
        onOpenChange={(o) => !o && setSwapping(null)}
        onPick={([exercise]) => {
          dispatch({ type: "swapExercise", i: swapping!, exercise });
          void loadPrevious([exercise.id]);
        }}
      />
      <FinishSheet open={finishing} onOpenChange={setFinishing} live={live} title={title} unit={unit} />
    </>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="font-medium tabular-nums">{value}</dd>
    </div>
  );
}

function ExerciseCard({
  live,
  i,
  unit,
  joinedAbove,
  joinedBelow,
  onSwap,
}: {
  live: LiveState;
  i: number;
  unit: Unit;
  joinedAbove: boolean;
  joinedBelow: boolean;
  onSwap: () => void;
}) {
  const { t } = useTranslation();
  const e: LiveExercise = live.exercises[i];
  const kind = e.exercise.kind;
  const f = fieldsOf(kind);
  const labels = setLabels(
    e.sets.map((s) => s.type),
    { normal: "", warmup: t("sets.letters.warmup"), drop: t("sets.letters.drop"), failure: t("sets.letters.failure") },
  );
  const values = (f.weight ? 1 : 0) + (f.reps ? 1 : 0) + (f.duration ? 1 : 0) + (f.distance ? 1 : 0);
  const cols = cn("grid items-center gap-1.5", values === 2 ? "grid-cols-[2.25rem_1fr_4rem_4rem_2.25rem]" : "grid-cols-[2.25rem_1fr_5rem_2.25rem]");
  const last = i === live.exercises.length - 1;
  const restS = e.restS ?? defaultRestS;
  return (
    <li
      className={cn(
        "rounded-xl border bg-card p-3",
        e.superset && "border-l-4 border-l-primary",
        joinedAbove ? "rounded-t-none [border-top-style:dashed]" : "mt-3",
        joinedBelow && "rounded-b-none border-b-0",
      )}
    >
      <div className="flex items-center gap-2">
        <Link to={`/exercises/${e.exercise.id}`} aria-label={e.exercise.name} className="shrink-0 rounded-lg">
          <ExerciseThumb exercise={e.exercise} />
        </Link>
        <div className="min-w-0 flex-1">
          {e.superset && !joinedAbove && (
            <p className="text-xs font-semibold text-primary">{t("routines.superset", { letter: e.superset })}</p>
          )}
          <p className="truncate font-medium">{e.exercise.name}</p>
          {!joinedBelow && (
            <p className="text-xs text-muted-foreground">
              {restS ? t("live.restAfter", { time: formatDuration(restS) }) : t("live.noRest")}
            </p>
          )}
        </div>
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon-sm" aria-label={t("routines.exerciseMenu", { name: e.exercise.name })}>
              <MoreVertical />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem disabled={i === 0} onSelect={() => dispatch({ type: "moveExercise", i, by: -1 })}>
              <ArrowUp />
              {t("routines.moveUp")}
            </DropdownMenuItem>
            <DropdownMenuItem disabled={last} onSelect={() => dispatch({ type: "moveExercise", i, by: 1 })}>
              <ArrowDown />
              {t("routines.moveDown")}
            </DropdownMenuItem>
            <DropdownMenuItem disabled={!e.superset && last} onSelect={() => dispatch({ type: "superset", i })}>
              {e.superset ? <Link2Off /> : <Link2 />}
              {t(e.superset ? "routines.leaveSuperset" : "routines.supersetWithNext")}
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={onSwap}>
              <Replace />
              {t("routines.replaceExercise")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onSelect={() => dispatch({ type: "removeExercise", i })}>
              <Trash2 />
              {t("routines.removeExercise")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <Input
        className="mt-2 h-8 text-sm"
        placeholder={t("routines.exerciseNotes")}
        aria-label={t("routines.exerciseNotes")}
        maxLength={1000}
        value={e.notes}
        onChange={(ev) => dispatch({ type: "exerciseNotes", i, notes: ev.target.value })}
      />
      <div className="mt-3 space-y-1" role="table" aria-label={e.exercise.name}>
        <div role="row" className={cn(cols, "text-center text-xs font-medium text-muted-foreground uppercase")}>
          <span role="columnheader">{t("sets.set")}</span>
          <span role="columnheader">{t("live.previous")}</span>
          {f.weight && <span role="columnheader">{weightSign(kind) + unit.toUpperCase()}</span>}
          {f.distance && <span role="columnheader">{t("sets.km")}</span>}
          {f.reps && <span role="columnheader">{t("sets.reps")}</span>}
          {f.duration && <span role="columnheader">{t("sets.time")}</span>}
          <span role="columnheader" className="sr-only">
            {t("live.done")}
          </span>
        </div>
        {e.sets.map((s, j) => {
          const h = hint(live, i, j);
          const prev = live.previous[e.exercise.id]?.[j];
          const set = (patch: Partial<LiveSet>) => dispatch({ type: "updateSet", i, j, patch });
          return (
            // A tick fills fields from their hints: remount so each rereads its value.
            <div key={`${e.sets.length}-${j}-${s.done}`} className={cn("-mx-1 rounded-md px-1 py-0.5", s.done && "bg-primary/10")}>
              <div role="row" className={cols}>
                <SetTypeMenu type={s.type} label={labels[j]} onType={(type) => set({ type })}>
                  <DropdownMenuItem onSelect={() => set({ rpe: s.rpe === undefined ? null : undefined })}>
                    <Gauge />
                    {t(s.rpe === undefined ? "live.addRpe" : "live.removeRpe")}
                  </DropdownMenuItem>
                  <DropdownMenuItem variant="destructive" onSelect={() => dispatch({ type: "removeSet", i, j })}>
                    <Trash2 />
                    {t("sets.remove", { set: labels[j] })}
                  </DropdownMenuItem>
                </SetTypeMenu>
                <span className="truncate text-center text-sm text-muted-foreground tabular-nums">
                  {prev ? previousText(prev, kind, unit) : "—"}
                </span>
                {f.weight && (
                  <CellInput
                    inputMode="decimal"
                    aria-label={t("sets.weightOf", { set: labels[j] })}
                    placeholder={h.weightG == null ? "" : formatWeight(h.weightG, unit)}
                    value={s.weightG == null ? "" : formatWeight(s.weightG, unit)}
                    parse={(v) => parseWeight(v, unit)}
                    onValue={(weightG) => set({ weightG })}
                  />
                )}
                {f.distance && (
                  <CellInput
                    inputMode="decimal"
                    aria-label={t("sets.distanceOf", { set: labels[j] })}
                    placeholder={h.distanceM == null ? "" : formatKm(h.distanceM)}
                    value={s.distanceM == null ? "" : formatKm(s.distanceM)}
                    parse={parseKm}
                    onValue={(distanceM) => set({ distanceM })}
                  />
                )}
                {f.reps && (
                  <CellInput
                    inputMode="numeric"
                    aria-label={t("sets.repsOf", { set: labels[j] })}
                    placeholder={h.reps == null ? "" : String(h.reps)}
                    value={s.reps == null ? "" : String(s.reps)}
                    parse={parseCount}
                    onValue={(reps) => set({ reps })}
                  />
                )}
                {f.duration && (
                  <CellInput
                    inputMode="numeric"
                    aria-label={t("sets.timeOf", { set: labels[j] })}
                    placeholder={h.durationS == null ? "" : formatDuration(h.durationS)}
                    value={s.durationS == null ? "" : formatDuration(s.durationS)}
                    parse={parseDuration}
                    onValue={(durationS) => set({ durationS })}
                  />
                )}
                <Button
                  variant={s.done ? "default" : "outline"}
                  size="icon-sm"
                  aria-pressed={s.done}
                  aria-label={t("live.tick", { set: labels[j] })}
                  onClick={() => dispatch({ type: "toggleSet", i, j, now: Date.now() })}
                >
                  <Check />
                </Button>
              </div>
              {s.rpe !== undefined && (
                <div className="mt-1 flex items-center justify-end gap-2 text-sm">
                  <span className="text-muted-foreground">{t("live.rpe")}</span>
                  <Select value={s.rpe == null ? "none" : String(s.rpe)} onValueChange={(v) => set({ rpe: v === "none" ? null : Number(v) })}>
                    <SelectTrigger size="sm" className="w-20" aria-label={t("live.rpeOf", { set: labels[j] })}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">—</SelectItem>
                      {rpeChoices.map((r) => (
                        <SelectItem key={r} value={String(r)}>
                          {r}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
            </div>
          );
        })}
      </div>
      <Button variant="secondary" size="sm" className="mt-2 w-full" disabled={e.sets.length >= 50} onClick={() => dispatch({ type: "addSet", i })}>
        <Plus data-icon="inline-start" />
        {t("sets.add")}
      </Button>
    </li>
  );
}

/** A whole number of reps; null when blank. */
const parseCount = (v: string) => (v.trim() === "" ? null : /^\d+$/.test(v.trim()) ? Number(v) : undefined);

function previousText(p: LoggedSet, kind: LiveExercise["exercise"]["kind"], unit: Unit): string {
  const f = fieldsOf(kind);
  if (f.distance) return [p.distanceM != null && `${formatKm(p.distanceM)} km`, p.durationS != null && formatDuration(p.durationS)].filter(Boolean).join(" · ");
  if (f.duration) return p.durationS == null ? "—" : formatDuration(p.durationS);
  const w = f.weight && p.weightG != null ? weightSign(kind) + formatWeight(p.weightG, unit) : "";
  return `${w}${w ? " " : ""}× ${p.reps ?? "—"}`;
}
