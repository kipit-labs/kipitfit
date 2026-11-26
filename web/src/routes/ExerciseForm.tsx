import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@kipit-labs/kipitidy/button";
import { CheckboxField } from "@kipit-labs/kipitidy/checkbox-field";
import { FloatingInput } from "@kipit-labs/kipitidy/floating-input";
import { FloatingSelect } from "@kipit-labs/kipitidy/floating-select";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@kipit-labs/kipitidy/sheet";

import { api, type Exercise, type ExerciseInput, type ExerciseKind } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { categories, equipment, kinds, muscles } from "@/lib/exercises";

const blank: Required<ExerciseInput> = {
  name: "",
  kind: "weight_reps",
  category: "strength",
  equipment: "barbell",
  primaryMuscles: ["chest"],
  secondaryMuscles: [],
};

/** Creates a custom exercise, or edits one when `exercise` is given, in a bottom sheet. */
export function ExerciseForm({
  exercise,
  trigger,
  onSaved,
}: {
  exercise?: Exercise;
  trigger: React.ReactNode;
  onSaved?: (e: Exercise) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>{trigger}</SheetTrigger>
      <SheetContent side="bottom" className="max-h-[90svh] overflow-y-auto rounded-t-2xl">
        <SheetHeader>
          <SheetTitle>{t(exercise ? "exercises.edit" : "exercises.new")}</SheetTitle>
          <SheetDescription className="sr-only">{t("workout.libraryDescription")}</SheetDescription>
        </SheetHeader>
        {/* Mounted only while open, so each opening starts from the saved values. */}
        {open && (
          <Fields
            exercise={exercise}
            onSaved={(e) => {
              setOpen(false);
              onSaved?.(e);
            }}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}

function Fields({ exercise, onSaved }: { exercise?: Exercise; onSaved: (e: Exercise) => void }) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const [v, setV] = useState<Required<ExerciseInput>>(exercise ? { ...blank, ...pick(exercise) } : blank);
  const save = useMutation({
    meta: { error: false },
    mutationFn: () => (exercise ? api.updateExercise(exercise.id, v) : api.createExercise(v)),
    onSuccess: (e) => {
      qc.setQueryData(["exercises", e.id], e);
      void qc.invalidateQueries({ queryKey: ["exercises"], exact: true });
      onSaved(e);
    },
  });
  const options = (group: string, values: string[]) =>
    values
      .map((value) => ({ value, label: t(`exercises.${group}.${value}`) }))
      .sort((a, b) => a.label.localeCompare(b.label));
  const primary = v.primaryMuscles[0];
  return (
    <form
      className="space-y-3 px-4"
      onSubmit={(e) => {
        e.preventDefault();
        save.mutate();
      }}
    >
      <FloatingInput
        label={t("exercises.fields.name")}
        required
        maxLength={100}
        autoComplete="off"
        value={v.name}
        onChange={(e) => setV({ ...v, name: e.target.value })}
      />
      <FloatingSelect
        label={t("exercises.fields.kind")}
        value={v.kind}
        onValueChange={(kind) => setV({ ...v, kind: kind as ExerciseKind })}
        options={kinds.map((k) => ({ value: k, label: t(`exercises.kinds.${k}`) }))}
      />
      <div className="grid grid-cols-2 gap-3">
        <FloatingSelect
          label={t("exercises.fields.equipment")}
          value={v.equipment}
          onValueChange={(equipment) => setV({ ...v, equipment })}
          options={options("equipment", equipment)}
        />
        <FloatingSelect
          label={t("exercises.fields.category")}
          value={v.category}
          onValueChange={(category) => setV({ ...v, category })}
          options={options("categories", categories)}
        />
      </div>
      <FloatingSelect
        label={t("exercises.fields.primaryMuscles")}
        value={primary}
        onValueChange={(m) =>
          setV({ ...v, primaryMuscles: [m], secondaryMuscles: v.secondaryMuscles.filter((s) => s !== m) })
        }
        options={options("muscles", muscles)}
      />
      <fieldset className="space-y-2">
        <legend className="mb-2 text-sm font-medium">{t("exercises.fields.secondaryMuscles")}</legend>
        <div className="grid grid-cols-2 gap-x-3 gap-y-2">
          {options("muscles", muscles)
            .filter((m) => !v.primaryMuscles.includes(m.value))
            .map((m) => (
              <CheckboxField
                key={m.value}
                label={m.label}
                checked={v.secondaryMuscles.includes(m.value)}
                onCheckedChange={(on) =>
                  setV({
                    ...v,
                    secondaryMuscles: on
                      ? [...v.secondaryMuscles, m.value]
                      : v.secondaryMuscles.filter((s) => s !== m.value),
                  })
                }
              />
            ))}
        </div>
      </fieldset>
      {save.isError && (
        <p role="alert" className="text-sm text-destructive">
          {errorMessage(save.error)}
        </p>
      )}
      <SheetFooter className="px-0">
        <Button type="submit" size="lg" loading={save.isPending}>
          {t(exercise ? "common.save" : "exercises.create")}
        </Button>
        <SheetClose asChild>
          <Button type="button" variant="outline" size="lg">
            {t("common.cancel")}
          </Button>
        </SheetClose>
      </SheetFooter>
    </form>
  );
}

const pick = (e: Exercise): Required<ExerciseInput> => ({
  name: e.name,
  kind: e.kind,
  category: e.category,
  equipment: e.equipment,
  primaryMuscles: e.primaryMuscles,
  secondaryMuscles: e.secondaryMuscles,
});
