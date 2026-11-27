import { useQuery } from "@tanstack/react-query";
import { ChevronRight, Plus, Search, SearchX, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate, useSearchParams } from "react-router";
import { cn } from "cn";
import { Button } from "@kipit-labs/kipitidy/button";
import { EmptyState } from "@kipit-labs/kipitidy/empty-state";
import { IconTile } from "@kipit-labs/kipitidy/icon-tile";
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@kipit-labs/kipitidy/input-group";
import { Loading } from "@kipit-labs/kipitidy/loading";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@kipit-labs/kipitidy/select";

import type { ExerciseSummary } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { categories, equipment, equipmentIcon, exercisesQuery, filterExercises, muscles, type Filters } from "@/lib/exercises";

import { ExerciseForm } from "./ExerciseForm";
import { PageTitle } from "./Layout";

// Rows rendered at a time; more come as the end of the list scrolls into view.
const page = 40;
// Select items can't have an empty value.
const all = "all";

export const chip = "rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground";

/** The exercise library: search, filters, and the user's own exercises. */
export function Exercises() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const list = useQuery(exercisesQuery);
  // Kept in the URL so going back from an exercise restores the search.
  const [params, setParams] = useSearchParams();
  const filters: Filters = {
    q: params.get("q") ?? "",
    muscle: params.get("muscle") ?? "",
    equipment: params.get("equipment") ?? "",
    category: params.get("category") ?? "",
  };
  const set = (key: keyof Filters, value: string) =>
    setParams(
      (p) => {
        if (value && value !== all) p.set(key, value);
        else p.delete(key);
        return p;
      },
      { replace: true },
    );
  const { q, muscle, equipment: equip, category } = filters;
  const shown = useMemo(
    () => filterExercises(list.data ?? [], { q, muscle, equipment: equip, category }, t),
    [list.data, q, muscle, equip, category, t],
  );
  const key = [q, muscle, equip, category].join("|");
  const filtered = !!(filters.muscle || filters.equipment || filters.category);

  return (
    <>
      <div className="mb-6 flex items-center justify-between gap-2">
        <PageTitle className="mb-0">{t("exercises.title")}</PageTitle>
        <ExerciseForm
          trigger={
            <Button size="sm">
              <Plus data-icon="inline-start" />
              {t("exercises.new")}
            </Button>
          }
          onSaved={(e) => navigate(`/exercises/${e.id}`)}
        />
      </div>
      <div className="sticky top-0 z-10 -mx-4 space-y-2 bg-background/95 px-4 pt-1 pb-3 backdrop-blur">
        <SearchField value={filters.q} onChange={(q) => set("q", q)} />
        <div className="grid grid-cols-3 gap-2">
          <Filter value={filters.muscle} onChange={(v) => set("muscle", v)} label={t("exercises.fields.muscle")} allLabel={t("exercises.allMuscles")}>
            {muscles.map((m) => ({ value: m, label: t(`exercises.muscles.${m}`) }))}
          </Filter>
          <Filter value={filters.equipment} onChange={(v) => set("equipment", v)} label={t("exercises.fields.equipment")} allLabel={t("exercises.allEquipment")}>
            {equipment.map((e) => ({ value: e, label: t(`exercises.equipment.${e}`) }))}
          </Filter>
          <Filter value={filters.category} onChange={(v) => set("category", v)} label={t("exercises.fields.category")} allLabel={t("exercises.allCategories")}>
            {categories.map((c) => ({ value: c, label: t(`exercises.categories.${c}`) }))}
          </Filter>
        </div>
        {list.data && (
          <p className="text-sm text-muted-foreground" aria-live="polite">
            {t("exercises.count", { count: shown.length })}
          </p>
        )}
      </div>
      {list.isPending ? (
        <Loading className="py-12" />
      ) : list.isError ? (
        <EmptyState
          icon={SearchX}
          title={errorMessage(list.error)}
          action={<Button onClick={() => list.refetch()}>{t("common.retry")}</Button>}
        />
      ) : shown.length === 0 ? (
        <EmptyState
          icon={SearchX}
          title={t("exercises.noResults")}
          description={t("exercises.noResultsDescription")}
          action={
            (filtered || filters.q) && (
              <Button variant="outline" onClick={() => setParams({}, { replace: true })}>
                {t("exercises.clearFilters")}
              </Button>
            )
          }
        />
      ) : (
        <Paged key={key} items={shown}>
          {(e) => <Row key={e.id} exercise={e} />}
        </Paged>
      )}
    </>
  );
}

/** The library's search box, with its own clear button. */
export function SearchField({ value, onChange }: { value: string; onChange: (q: string) => void }) {
  const { t } = useTranslation();
  return (
    <InputGroup className="h-11">
      <InputGroupAddon>
        <Search />
      </InputGroupAddon>
      <InputGroupInput
        type="search"
        enterKeyHint="search"
        // The app's own clear button replaces WebKit's built-in one.
        className="[&::-webkit-search-cancel-button]:appearance-none"
        aria-label={t("exercises.search")}
        placeholder={t("exercises.search")}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
      {value && (
        <InputGroupAddon align="inline-end">
          <InputGroupButton size="icon-xs" aria-label={t("exercises.clearSearch")} onClick={() => onChange("")}>
            <X />
          </InputGroupButton>
        </InputGroupAddon>
      )}
    </InputGroup>
  );
}

export function Filter({
  value,
  onChange,
  label,
  allLabel,
  children,
}: {
  value: string;
  onChange: (v: string) => void;
  /** Shown while unset: the whole list. */
  label: string;
  allLabel: string;
  children: { value: string; label: string }[];
}) {
  const options = [...children].sort((a, b) => a.label.localeCompare(b.label));
  return (
    <Select value={value || all} onValueChange={onChange}>
      <SelectTrigger
        size="sm"
        aria-label={label}
        className={cn("w-full min-w-0 [&>span]:truncate", value && "border-primary text-primary")}
      >
        <SelectValue>{value ? options.find((o) => o.value === value)?.label : label}</SelectValue>
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={all}>{allLabel}</SelectItem>
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value}>
            {o.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

/** Renders a long list a page at a time, so the whole library stays smooth on a phone. */
export function Paged<T>({ items, children }: { items: T[]; children: (item: T) => React.ReactNode }) {
  const [count, setCount] = useState(page);
  const end = useRef<HTMLLIElement>(null);
  const more = count < items.length;
  useEffect(() => {
    if (!more || !end.current) return;
    const io = new IntersectionObserver(([e]) => e.isIntersecting && setCount((c) => c + page), {
      rootMargin: "600px",
    });
    io.observe(end.current);
    return () => io.disconnect();
  }, [more, count]);
  return (
    <ul className="divide-y">
      {items.slice(0, count).map(children)}
      {more && <li ref={end} aria-hidden className="h-16" />}
    </ul>
  );
}

function Row({ exercise: e }: { exercise: ExerciseSummary }) {
  const { t } = useTranslation();
  return (
    <li>
      <Link to={`/exercises/${e.id}`} className="flex items-center gap-3 py-3">
        <IconTile icon={equipmentIcon(e.equipment)} />
        <div className="min-w-0 flex-1 space-y-1">
          <p className="truncate font-medium">{e.name}</p>
          <div className="flex flex-wrap gap-1">
            {e.custom && <span className={`${chip} bg-primary/10 text-primary`}>{t("exercises.custom")}</span>}
            {e.primaryMuscles.map((m) => (
              <span key={m} className={chip}>
                {t(`exercises.muscles.${m}`)}
              </span>
            ))}
            <span className={chip}>{t(`exercises.equipment.${e.equipment}`)}</span>
          </div>
        </div>
        <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      </Link>
    </li>
  );
}
