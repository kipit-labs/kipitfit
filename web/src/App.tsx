import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Navigate, useLocation } from "react-router";
import { Button } from "@kipit-labs/kipitidy/button";
import { Loading } from "@kipit-labs/kipitidy/loading";

import { isLanguage, setLanguage } from "@/i18n";
import { authQuery } from "@/lib/auth";

import { AcceptInvite, Login, Setup } from "./features/auth/Auth";
import { Layout } from "./features/layout/Layout";

/** Shows the sign-in screen the auth state calls for, else the app. */
export function App() {
  const { t } = useTranslation();
  const auth = useQuery(authQuery);
  const { pathname } = useLocation();
  const language = auth.data?.user?.language;
  // The account's language follows the user across devices.
  useEffect(() => {
    if (isLanguage(language)) setLanguage(language);
  }, [language]);

  if (auth.isPending) return <Loading className="min-h-svh items-center" />;
  if (auth.isError)
    return (
      <div className="flex min-h-svh flex-col items-center justify-center gap-4 p-4 text-center">
        <p>{t("errors.network")}</p>
        <Button variant="outline" onClick={() => auth.refetch()}>
          {t("common.retry")}
        </Button>
      </div>
    );
  const { setupRequired, user } = auth.data;
  if (!user) {
    if (pathname === "/invite") return <AcceptInvite />;
    return setupRequired ? <Setup /> : <Login />;
  }
  if (pathname === "/invite") return <Navigate to="/" replace />;
  return <Layout />;
}
