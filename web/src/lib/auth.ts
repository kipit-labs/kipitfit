import { useQuery } from "@tanstack/react-query";

import { api, type User } from "./api";

export const authQuery = { queryKey: ["auth"], queryFn: api.authState, staleTime: Infinity } as const;

/** The signed-in user; only inside the signed-in shell. */
export function useMe(): User {
  return useQuery(authQuery).data!.user!;
}
