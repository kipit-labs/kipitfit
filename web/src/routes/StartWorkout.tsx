import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@kipit-labs/kipitidy/alert-dialog";

import type { LoggedWorkout, Routine } from "@/lib/api";
import { useMe } from "@/lib/auth";
import { editing, start } from "@/live/model";
import { getLive, loadPrevious, setLive } from "@/live/store";
import { ulid } from "@/live/ulid";

/**
 * Starts a live workout, empty, from a routine, or a saved one to edit. One runs per device: with one
 * already going, a dialog asks to resume it or discard it and start.
 */
export function useStartWorkout(): [(routine?: Routine, workout?: LoggedWorkout) => void, React.ReactNode] {
  const { t } = useTranslation();
  const me = useMe();
  const navigate = useNavigate();
  const [asking, setAsking] = useState<{ routine?: Routine; workout?: LoggedWorkout } | null>(null);
  const go = (routine?: Routine, workout?: LoggedWorkout) => {
    if (workout) setLive(editing(workout, me.id));
    else {
      setLive(start(ulid(), me.id, Date.now(), routine));
      void loadPrevious(routine?.exercises.map((e) => e.exerciseId) ?? []);
    }
    navigate("/live");
  };
  const begin = (routine?: Routine, workout?: LoggedWorkout) =>
    getLive()?.userId === me.id ? setAsking({ routine, workout }) : go(routine, workout);
  const dialog = (
    <AlertDialog open={!!asking} onOpenChange={(o) => !o && setAsking(null)}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("live.inProgressTitle")}</AlertDialogTitle>
          <AlertDialogDescription>{t("live.inProgressDescription")}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
          <AlertDialogAction variant="outline" onClick={() => go(asking?.routine, asking?.workout)}>
            {t("live.discardAndStart")}
          </AlertDialogAction>
          <AlertDialogAction onClick={() => navigate("/live")}>{t("live.resume")}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
  return [begin, dialog];
}
