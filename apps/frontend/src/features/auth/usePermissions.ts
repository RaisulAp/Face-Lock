import { useAuth } from "../../lib/auth/useAuth";

export function usePermissions() {
  const { can, user } = useAuth();
  return {
    can,
    permissions: user?.permissions ?? [],
  };
}
