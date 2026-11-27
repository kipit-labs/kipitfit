import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowDown, ArrowUp, ChevronLeft, Link2, Link2Off, MoreVertical, Plus, SearchX, Trash2, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams, useSearchParams } from "react-router";
import { toast } from "sonner";
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
import { EmptyState } from "@kipit-labs/kipitidy/empty-state";
import { FloatingInput } from "@kipit-labs/kipitidy/floating-input";
import { FloatingSelect } from "@kipit-labs/kipitidy/floating-select";
import { FloatingTextarea } from "@kipit-labs/kipitidy/floating-textarea";
import { Input } from "@kipit-labs/kipitidy/input";
import { Loading } from "@kipit-labs/kipitidy/loading";

import { api, type ExerciseSummary, type Folder, type Routine, type RoutineExercise, type RoutineSet } from "@/lib/api";
import { useMe } from "@/lib/auth";
import { errorMessage } from "@/lib/errors";
import { defaultRestS, leaveSuperset, move, relabel, routinesQuery, supersetWithNext } from "@/lib/routines";
import { fieldsOf, setLabels, weightSign } from "@/lib/sets";
import { formatDuration, formatKm, formatReps, formatWeight, parseDuration, parseKm, parseReps, parseWeight } from "@/lib/units";

import { CellInput } from "@/features/live/CellInput";
import { ExercisePicker } from "@/features/exercises/ExercisePicker";
import { SetTypeMenu } from "@/features/live/SetType";

type Item = RoutineExercise & { key: number; exercise: ExerciseSummary };
type Draft = { name: string; notes: string; folderId?: string; exercises: Item[] };

let nextKey = 0;
const item = (e: RoutineExercise & { exercise: ExerciseSummary }): Item => ({ ...e, key: nextKey++ });

const restChoices = [0, 30, 45, 60, 75, 90, 120, 150, 180, 240, 300];
const noFolder = "none";

/** Creates a routine (/routines/new, ?folder= to file it) or edits one (/routines/:id). */
export function RoutineEditor() {
  const { t } = useTranslation();
  const { id } = useParams();
  const list = useQuery(routinesQuery);
  if (list.isPending) return <Loading className="py-12" />;
  if (list.isError) return <EmptyState icon={SearchX} title={errorMessage(list.error)} />;
  const routine = id ? list.data.routines.find((r) => r.id === id) : undefined;
  if (id && !routine) return <EmptyState icon={SearchX} title={t("errors.not_found")} />;
  return <Editor key={id ?? "new"} routine={routine} folders={list.data.folders} />;
}

function Editor({ routine, folders }: { routine?: Routine; folders: Folder[] }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const [params] = useSearchParams();
  const [draft, setDraft] = useState<Draft>(() =>
    routine
      ? { ...routine, exercises: routine.exercises.map((e) => item({ ...e, exercise: e.exercise! })) }
      : { name: "", notes: "", folderId: params.get("folder") ?? undefined, exercises: [] },
  );
  const [picking, setPicking] = useState(false);
  const back = () => navigate("/workout");
  const done = () => {
    void qc.invalidateQueries({ queryKey: routinesQuery.queryKey });
    back();
  };
  const save = useMutation({
    meta: { error: t("routines.saveFailed") },
    mutationFn: () => {
      const body = {
        name: draft.name,
        notes: draft.notes,
        folderId: draft.folderId,
        exercises: draft.exercises.map(({ exerciseId, superset, restS, notes, sets }) => ({ exerciseId, superset, restS, notes, sets })),
      };
      return routine ? api.updateRoutine(routine.id, body) : api.createRoutine(body);
    },
    onSuccess: done,
  });
  const remove = useMutation({
    meta: { error: t("routines.deleteFailed") },
    mutationFn: () => api.deleteRoutine(routine!.id),
    onSuccess: () => {
      toast.success(t("routines.deleted"));
      done();
    },
  });
  const setExercises = (f: (l: Item[]) => Item[]) => setDraft((d) => ({ ...d, exercises: f(d.exercises) }));
  const change = (i: number, patch: Partial<Item>) =>
    setExercises((l) => l.map((e, j) => (j === i ? { ...e, ...patch } : e)));

  return (
    <>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          save.mutate();
        }}
      >
        <div className="sticky top-0 z-10 -mx-4 mb-4 flex items-center justify-between gap-2 bg-background/95 px-4 py-2 backdrop-blur">
          <Button type="button" variant="ghost" size="sm" className="-ml-2" onClick={back}>
            <ChevronLeft data-icon="inline-start" />
            {t("common.cancel")}
          </Button>
          <h1 className="truncate font-heading font-semibold">{t(routine ? "routines.edit" : "routines.new")}</h1>
          <Button type="submit" size="sm" loading={save.isPending}>
            {t("common.save")}
          </Button>
        </div>
        <div className="space-y-3">
          <FloatingInput
            label={t("routines.name")}
            required
            maxLength={100}
            autoComplete="off"
            value={draft.name}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
          />
          {folders.length > 0 && (
            <FloatingSelect
              label={t("routines.folder")}
              value={draft.folderId ?? noFolder}
              onValueChange={(v) => setDraft({ ...draft, folderId: v === noFolder ? undefined : v })}
              options={[{ value: noFolder, label: t("routines.noFolder") }, ...folders.map((f) => ({ value: f.id, label: f.name }))]}
            />
          )}
          <FloatingTextarea
            label={t("routines.notes")}
            maxLength={1000}
            value={draft.notes}
            onChange={(e) => setDraft({ ...draft, notes: e.target.value })}
          />
        </div>

        <ol className="mt-6">
          {draft.exercises.map((e, i) => (
            <ExerciseCard
              key={e.key}
              item={e}
              first={i === 0}
              last={i === draft.exercises.length - 1}
              joinedAbove={!!e.superset && draft.exercises[i - 1]?.superset === e.superset}
            joinedBelow={!!e.superset && draft.exercises[i + 1]?.superset === e.superset}
              onChange={(patch) => change(i, patch)}
              onMove={(by) => setExercises((l) => relabel(move(l, i, by)))}
              onSuperset={() => setExercises((l) => (e.superset ? leaveSuperset(l, i) : supersetWithNext(l, i)))}
              onRemove={() => setExercises((l) => relabel(l.filter((_, j) => j !== i)))}
            />
          ))}
        </ol>
        {draft.exercises.length === 0 && (
          <p className="py-8 text-center text-sm text-muted-foreground">{t("routines.noExercises")}</p>
        )}
        <Button type="button" variant="outline" size="lg" className="mt-3 w-full" onClick={() => setPicking(true)}>
          <Plus data-icon="inline-start" />
          {t("routines.addExercises")}
        </Button>
        <ExercisePicker
          open={picking}
          onOpenChange={setPicking}
          onPick={(exs) =>
            setExercises((l) => [
              ...l,
              ...exs.map((ex) =>
                item({ exerciseId: ex.id, exercise: ex, restS: null, notes: "", sets: [{ type: "normal" }] }),
              ),
            ])
          }
        />
      </form>
      {/* Outside the form: React bubbles the dialog's submit through its portal, which would save too. */}
      {routine && (
        <ConfirmDialog
          trigger={
            <Button type="button" variant="ghost" className="mt-6 w-full text-destructive" loading={remove.isPending}>
              <Trash2 data-icon="inline-start" />
              {t("routines.delete")}
            </Button>
          }
          title={t("routines.deleteTitle", { name: routine.name })}
          description={t("routines.deleteDescription")}
          confirmLabel={t("routines.deleteConfirm")}
          cancelLabel={t("common.cancel")}
          destructive
          onConfirm={() => remove.mutate()}
        />
      )}
    </>
  );
}

function ExerciseCard({
  item: e,
  first,
  last,
  joinedAbove,
  joinedBelow,
  onChange,
  onMove,
  onSuperset,
  onRemove,
}: {
  item: Item;
  first: boolean;
  last: boolean;
  joinedAbove: boolean;
  joinedBelow: boolean;
  onChange: (patch: Partial<Item>) => void;
  onMove: (by: -1 | 1) => void;
  onSuperset: () => void;
  onRemove: () => void;
}) {
  const { t } = useTranslation();
  const unit = useMe().unit;
  const f = fieldsOf(e.exercise.kind);
  const labels = setLabels(
    e.sets.map((s) => s.type),
    { normal: "", warmup: t("sets.letters.warmup"), drop: t("sets.letters.drop"), failure: t("sets.letters.failure") },
  );
  const setSet = (i: number, patch: Partial<RoutineSet>) =>
    onChange({ sets: e.sets.map((s, j) => (j === i ? { ...s, ...patch } : s)) });
  const cols = cn("grid items-center gap-2", f.weight || f.distance ? "grid-cols-[2.25rem_1fr_1fr_2rem]" : "grid-cols-[2.25rem_1fr_2rem]");
  return (
    <li
      className={cn(
        "rounded-xl border bg-card p-3",
        // A superset's cards hang together on a rail in the primary colour.
        e.superset && "border-l-4 border-l-primary",
        joinedAbove ? "rounded-t-none [border-top-style:dashed]" : "mt-3",
        joinedBelow && "rounded-b-none border-b-0",
      )}
    >
      <div className="flex items-center gap-2">
        <div className="min-w-0 flex-1">
          {e.superset && !joinedAbove && (
            <p className="text-xs font-semibold text-primary">{t("routines.superset", { letter: e.superset })}</p>
          )}
          <p className="truncate font-medium">
            {e.exercise.name}
            {e.exercise.archived && <span className="ml-2 text-xs font-normal text-muted-foreground">{t("exercises.archived")}</span>}
          </p>
        </div>
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="ghost" size="icon-sm" aria-label={t("routines.exerciseMenu", { name: e.exercise.name })}>
              <MoreVertical />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem disabled={first} onSelect={() => onMove(-1)}>
              <ArrowUp />
              {t("routines.moveUp")}
            </DropdownMenuItem>
            <DropdownMenuItem disabled={last} onSelect={() => onMove(1)}>
              <ArrowDown />
              {t("routines.moveDown")}
            </DropdownMenuItem>
            <DropdownMenuItem disabled={!e.superset && last} onSelect={onSuperset}>
              {e.superset ? <Link2Off /> : <Link2 />}
              {t(e.superset ? "routines.leaveSuperset" : "routines.supersetWithNext")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onSelect={onRemove}>
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
        onChange={(ev) => onChange({ notes: ev.target.value })}
      />
      <FloatingSelect
        className="mt-2"
        label={t("routines.rest")}
        value={e.restS == null ? "default" : String(e.restS)}
        onValueChange={(v) => onChange({ restS: v === "default" ? null : Number(v) })}
        options={[
          { value: "default", label: t("routines.restDefault", { time: formatDuration(defaultRestS) }) },
          ...restChoices.map((s) => ({ value: String(s), label: s === 0 ? t("routines.restOff") : formatDuration(s) })),
        ]}
      />
      <div className="mt-3 space-y-1.5" role="table" aria-label={t("routines.targetSets")}>
        <div role="row" className={cn(cols, "text-center text-xs font-medium text-muted-foreground uppercase")}>
          <span role="columnheader">{t("sets.set")}</span>
          {f.weight && <span role="columnheader">{weightSign(e.exercise.kind) + unit.toUpperCase()}</span>}
          {f.distance && <span role="columnheader">{t("sets.km")}</span>}
          {f.reps && <span role="columnheader">{t("sets.reps")}</span>}
          {f.duration && <span role="columnheader">{t("sets.time")}</span>}
          <span />
        </div>
        {e.sets.map((s, i) => (
          // Adding or removing a set remounts the rows, so each field rereads its value.
          <div role="row" key={`${e.sets.length}-${i}`} className={cols}>
            <SetTypeMenu type={s.type} label={labels[i]} onType={(type) => setSet(i, { type })} />
            {f.weight && (
              <CellInput
                inputMode="decimal"
                aria-label={t("sets.weightOf", { set: labels[i] })}
                placeholder="—"
                value={s.weightG == null ? "" : formatWeight(s.weightG, unit, true)}
                parse={(v) => parseWeight(v, unit)}
                onValue={(weightG) => setSet(i, { weightG })}
              />
            )}
            {f.distance && (
              <CellInput
                inputMode="decimal"
                aria-label={t("sets.distanceOf", { set: labels[i] })}
                placeholder="—"
                value={s.distanceM == null ? "" : formatKm(s.distanceM, true)}
                parse={parseKm}
                onValue={(distanceM) => setSet(i, { distanceM })}
              />
            )}
            {f.reps && (
              <CellInput
                inputMode="text"
                aria-label={t("sets.repsOf", { set: labels[i] })}
                placeholder="8–12"
                value={formatReps(s.repsMin, s.repsMax)}
                parse={parseReps}
                onValue={(r) => setSet(i, { repsMin: r.min, repsMax: r.max })}
              />
            )}
            {f.duration && (
              <CellInput
                inputMode="numeric"
                aria-label={t("sets.timeOf", { set: labels[i] })}
                placeholder="1:00"
                value={s.durationS == null ? "" : formatDuration(s.durationS)}
                parse={parseDuration}
                onValue={(durationS) => setSet(i, { durationS })}
              />
            )}
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label={t("sets.remove", { set: labels[i] })}
              onClick={() => onChange({ sets: e.sets.filter((_, j) => j !== i) })}
            >
              <X />
            </Button>
          </div>
        ))}
      </div>
      <Button
        type="button"
        variant="secondary"
        size="sm"
        className="mt-2 w-full"
        disabled={e.sets.length >= 50}
        // A new set starts as a copy of the last one.
        onClick={() => onChange({ sets: [...e.sets, { ...(e.sets.at(-1) ?? { type: "normal" }) }] })}
      >
        <Plus data-icon="inline-start" />
        {t("sets.add")}
      </Button>
    </li>
  );
}
