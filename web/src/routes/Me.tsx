import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { LogOut, Trash2 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@kipit-labs/kipitidy/button";
import { CopyField } from "@kipit-labs/kipitidy/copy-button";
import { FloatingInput } from "@kipit-labs/kipitidy/floating-input";
import { Loading } from "@kipit-labs/kipitidy/loading";
import { Section } from "@kipit-labs/kipitidy/section";
import { Segmented, SegmentedItem } from "@kipit-labs/kipitidy/segmented";

import { currentLanguage, isLanguage, setLanguage } from "@/i18n";
import { api, type User } from "@/lib/api";
import { authQuery, useMe } from "@/lib/auth";
import { errorMessage } from "@/lib/errors";

import { PageTitle } from "./Layout";

export function Me() {
  const { t } = useTranslation();
  const me = useMe();
  const qc = useQueryClient();
  const logout = useMutation({ mutationFn: api.logout, onSuccess: () => qc.resetQueries() });
  return (
    <>
      <PageTitle>{t("nav.me")}</PageTitle>
      <div className="space-y-4">
        <Profile me={me} />
        {me.isAdmin && <Invites />}
        <About />
        <Button variant="outline" className="w-full" loading={logout.isPending} onClick={() => logout.mutate()}>
          <LogOut data-icon="inline-start" />
          {t("me.signOut")}
        </Button>
      </div>
    </>
  );
}

function Profile({ me }: { me: User }) {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const [name, setName] = useState(me.name);
  const update = useMutation({
    meta: { error: t("me.saveFailed") },
    mutationFn: api.updateMe,
    onSuccess: (user) => {
      qc.setQueryData(authQuery.queryKey, { setupRequired: false, user });
      if (isLanguage(user.language)) setLanguage(user.language);
    },
  });
  const saveName = () => name.trim() !== me.name && update.mutate({ name });
  return (
    <Section title={t("me.profile")} description={me.email}>
      <FloatingInput
        label={t("auth.name")}
        autoComplete="name"
        value={name}
        onChange={(e) => setName(e.target.value)}
        onBlur={saveName}
        onKeyDown={(e) => e.key === "Enter" && saveName()}
      />
      <Setting label={t("me.language")}>
        <Segmented value={me.language || currentLanguage()} onValueChange={(v) => update.mutate({ language: v as User["language"] })}>
          <SegmentedItem value="en">English</SegmentedItem>
          <SegmentedItem value="fr">Français</SegmentedItem>
        </Segmented>
      </Setting>
      <Setting label={t("me.unit")}>
        <Segmented value={me.unit} onValueChange={(v) => update.mutate({ unit: v as User["unit"] })}>
          <SegmentedItem value="kg">kg</SegmentedItem>
          <SegmentedItem value="lb">lb</SegmentedItem>
        </Segmented>
      </Setting>
    </Section>
  );
}

function Setting({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-sm font-medium">{label}</span>
      {children}
    </div>
  );
}

function Invites() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const invites = useQuery({ queryKey: ["invites"], queryFn: api.invites });
  const [email, setEmail] = useState("");
  const [link, setLink] = useState("");
  const create = useMutation({
    meta: { error: false },
    mutationFn: api.createInvite,
    onSuccess: (inv) => {
      setLink(inv.link);
      setEmail("");
      void qc.invalidateQueries({ queryKey: ["invites"] });
    },
  });
  const revoke = useMutation({
    meta: { error: t("me.revokeFailed") },
    mutationFn: api.deleteInvite,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["invites"] }),
  });
  const date = new Intl.DateTimeFormat(currentLanguage(), { dateStyle: "medium" });
  const count = invites.data?.length ?? 0;
  return (
    <Section title={t("me.invites")} description={t("me.invitesDescription")}>
      <form
        className="flex items-start gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          create.mutate(email);
        }}
      >
        <FloatingInput
          className="flex-1"
          label={t("auth.email")}
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          aria-invalid={create.isError || undefined}
          description={create.isError ? errorMessage(create.error) : undefined}
        />
        <Button type="submit" size="lg" className="h-14 rounded-xl" loading={create.isPending}>
          {t("me.invite")}
        </Button>
      </form>
      {link && (
        <div className="space-y-1.5">
          <p className="text-sm text-muted-foreground">{t("me.inviteLink")}</p>
          <CopyField value={link} label={t("me.copy")} onCopy={() => toast.success(t("me.copied"))} />
        </div>
      )}
      {invites.isPending ? (
        <Loading className="py-4" />
      ) : (
        count > 0 && (
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">{t("me.pending", { count })}</p>
            <ul className="divide-y rounded-lg border">
              {invites.data?.map((i) => (
                <li key={i.id} className="flex items-center gap-2 py-2 pr-1 pl-3">
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm">{i.email}</p>
                    <p className="text-xs text-muted-foreground">{t("me.expires", { date: date.format(new Date(i.expiresAt)) })}</p>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    aria-label={t("me.revoke")}
                    title={t("me.revoke")}
                    disabled={revoke.isPending}
                    onClick={() => revoke.mutate(i.id)}
                  >
                    <Trash2 />
                  </Button>
                </li>
              ))}
            </ul>
          </div>
        )
      )}
    </Section>
  );
}

/** The credit the exercise pack's licence asks for; nothing for the built-in library. */
function About() {
  const { t } = useTranslation();
  const lib = useQuery({ queryKey: ["library"], queryFn: api.library, staleTime: Infinity });
  const pack = lib.data?.pack;
  if (!pack?.credit) return null;
  return (
    <Section title={t("me.about")}>
      <a href={pack.creditUrl} target="_blank" rel="noopener" className="text-sm underline underline-offset-4">
        {pack.credit}
      </a>
    </Section>
  );
}
