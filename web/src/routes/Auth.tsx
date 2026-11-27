import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, type FormEvent, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate, useSearchParams } from "react-router";
import { Button } from "@kipit-labs/kipitidy/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@kipit-labs/kipitidy/card";
import { FloatingInput } from "@kipit-labs/kipitidy/floating-input";

import { api, type User } from "@/lib/api";
import { authQuery } from "@/lib/auth";
import { errorMessage } from "@/lib/errors";

function Shell({ title, subtitle, children }: { title: string; subtitle?: string; children: ReactNode }) {
  return (
    <div className="flex min-h-svh items-center justify-center bg-muted/40 p-4">
      <div className="w-full max-w-sm space-y-6">
        <p className="flex items-center justify-center gap-2 font-heading text-2xl font-semibold">
          <img src="/icon.svg" alt="" className="size-8" />
          kipitfit
        </p>
        <Card>
          <CardHeader>
            {/* The kit's CardTitle is a div: screen readers need the page's heading. */}
            <CardTitle>
              <h1>{title}</h1>
            </CardTitle>
            {subtitle && <CardDescription>{subtitle}</CardDescription>}
          </CardHeader>
          {children}
        </Card>
      </div>
    </div>
  );
}

/** A credentials form: signs in with what mutationFn returns. */
function useSignIn<T>(mutationFn: (v: T) => Promise<User>) {
  const qc = useQueryClient();
  return useMutation({
    meta: { error: false },
    mutationFn,
    onSuccess: (user) => qc.setQueryData(authQuery.queryKey, { setupRequired: false, user }),
  });
}

function Form({
  onSubmit,
  pending,
  error,
  submit,
  children,
}: {
  onSubmit: () => void;
  pending: boolean;
  error: unknown;
  submit: string;
  children: ReactNode;
}) {
  return (
    <form
      onSubmit={(e: FormEvent) => {
        e.preventDefault();
        onSubmit();
      }}
    >
      <CardContent className="space-y-4">
        {children}
        {error != null && (
          <p role="alert" className="text-sm text-destructive">
            {errorMessage(error)}
          </p>
        )}
        <Button type="submit" size="lg" className="mt-2 h-11 w-full rounded-xl" loading={pending}>
          {submit}
        </Button>
      </CardContent>
    </form>
  );
}

function PasswordInput({ value, onChange, isNew }: { value: string; onChange: (v: string) => void; isNew?: boolean }) {
  const { t } = useTranslation();
  return (
    <FloatingInput
      label={t("auth.password")}
      type="password"
      required
      minLength={isNew ? 10 : undefined}
      autoComplete={isNew ? "new-password" : "current-password"}
      description={isNew ? t("auth.passwordHint", { min: 10 }) : undefined}
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  );
}

export function Setup() {
  const { t } = useTranslation();
  const [f, setF] = useState({ setupToken: "", email: "", name: "", password: "" });
  const set = (k: keyof typeof f) => (v: string) => setF({ ...f, [k]: v });
  const setup = useSignIn(api.setup);
  return (
    <Shell title={t("auth.setupTitle")} subtitle={t("auth.setupSubtitle")}>
      <Form onSubmit={() => setup.mutate(f)} pending={setup.isPending} error={setup.error} submit={t("auth.setupSubmit")}>
        <FloatingInput
          label={t("auth.setupToken")}
          required
          autoFocus
          autoComplete="off"
          value={f.setupToken}
          onChange={(e) => set("setupToken")(e.target.value)}
        />
        <FloatingInput label={t("auth.name")} autoComplete="name" value={f.name} onChange={(e) => set("name")(e.target.value)} />
        <FloatingInput
          label={t("auth.email")}
          type="email"
          required
          autoComplete="email"
          value={f.email}
          onChange={(e) => set("email")(e.target.value)}
        />
        <PasswordInput isNew value={f.password} onChange={set("password")} />
      </Form>
    </Shell>
  );
}

export function Login() {
  const { t } = useTranslation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const login = useSignIn(api.login);
  return (
    <Shell title={t("auth.loginTitle")} subtitle={t("auth.loginSubtitle")}>
      <Form onSubmit={() => login.mutate({ email, password })} pending={login.isPending} error={login.error} submit={t("auth.loginSubmit")}>
        <FloatingInput
          label={t("auth.email")}
          type="email"
          required
          autoFocus
          autoComplete="username"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <PasswordInput value={password} onChange={setPassword} />
      </Form>
    </Shell>
  );
}

export function AcceptInvite() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const accept = useSignIn(api.acceptInvite);
  return (
    <Shell title={t("auth.inviteTitle")} subtitle={t("auth.inviteSubtitle")}>
      <Form
        onSubmit={() =>
          accept.mutate({ token: params.get("token") ?? "", name, password }, { onSuccess: () => navigate("/", { replace: true }) })
        }
        pending={accept.isPending}
        error={accept.error}
        submit={t("auth.inviteSubmit")}
      >
        <FloatingInput label={t("auth.name")} autoFocus autoComplete="name" value={name} onChange={(e) => setName(e.target.value)} />
        <PasswordInput isNew value={password} onChange={setPassword} />
      </Form>
    </Shell>
  );
}
