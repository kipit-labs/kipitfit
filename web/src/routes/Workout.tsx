import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowDown,
  ArrowUp,
  ChevronDown,
  ChevronRight,
  ClipboardList,
  CloudAlert,
  CloudOff,
  FolderPlus,
  History,
  Library,
  MoreVertical,
  Pencil,
  Play,
  Plus,
  SearchX,
  Trash2,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { toast } from "sonner";
import { Button } from "@kipit-labs/kipitidy/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@kipit-labs/kipitidy/collapsible";
import { ConfirmDialog } from "@kipit-labs/kipitidy/confirm-dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@kipit-labs/kipitidy/dropdown-menu";
import { EmptyState } from "@kipit-labs/kipitidy/empty-state";
import { IconTile } from "@kipit-labs/kipitidy/icon-tile";
import { Loading } from "@kipit-labs/kipitidy/loading";
import { Spinner } from "@kipit-labs/kipitidy/spinner";

import { api, ApiError, type Folder, type Routine } from "@/lib/api";
import { useMe } from "@/lib/auth";
import { errorMessage } from "@/lib/errors";
import { move, routinesQuery } from "@/lib/routines";
import { discardFinish, retryFinish, useOutbox } from "@/live/store";

import { PageTitle } from "./Layout";
import { NameDialog } from "./NameDialog";
import { useStartWorkout } from "./StartWorkout";

type Data = { folders: Folder[]; routines: Routine[] };
type Start = (routine?: Routine) => void;

/** The Workout tab: start a workout, the routines in their folders, the library. */
export function Workout() {
  const { t } = useTranslation();
  const list = useQuery(routinesQuery);
  const [naming, setNaming] = useState(false);
  const order = useOrder();
  const [begin, startDialog] = useStartWorkout();
  const createFolder = useMutation({
    meta: { error: t("routines.folderSaveFailed") },
    mutationFn: api.createFolder,
    onSuccess: () => {
      setNaming(false);
      void order.qc.invalidateQueries({ queryKey: routinesQuery.queryKey });
    },
  });
  const data = list.data;
  const loose = data?.routines.filter((r) => !r.folderId) ?? [];
  return (
    <>
      <PageTitle>{t("nav.workout")}</PageTitle>
      <Outbox />
      <Button size="lg" className="w-full" onClick={() => begin()}>
        <Play data-icon="inline-start" />
        {t("workout.startEmpty")}
      </Button>
      {startDialog}

      <div className="mt-8 mb-3 flex items-center justify-between gap-2">
        <h2 className="font-heading text-lg font-semibold">{t("routines.title")}</h2>
        <div className="flex gap-1">
          <Button variant="ghost" size="icon-sm" aria-label={t("routines.newFolder")} onClick={() => setNaming(true)}>
            <FolderPlus />
          </Button>
          <Button size="sm" variant="outline" asChild>
            <Link to="/routines/new">
              <Plus data-icon="inline-start" />
              {t("routines.new")}
            </Link>
          </Button>
        </div>
      </div>
      <NameDialog
        open={naming}
        onOpenChange={setNaming}
        title={t("routines.newFolder")}
        label={t("routines.folderName")}
        saving={createFolder.isPending}
        onSave={(name) => createFolder.mutate(name)}
      />

      {list.isPending ? (
        <Loading className="py-12" />
      ) : list.isError ? (
        <EmptyState
          icon={SearchX}
          title={errorMessage(list.error)}
          action={<Button onClick={() => list.refetch()}>{t("common.retry")}</Button>}
        />
      ) : data!.routines.length === 0 && data!.folders.length === 0 ? (
        <EmptyState icon={ClipboardList} title={t("routines.emptyTitle")} description={t("routines.emptyDescription")} />
      ) : (
        <div className="space-y-3">
          {loose.map((r) => (
            <RoutineCard key={r.id} routine={r} data={data!} order={order} begin={begin} />
          ))}
          {data!.folders.map((f, i) => (
            <FolderGroup key={f.id} folder={f} index={i} data={data!} order={order} begin={begin} />
          ))}
        </div>
      )}

      <Link to="/history" className="mt-8 flex items-center gap-3 rounded-xl border bg-card p-4 transition-colors hover:bg-muted">
        <IconTile icon={History} />
        <div className="min-w-0 flex-1">
          <p className="font-medium">{t("workout.history")}</p>
          <p className="text-sm text-muted-foreground">{t("workout.historyDescription")}</p>
        </div>
        <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      </Link>
      <Link to="/exercises" className="mt-3 flex items-center gap-3 rounded-xl border bg-card p-4 transition-colors hover:bg-muted">
        <IconTile icon={Library} />
        <div className="min-w-0 flex-1">
          <p className="font-medium">{t("workout.library")}</p>
          <p className="text-sm text-muted-foreground">{t("workout.libraryDescription")}</p>
        </div>
        <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      </Link>
    </>
  );
}

/** Saves a new order of folders or routines, shown right away. */
function useOrder() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const save = useMutation({
    meta: { error: t("routines.reorderFailed") },
    mutationFn: api.reorder,
    onSettled: () => qc.invalidateQueries({ queryKey: routinesQuery.queryKey }),
  });
  const apply = (next: Data, body: { folders?: string[]; routines?: string[] }) => {
    qc.setQueryData(routinesQuery.queryKey, next);
    save.mutate(body);
  };
  return {
    qc,
    folder: (data: Data, i: number, by: -1 | 1) => {
      const folders = move(data.folders, i, by);
      apply({ ...data, folders }, { folders: folders.map((f) => f.id) });
    },
    // Moves a routine past its neighbour in the same folder (or among those in none).
    routine: (data: Data, r: Routine, by: -1 | 1) => {
      const all = data.routines;
      const group = all.filter((x) => x.folderId === r.folderId);
      const other = group[group.indexOf(r) + by];
      if (!other) return;
      const routines = all.map((x) => (x === r ? other : x === other ? r : x));
      apply({ ...data, routines }, { routines: routines.map((x) => x.id) });
    },
  };
}

type Order = ReturnType<typeof useOrder>;

function FolderGroup({
  folder: f,
  index,
  data,
  order,
  begin,
}: {
  folder: Folder;
  index: number;
  data: Data;
  order: Order;
  begin: Start;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(true);
  const [renaming, setRenaming] = useState(false);
  const routines = data.routines.filter((r) => r.folderId === f.id);
  const done = () => void order.qc.invalidateQueries({ queryKey: routinesQuery.queryKey });
  const rename = useMutation({
    meta: { error: t("routines.folderSaveFailed") },
    mutationFn: (name: string) => api.renameFolder(f.id, name),
    onSuccess: () => {
      setRenaming(false);
      done();
    },
  });
  const remove = useMutation({
    meta: { error: t("routines.folderDeleteFailed") },
    mutationFn: () => api.deleteFolder(f.id),
    onSuccess: () => {
      toast.success(t("routines.folderDeleted"));
      done();
    },
  });
  return (
    <Collapsible open={open} onOpenChange={setOpen} className="rounded-xl border">
      <div className="flex items-center gap-1 pr-2">
        <CollapsibleTrigger className="flex min-w-0 flex-1 items-center gap-2 p-3 text-left">
          <ChevronDown className={open ? "size-4 shrink-0" : "size-4 shrink-0 -rotate-90"} aria-hidden />
          <span className="truncate font-medium">{f.name}</span>
          <span className="text-sm text-muted-foreground">{routines.length}</span>
        </CollapsibleTrigger>
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon-sm" aria-label={t("routines.folderMenu", { name: f.name })}>
              <MoreVertical />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem asChild>
              <Link to={`/routines/new?folder=${f.id}`}>
                <Plus />
                {t("routines.newInFolder")}
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={() => setRenaming(true)}>
              <Pencil />
              {t("routines.renameFolder")}
            </DropdownMenuItem>
            <DropdownMenuItem disabled={index === 0} onSelect={() => order.folder(data, index, -1)}>
              <ArrowUp />
              {t("routines.moveUp")}
            </DropdownMenuItem>
            <DropdownMenuItem disabled={index === data.folders.length - 1} onSelect={() => order.folder(data, index, 1)}>
              <ArrowDown />
              {t("routines.moveDown")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <ConfirmDialog
              trigger={
                // Keeps the menu from closing (and unmounting the dialog) on select.
                <DropdownMenuItem variant="destructive" onSelect={(e) => e.preventDefault()}>
                  <Trash2 />
                  {t("routines.deleteFolder")}
                </DropdownMenuItem>
              }
              title={t("routines.deleteFolderTitle", { name: f.name })}
              description={t("routines.deleteFolderDescription")}
              confirmLabel={t("routines.deleteConfirm")}
              cancelLabel={t("common.cancel")}
              destructive
              onConfirm={() => remove.mutate()}
            />
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <NameDialog
        open={renaming}
        onOpenChange={setRenaming}
        title={t("routines.renameFolder")}
        label={t("routines.folderName")}
        initial={f.name}
        saving={rename.isPending}
        onSave={(name) => rename.mutate(name)}
      />
      <CollapsibleContent className="space-y-2 px-3 pb-3">
        {routines.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("routines.emptyFolder")}</p>
        ) : (
          routines.map((r) => <RoutineCard key={r.id} routine={r} data={data} order={order} begin={begin} />)
        )}
      </CollapsibleContent>
    </Collapsible>
  );
}

function RoutineCard({ routine: r, data, order, begin }: { routine: Routine; data: Data; order: Order; begin: Start }) {
  const { t, i18n } = useTranslation();
  const group = data.routines.filter((x) => x.folderId === r.folderId);
  const i = group.indexOf(r);
  const names = r.exercises.map((e) => e.exercise?.name).filter(Boolean);
  return (
    <div className="flex items-start gap-1 rounded-xl border bg-card p-3">
      <div className="min-w-0 flex-1">
        <Link to={`/routines/${r.id}`} className="block">
          <p className="truncate font-medium">{r.name}</p>
          {r.lastDoneAt && <p className="text-xs text-muted-foreground">{t("routines.lastDone", { when: ago(r.lastDoneAt, i18n.language) })}</p>}
          <p className="line-clamp-2 text-sm text-muted-foreground">
            {names.length ? names.join(", ") : t("routines.noExercises")}
          </p>
        </Link>
        <Button size="sm" className="mt-2" disabled={r.exercises.length === 0} onClick={() => begin(r)}>
          <Play data-icon="inline-start" />
          {t("routines.start")}
        </Button>
      </div>
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon-sm" aria-label={t("routines.routineMenu", { name: r.name })}>
            <MoreVertical />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem asChild>
            <Link to={`/routines/${r.id}`}>
              <Pencil />
              {t("routines.edit")}
            </Link>
          </DropdownMenuItem>
          <DropdownMenuItem disabled={i === 0} onSelect={() => order.routine(data, r, -1)}>
            <ArrowUp />
            {t("routines.moveUp")}
          </DropdownMenuItem>
          <DropdownMenuItem disabled={i === group.length - 1} onSelect={() => order.routine(data, r, 1)}>
            <ArrowDown />
            {t("routines.moveDown")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

/** "3 days ago", "yesterday", "today" in the app's language. */
function ago(iso: string, lang: string): string {
  const day = 86_400_000;
  const days = Math.round((new Date().setHours(0, 0, 0, 0) - new Date(iso).setHours(0, 0, 0, 0)) / day);
  return new Intl.RelativeTimeFormat(lang, { numeric: "auto" }).format(-days, "day");
}

/** Finished workouts still on their way to the server. */
function Outbox() {
  const { t } = useTranslation();
  const me = useMe();
  const { pending, state } = useOutbox(me.id);
  if (pending.length === 0) return null;
  return (
    <div className="mb-4 space-y-2" aria-live="polite">
      {pending.map((f) => {
        const err = f.error as { code?: string; message?: string; params?: Record<string, string | number> } | string | undefined;
        return (
          <div key={f.id} className="flex items-center gap-3 rounded-xl border bg-card p-3 text-sm">
            {err ? <CloudAlert className="size-5 shrink-0 text-destructive" /> : state === "saving" ? <Spinner /> : <CloudOff className="size-5 shrink-0 text-muted-foreground" />}
            <div className="min-w-0 flex-1">
              <p className="truncate font-medium">{f.workout.title}</p>
              <p className="text-muted-foreground">
                {err
                  ? errorMessage(typeof err === "string" ? err : new ApiError(400, err.message ?? "", err.code, err.params))
                  : t(state === "saving" ? "live.saving" : "live.willRetry")}
              </p>
            </div>
            {err ? (
              <>
                <Button size="sm" variant="outline" onClick={() => retryFinish(f.id, me.id)}>
                  {t("common.retry")}
                </Button>
                <ConfirmDialog
                  trigger={
                    <Button size="icon-sm" variant="ghost" aria-label={t("live.discardFinish")}>
                      <Trash2 />
                    </Button>
                  }
                  title={t("live.discardFinish")}
                  description={t("live.discardFinishDescription")}
                  confirmLabel={t("live.discardConfirm")}
                  cancelLabel={t("common.cancel")}
                  destructive
                  onConfirm={() => discardFinish(f.id)}
                />
              </>
            ) : (
              state === "retry" && (
                <Button size="sm" variant="outline" onClick={() => retryFinish(f.id, me.id)}>
                  {t("common.retry")}
                </Button>
              )
            )}
          </div>
        );
      })}
    </div>
  );
}
