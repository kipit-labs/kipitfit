import i18n from "@/i18n";

import { ApiError } from "./api";

/**
 * What to tell the user about a failed request: the translation of the API's
 * error code when there is one, else the API's own message.
 */
export function errorMessage(err: unknown): string {
  if (!(err instanceof ApiError)) return i18n.t("errors.network");
  if (err.code && i18n.exists(`errors.${err.code}`)) return i18n.t(`errors.${err.code}`, err.params);
  return err.message;
}
