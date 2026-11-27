import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronLeft, Pencil, SearchX, Trash2 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation, useNavigate, useParams } from "react-router";
import { toast } from "sonner";
import { Button } from "@kipit-labs/kipitidy/button";
import { ConfirmDialog } from "@kipit-labs/kipitidy/confirm-dialog";
import { EmptyState } from "@kipit-labs/kipitidy/empty-state";
import { IconTile } from "@kipit-labs/kipitidy/icon-tile";
import { Loading } from "@kipit-labs/kipitidy/loading";
import { Section } from "@kipit-labs/kipitidy/section";

import { api, type Exercise, type ExerciseSummary } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { equipmentIcon, exercisesQuery } from "@/lib/exercises";

import { ExerciseForm } from "./ExerciseForm";
import { ExerciseFrames } from "./ExerciseMedia";
import { chip } from "./Exercises";
import { PageTitle } from "./Layout";

export function ExerciseDetail() {
  const { t } = useTranslation();
  const { id = "" } = useParams();
  const qc = useQueryClient();
  const navigate = useNavigate();
  // Back keeps the list's search; a direct visit has no history to go back to.
  const first = useLocation().key === "default";
  const ex = useQuery({
    queryKey: ["exercises", id],
    queryFn: () => api.exercise(id),
    // The list row shows right away; instructions arrive with the full exercise.
    placeholderData: () => qc.getQueryData<ExerciseSummary[]>(exercisesQuery.queryKey)?.find((e) => e.id === id),
  });
  return (
    <>
      <Button
        variant="ghost"
        size="sm"
        className="-ml-2 mb-2"
        onClick={() => (first ? navigate("/exercises", { replace: true }) : navigate(-1))}
      >
        <ChevronLeft data-icon="inline-start" />
        {t("exercises.back")}
      </Button>
      {ex.isPending ? (
        <Loading className="py-12" />
      ) : ex.isError ? (
        <EmptyState icon={SearchX} title={errorMessage(ex.error)} />
      ) : (
        <Details key={ex.data.id} exercise={ex.data} />
      )}
    </>
  );
}

function Details({ exercise: e }: { exercise: Exercise }) {
  const { t } = useTranslation();
  const [noFrames, setNoFrames] = useState(!e.media?.length);
  const facts = [
    [t("exercises.fields.kind"), t(`exercises.kinds.${e.kind}`)],
    [t("exercises.fields.category"), t(`exercises.categories.${e.category}`)],
    [t("exercises.fields.equipment"), t(`exercises.equipment.${e.equipment}`)],
    ...(e.level ? [[t("exercises.fields.level"), t(`exercises.levels.${e.level}`)]] : []),
  ];
  return (
    <div className="space-y-4">
      <ExerciseFrames exercise={e} onFail={() => setNoFrames(true)} />
      <div className="flex items-start gap-3">
        {noFrames && <IconTile icon={equipmentIcon(e.equipment)} className="mt-0.5" />}
        <div className="min-w-0 flex-1">
          <PageTitle className="mb-1">{e.name}</PageTitle>
          {e.custom && <span className={`${chip} bg-primary/10 text-primary`}>{t("exercises.custom")}</span>}
        </div>
      </div>
      {e.custom && <Actions exercise={e} />}
      <Section>
        <dl className="grid grid-cols-2 gap-x-4 gap-y-3">
          {facts.map(([label, value]) => (
            <div key={label}>
              <dt className="text-xs text-muted-foreground">{label}</dt>
              <dd className="text-sm font-medium">{value}</dd>
            </div>
          ))}
        </dl>
        <Muscles label={t("exercises.fields.primaryMuscles")} muscles={e.primaryMuscles} />
        {e.secondaryMuscles.length > 0 && (
          <Muscles label={t("exercises.fields.secondaryMuscles")} muscles={e.secondaryMuscles} />
        )}
      </Section>
      {!!e.instructions?.length && (
        <Section title={t("exercises.fields.instructions")}>
          <ol className="space-y-3">
            {e.instructions.map((step, i) => (
              <li key={i} className="flex gap-3 text-sm">
                <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-medium">
                  {i + 1}
                </span>
                <span className="pt-0.5">{step}</span>
              </li>
            ))}
          </ol>
        </Section>
      )}
    </div>
  );
}

function Muscles({ label, muscles }: { label: string; muscles: string[] }) {
  const { t } = useTranslation();
  return (
    <div className="space-y-1.5">
      <p className="text-xs text-muted-foreground">{label}</p>
      <div className="flex flex-wrap gap-1">
        {muscles.map((m) => (
          <span key={m} className={chip}>
            {t(`exercises.muscles.${m}`)}
          </span>
        ))}
      </div>
    </div>
  );
}

function Actions({ exercise: e }: { exercise: Exercise }) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const navigate = useNavigate();
  const remove = useMutation({
    meta: { error: t("exercises.deleteFailed") },
    mutationFn: () => api.deleteExercise(e.id),
    onSuccess: () => {
      navigate("/exercises", { replace: true });
      qc.removeQueries({ queryKey: ["exercises", e.id] });
      void qc.invalidateQueries({ queryKey: ["exercises"], exact: true });
      toast.success(t("exercises.deleted"));
    },
  });
  return (
    <div className="grid grid-cols-2 gap-2">
      <ExerciseForm
        exercise={e}
        trigger={
          <Button variant="outline">
            <Pencil data-icon="inline-start" />
            {t("exercises.edit")}
          </Button>
        }
      />
      <ConfirmDialog
        trigger={
          <Button variant="outline" loading={remove.isPending}>
            <Trash2 data-icon="inline-start" />
            {t("exercises.delete")}
          </Button>
        }
        title={t("exercises.deleteTitle", { name: e.name })}
        description={t("exercises.deleteDescription")}
        confirmLabel={t("exercises.deleteConfirm")}
        cancelLabel={t("common.cancel")}
        destructive
        onConfirm={() => remove.mutate()}
      />
    </div>
  );
}
