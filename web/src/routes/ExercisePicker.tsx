import { useQuery } from "@tanstack/react-query";
import { Check } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { cn } from "cn";
import { Button } from "@kipit-labs/kipitidy/button";
import { IconTile } from "@kipit-labs/kipitidy/icon-tile";
import { Loading } from "@kipit-labs/kipitidy/loading";
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@kipit-labs/kipitidy/sheet";

import type { ExerciseSummary } from "@/lib/api";
import { equipment, equipmentIcon, exercisesQuery, filterExercises, muscles } from "@/lib/exercises";

import { Filter, Paged, SearchField } from "./Exercises";

/**
 * Picks exercises from the library in a bottom sheet: several at once, or one
 * (a tap picks it) with single.
 */
export function ExercisePicker({
  open,
  onOpenChange,
  single,
  onPick,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  single?: boolean;
  onPick: (exercises: ExerciseSummary[]) => void;
}) {
  const { t } = useTranslation();
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="bottom" className="flex gap-0 rounded-t-2xl data-[side=bottom]:h-[92svh]">
        <SheetHeader>
          <SheetTitle>{t(single ? "routines.replaceExercise" : "routines.addExercises")}</SheetTitle>
          <SheetDescription className="sr-only">{t("workout.libraryDescription")}</SheetDescription>
        </SheetHeader>
        {/* Mounted only while open, so each opening starts with nothing picked. */}
        {open && (
          <Picker
            single={single}
            onPick={(exs) => {
              onOpenChange(false);
              onPick(exs);
            }}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}

function Picker({ single, onPick }: { single?: boolean; onPick: (exercises: ExerciseSummary[]) => void }) {
  const { t } = useTranslation();
  const list = useQuery(exercisesQuery);
  const [q, setQ] = useState("");
  const [muscle, setMuscle] = useState("");
  const [equip, setEquip] = useState("");
  const [picked, setPicked] = useState<ExerciseSummary[]>([]);
  const shown = useMemo(
    () => filterExercises(list.data ?? [], { q, muscle, equipment: equip, category: "" }, t),
    [list.data, q, muscle, equip, t],
  );
  const toggle = (e: ExerciseSummary) => {
    if (single) return onPick([e]);
    setPicked((p) => (p.some((x) => x.id === e.id) ? p.filter((x) => x.id !== e.id) : [...p, e]));
  };
  const all = (v: string) => (v === "all" ? "" : v);
  return (
    <>
      <div className="space-y-2 px-4 pb-3">
        <SearchField value={q} onChange={setQ} />
        <div className="grid grid-cols-2 gap-2">
          <Filter value={muscle} onChange={(v) => setMuscle(all(v))} label={t("exercises.fields.muscle")} allLabel={t("exercises.allMuscles")}>
            {muscles.map((m) => ({ value: m, label: t(`exercises.muscles.${m}`) }))}
          </Filter>
          <Filter value={equip} onChange={(v) => setEquip(all(v))} label={t("exercises.fields.equipment")} allLabel={t("exercises.allEquipment")}>
            {equipment.map((e) => ({ value: e, label: t(`exercises.equipment.${e}`) }))}
          </Filter>
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto border-t px-4">
        {list.isPending ? (
          <Loading className="py-12" />
        ) : shown.length === 0 ? (
          <p className="py-12 text-center text-sm text-muted-foreground">{t("exercises.noResults")}</p>
        ) : (
          <Paged key={[q, muscle, equip].join("|")} items={shown}>
            {(e) => {
              const on = picked.some((x) => x.id === e.id);
              return (
                <li key={e.id}>
                  <button
                    type="button"
                    aria-pressed={single ? undefined : on}
                    onClick={() => toggle(e)}
                    className="flex w-full items-center gap-3 py-2.5 text-left"
                  >
                    <IconTile icon={equipmentIcon(e.equipment)} />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium">{e.name}</span>
                      <span className="block truncate text-xs text-muted-foreground">
                        {[...e.primaryMuscles.map((m) => t(`exercises.muscles.${m}`)), t(`exercises.equipment.${e.equipment}`)].join(" · ")}
                      </span>
                    </span>
                    {!single && (
                      <span
                        aria-hidden
                        className={cn(
                          "flex size-6 shrink-0 items-center justify-center rounded-full border",
                          on && "border-primary bg-primary text-primary-foreground",
                        )}
                      >
                        {on && <Check className="size-4" />}
                      </span>
                    )}
                  </button>
                </li>
              );
            }}
          </Paged>
        )}
      </div>
      {!single && (
        <SheetFooter className="border-t">
          <Button size="lg" disabled={picked.length === 0} onClick={() => onPick(picked)}>
            {t("routines.addCount", { count: picked.length })}
          </Button>
        </SheetFooter>
      )}
    </>
  );
}
