import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronLeft, History as HistoryIcon, Pencil, SearchX, Trash2, Trophy } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link, useLocation, useNavigate, useParams } from "react-router";
import { toast } from "sonner";
import { Button } from "@kipit-labs/kipitidy/button";
import { ConfirmDialog } from "@kipit-labs/kipitidy/confirm-dialog";
import { EmptyState } from "@kipit-labs/kipitidy/empty-state";
import { Loading } from "@kipit-labs/kipitidy/loading";
import { Section } from "@kipit-labs/kipitidy/section";

import { api, type LoggedWorkout, type Unit } from "@/lib/api";
import { useMe } from "@/lib/auth";
import { errorMessage } from "@/lib/errors";
import { setLabels, setText } from "@/lib/sets";
import { formatDuration, formatWeight } from "@/lib/units";
import { currentLanguage } from "@/i18n";

import { ExerciseThumb } from "./ExerciseMedia";
import { PageTitle } from "./Layout";
import { RecordBadge, RecordMark } from "./RecordBadge";
import { useStartWorkout } from "./StartWorkout";

const when = (iso: string) =>
  new Date(iso).toLocaleString(currentLanguage(), { weekday: "short", day: "numeric", month: "short", year: "numeric", hour: "numeric", minute: "2-digit" });
const durationOf = (w: LoggedWorkout) => formatDuration(Math.round((Date.parse(w.endedAt) - Date.parse(w.startedAt)) / 1000));
const setCount = (w: LoggedWorkout) => w.exercises.reduce((n, e) => n + e.sets.length, 0);

function Back({ to, label }: { to: string; label: string }) {
  const navigate = useNavigate();
  // Back keeps where the list was; a direct visit has no history to go back to.
  const first = useLocation().key === "default";
  return (
    <Button variant="ghost" size="sm" className="-ml-2 mb-2" onClick={() => (first ? navigate(to, { replace: true }) : navigate(-1))}>
      <ChevronLeft data-icon="inline-start" />
      {label}
    </Button>
  );
}

/** Past workouts, newest first, a page at a time. */
export function History() {
  const { t } = useTranslation();
  const { unit } = useMe();
  const list = useInfiniteQuery({
    queryKey: ["workouts"],
    queryFn: ({ pageParam }) => api.workouts(pageParam),
    initialPageParam: "",
    getNextPageParam: (p) => p.next,
  });
  const workouts = list.data?.pages.flatMap((p) => p.workouts) ?? [];
  return (
    <>
      <Back to="/workout" label={t("nav.workout")} />
      <PageTitle>{t("history.title")}</PageTitle>
      {list.isPending ? (
        <Loading className="py-12" />
      ) : list.isError ? (
        <EmptyState icon={SearchX} title={errorMessage(list.error)} action={<Button onClick={() => list.refetch()}>{t("common.retry")}</Button>} />
      ) : workouts.length === 0 ? (
        <EmptyState icon={HistoryIcon} title={t("history.emptyTitle")} description={t("history.emptyDescription")} />
      ) : (
        <ul className="space-y-2">
          {workouts.map((w) => (
            <li key={w.id}>
              <WorkoutCard workout={w} unit={unit} />
            </li>
          ))}
        </ul>
      )}
      {list.hasNextPage && (
        <Button variant="outline" className="mt-4 w-full" loading={list.isFetchingNextPage} onClick={() => list.fetchNextPage()}>
          {t("history.more")}
        </Button>
      )}
    </>
  );
}

function WorkoutCard({ workout: w, unit }: { workout: LoggedWorkout; unit: Unit }) {
  const { t } = useTranslation();
  return (
    <Link to={`/workouts/${w.id}`} className="block rounded-xl border bg-card p-4 transition-colors hover:bg-muted">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate font-medium">{w.title}</p>
          <p className="text-xs text-muted-foreground">{when(w.startedAt)}</p>
        </div>
        {w.records.length > 0 && (
          <span className="inline-flex shrink-0 items-center gap-1 rounded-full bg-record px-2 py-0.5 text-xs font-medium text-record-foreground">
            <Trophy className="size-3" aria-hidden />
            {t("history.records", { count: w.records.length })}
          </span>
        )}
      </div>
      <Facts workout={w} unit={unit} />
      <ul className="mt-2 space-y-0.5 text-sm text-muted-foreground">
        {w.exercises.map((e, i) => (
          <li key={i} className="truncate">
            {t("history.exerciseSets", { count: e.sets.length, name: e.exercise?.name ?? "" })}
          </li>
        ))}
      </ul>
    </Link>
  );
}

function Facts({ workout: w, unit }: { workout: LoggedWorkout; unit: Unit }) {
  const { t } = useTranslation();
  const facts = [
    [t("live.duration"), durationOf(w)],
    [t("live.volume"), `${formatWeight(w.volumeG, unit)} ${unit}`],
    [t("live.sets"), String(setCount(w))],
  ];
  return (
    <dl className="mt-3 grid grid-cols-3 gap-2 text-sm">
      {facts.map(([label, value]) => (
        <div key={label}>
          <dt className="text-xs text-muted-foreground">{label}</dt>
          <dd className="font-medium tabular-nums">{value}</dd>
        </div>
      ))}
    </dl>
  );
}

/** One past workout: its sets and records, to edit or delete. */
export function WorkoutDetail() {
  const { t } = useTranslation();
  const { id = "" } = useParams();
  const { unit } = useMe();
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [begin, startDialog] = useStartWorkout();
  const w = useQuery({ queryKey: ["workouts", id], queryFn: () => api.workout(id) });
  const remove = useMutation({
    meta: { error: t("history.deleteFailed") },
    mutationFn: () => api.deleteWorkout(id),
    onSuccess: () => {
      navigate("/history", { replace: true });
      // Later workouts' records may change; this one is gone: don't fetch it again while its page leaves.
      void qc.invalidateQueries({ queryKey: ["workouts"], predicate: (q) => q.queryKey[1] !== id });
      void qc.invalidateQueries({ queryKey: ["progress"] });
      void qc.invalidateQueries({ queryKey: ["routines"] });
      toast.success(t("history.deleted"));
    },
  });
  if (w.isPending) return <Loading className="py-12" />;
  if (w.isError)
    return (
      <>
        <Back to="/history" label={t("history.title")} />
        <EmptyState icon={SearchX} title={errorMessage(w.error)} />
      </>
    );
  const d = w.data;
  const letters = { normal: "", warmup: t("sets.letters.warmup"), drop: t("sets.letters.drop"), failure: t("sets.letters.failure") };
  return (
    <div className="space-y-4">
      <div>
        <Back to="/history" label={t("history.title")} />
        <PageTitle className="mb-1">{d.title}</PageTitle>
        <p className="text-sm text-muted-foreground">{when(d.startedAt)}</p>
        <Facts workout={d} unit={unit} />
      </div>
      <div className="grid grid-cols-2 gap-2">
        <Button variant="outline" onClick={() => begin(undefined, d)}>
          <Pencil data-icon="inline-start" />
          {t("history.edit")}
        </Button>
        <ConfirmDialog
          trigger={
            <Button variant="outline" loading={remove.isPending}>
              <Trash2 data-icon="inline-start" />
              {t("history.delete")}
            </Button>
          }
          title={t("history.deleteTitle", { title: d.title })}
          description={t("history.deleteDescription")}
          confirmLabel={t("history.deleteConfirm")}
          cancelLabel={t("common.cancel")}
          destructive
          onConfirm={() => remove.mutate()}
        />
      </div>
      {startDialog}
      {d.notes && <p className="text-sm whitespace-pre-line">{d.notes}</p>}
      {d.records.length > 0 && (
        <Section title={t("history.records", { count: d.records.length })}>
          <ul className="space-y-1.5">
            {d.records.map((r) => (
              <li key={`${r.exercise}-${r.kind}`} className="flex items-center justify-between gap-2 text-sm">
                <span className="min-w-0 truncate">{d.exercises[r.exercise]?.exercise?.name}</span>
                <RecordBadge kind={r.kind} value={r.value} unit={unit} />
              </li>
            ))}
          </ul>
        </Section>
      )}
      {d.exercises.map((e, i) => {
        const labels = setLabels(
          e.sets.map((s) => s.type),
          letters,
        );
        return (
          <Section key={i}>
            {e.exercise && (
              <Link to={`/exercises/${e.exerciseId}`} className="flex items-center gap-3 rounded-lg">
                <ExerciseThumb exercise={e.exercise} />
                <span className="min-w-0 flex-1 truncate font-medium">{e.exercise.name}</span>
              </Link>
            )}
            {e.notes && <p className="text-sm text-muted-foreground">{e.notes}</p>}
            <ol className="space-y-1 text-sm tabular-nums">
              {e.sets.map((s, j) => (
                <li key={j} className="flex gap-3">
                  <span className="w-5 text-center text-muted-foreground">{labels[j]}</span>
                  <span>{e.exercise ? setText(s, e.exercise.kind, unit, true) : ""}</span>
                  {s.rpe != null && <span className="text-muted-foreground">@{s.rpe}</span>}
                  {d.records.some((r) => r.exercise === i && r.set === j) && <RecordMark />}
                </li>
              ))}
            </ol>
          </Section>
        );
      })}
    </div>
  );
}
