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

import type { Routine } from "@/lib/api";
import { useMe } from "@/lib/auth";
import { start } from "@/live/model";
import { getLive, loadPrevious, setLive } from "@/live/store";
import { ulid } from "@/live/ulid";

/**
 * Starts a live workout, empty or from a routine. One runs per device: with one
 * already going, a dialog asks to resume it or discard it and start.
 */
export function useStartWorkout(): [(routine?: Routine) => void, React.ReactNode] {
  const { t } = useTranslation();
  const me = useMe();
  const navigate = useNavigate();
  const [asking, setAsking] = useState<{ routine?: Routine } | null>(null);
  const go = (routine?: Routine) => {
    setLive(start(ulid(), me.id, Date.now(), routine));
    void loadPrevious(routine?.exercises.map((e) => e.exerciseId) ?? []);
    navigate("/live");
  };
  const begin = (routine?: Routine) => (getLive()?.userId === me.id ? setAsking({ routine }) : go(routine));
  const dialog = (
    <AlertDialog open={!!asking} onOpenChange={(o) => !o && setAsking(null)}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("live.inProgressTitle")}</AlertDialogTitle>
          <AlertDialogDescription>{t("live.inProgressDescription")}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
          <AlertDialogAction variant="outline" onClick={() => go(asking?.routine)}>
            {t("live.discardAndStart")}
          </AlertDialogAction>
          <AlertDialogAction onClick={() => navigate("/live")}>{t("live.resume")}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
  return [begin, dialog];
}
