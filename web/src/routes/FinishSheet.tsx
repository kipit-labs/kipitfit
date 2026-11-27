import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Button } from "@kipit-labs/kipitidy/button";
import { FloatingInput } from "@kipit-labs/kipitidy/floating-input";
import { FloatingTextarea } from "@kipit-labs/kipitidy/floating-textarea";
import { Segmented, SegmentedItem } from "@kipit-labs/kipitidy/segmented";
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@kipit-labs/kipitidy/sheet";

import type { Unit } from "@/lib/api";
import { formatDuration, formatWeight } from "@/lib/units";
import { doneSets, recordsOf, toWorkout, updatedRoutine, volumeOf, type Live, type RoutineUpdate } from "@/live/model";
import { finish } from "@/live/store";

import { RecordBadge } from "./RecordBadge";

/** Ends the live workout: summary, title, notes, what to do with the routine, then save. */
export function FinishSheet({
  open,
  onOpenChange,
  live,
  title,
  unit,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  live: Live;
  title: string;
  unit: Unit;
}) {
  const { t } = useTranslation();
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="bottom" className="max-h-[90svh] overflow-y-auto rounded-t-2xl">
        <SheetHeader>
          <SheetTitle>{t("live.finishTitle")}</SheetTitle>
          <SheetDescription className="sr-only">{t("live.finishTitle")}</SheetDescription>
        </SheetHeader>
        {/* Mounted while open, so the duration is taken when the sheet opens. */}
        {open && <Form live={live} title={title} unit={unit} />}
      </SheetContent>
    </Sheet>
  );
}

function Form({ live, title: initial, unit }: { live: Live; title: string; unit: Unit }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [endedAt] = useState(Date.now);
  const [title, setTitle] = useState(initial);
  const [notes, setNotes] = useState("");
  const [update, setUpdate] = useState<RoutineUpdate>("keep");
  const sets = doneSets(live);
  const records = live.exercises.flatMap((e, i) => recordsOf(live, i).map((r) => ({ key: `${e.key}-${r.kind}`, name: e.exercise.name, r })));
  const save = () => {
    const body = updatedRoutine(live, update);
    finish(
      {
        id: live.id,
        userId: live.userId,
        workout: toWorkout(live, title.trim() || initial, notes, endedAt),
        routine: body && live.routine ? { id: live.routine.id, body } : undefined,
      },
      live.userId,
    );
    navigate("/workout");
  };
  return (
    <form
      className="space-y-4 px-4"
      onSubmit={(e) => {
        e.preventDefault();
        save();
      }}
    >
      <dl className="grid grid-cols-3 gap-2 rounded-xl bg-muted p-3 text-center">
        <Fact label={t("live.duration")} value={formatDuration(Math.floor((endedAt - live.startedAt) / 1000))} />
        <Fact label={t("live.volume")} value={`${formatWeight(volumeOf(live), unit)} ${unit}`} />
        <Fact label={t("live.sets")} value={String(sets)} />
      </dl>
      {records.length > 0 && (
        <section className="space-y-2" aria-label={t("live.newRecords", { count: records.length })}>
          <h3 className="text-sm font-medium">{t("live.newRecords", { count: records.length })}</h3>
          <ul className="space-y-1.5">
            {records.map(({ key, name, r }) => (
              <li key={key} className="flex items-center justify-between gap-2 text-sm">
                <span className="min-w-0 truncate">{name}</span>
                <RecordBadge kind={r.kind} value={r.value} unit={unit} />
              </li>
            ))}
          </ul>
        </section>
      )}
      <FloatingInput label={t("live.title")} required maxLength={100} value={title} onChange={(e) => setTitle(e.target.value)} />
      <FloatingTextarea label={t("routines.notes")} maxLength={1000} value={notes} onChange={(e) => setNotes(e.target.value)} />
      {live.routine && (
        <fieldset className="space-y-2">
          <legend className="mb-2 text-sm font-medium">{t("live.updateRoutine", { name: live.routine.name })}</legend>
          <Segmented value={update} onValueChange={(v) => v && setUpdate(v as RoutineUpdate)} className="w-full">
            <SegmentedItem value="keep" className="flex-1">{t("live.updateKeep")}</SegmentedItem>
            <SegmentedItem value="weights" className="flex-1">{t("live.updateWeights")}</SegmentedItem>
            <SegmentedItem value="sets" className="flex-1">{t("live.updateSets")}</SegmentedItem>
          </Segmented>
          <p className="text-sm text-muted-foreground">{t(`live.update_${update}`)}</p>
        </fieldset>
      )}
      {sets === 0 ? (
        <p role="alert" className="text-sm text-destructive">
          {t("errors.workout_empty")}
        </p>
      ) : (
        <p className="text-sm text-muted-foreground">{t("live.untickedLeftOut")}</p>
      )}
      <SheetFooter className="px-0">
        <Button type="submit" size="lg" disabled={sets === 0}>
          {t("live.save")}
        </Button>
      </SheetFooter>
    </form>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="font-semibold tabular-nums">{value}</dd>
    </div>
  );
}
