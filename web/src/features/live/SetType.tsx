import { useTranslation } from "react-i18next";
import { cn } from "cn";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@kipit-labs/kipitidy/dropdown-menu";

import type { SetType } from "@/lib/api";
import { setTypes } from "@/lib/sets";

const look: Record<SetType, string> = {
  normal: "bg-muted text-foreground",
  warmup: "bg-set-warmup/15 text-set-warmup",
  drop: "bg-set-drop/15 text-set-drop",
  failure: "bg-set-failure/15 text-set-failure",
};

/** The set-number button; it picks the set's type, plus the extra items given. */
export function SetTypeMenu({
  type,
  label,
  onType,
  children,
}: {
  type: SetType;
  label: string;
  onType: (t: SetType) => void;
  children?: React.ReactNode;
}) {
  const { t } = useTranslation();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={t("sets.typeOf", { set: label, type: t(`sets.types.${type}`) })}
        className={cn("flex h-8 w-9 items-center justify-center rounded-md text-sm font-semibold tabular-nums", look[type])}
      >
        {label}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start">
        {setTypes.map((st) => (
          <DropdownMenuItem key={st} onSelect={() => onType(st)}>
            <span className={cn("flex size-6 items-center justify-center rounded text-xs font-semibold", look[st])}>
              {st === "normal" ? "1" : t(`sets.letters.${st}`)}
            </span>
            {t(`sets.types.${st}`)}
          </DropdownMenuItem>
        ))}
        {children && <DropdownMenuSeparator />}
        {children}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
