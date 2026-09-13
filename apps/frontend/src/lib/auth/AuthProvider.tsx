import React, { createContext, useContext, useEffect, useState, useCallback } from "react";
import { tokenStore } from "./tokenStore";
import { authBroadcast } from "./broadcast";
import { requestSingleFlightRefresh } from "./refreshLock";
import { api } from "../api";
import { checkPermission, type PermissionCheck } from "../permissions";
import type { AuthUser, LoginResponse, AuthMeResponse } from "../../types/auth";
import { queryClient } from "../query";

interface AuthContextType {
    user: AuthUser | null;
    isLoading: boolean;
    isAuthenticated: boolean;
    can: (required: PermissionCheck, mode?: "all" | "any") => boolean;
    anyCan: (required: PermissionCheck) => boolean;
    login: ((data: LoginResponse) => void) & ((email: string, password: string) => Promise<void>);
    logout: () => Promise<void>;
    refetchUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
    const [user, setUser] = useState<AuthUser | null>(null);
    const [isLoading, setIsLoading] = useState<boolean>(true);

    const fetchMe = useCallback(async (): Promise<AuthUser | null> => {
        try {
            const data = await api.get<AuthMeResponse | AuthUser>("/api/v1/auth/me");
            const resolvedUser = ("user" in data && data.user) ? data.user : (data as AuthUser);
            setUser(resolvedUser);
            return resolvedUser;
        } catch {
            setUser(null);
            return null;
        }
    }, []);

    const logout = useCallback(async () => {
        const rt = tokenStore.getRefreshToken();
        tokenStore.clear();
        setUser(null);
        queryClient.clear();
        authBroadcast.send({ type: "auth:logout" });

        if (rt) {
            try {
                await api.post("/api/v1/auth/logout", { refresh_token: rt });
            } catch {
                // Ignore logout failure
            }
        }
    }, []);

    const login = useCallback(async (arg1: LoginResponse | string, arg2?: string) => {
        if (typeof arg1 === "string" && arg2 !== undefined) {
            const res = await api.post<LoginResponse>("/api/v1/auth/login", {
                email: arg1,
                password: arg2,
            });
            tokenStore.setAccessToken(res.access_token);
            tokenStore.setRefreshToken(res.refresh_token);
            setUser(res.user);
            authBroadcast.send({ type: "auth:user-changed", userId: res.user.id });
            return;
        }

        const data = arg1 as LoginResponse;
        tokenStore.setAccessToken(data.access_token);
        tokenStore.setRefreshToken(data.refresh_token);
        setUser(data.user);
        authBroadcast.send({ type: "auth:user-changed", userId: data.user.id });
    }, []) as ((data: LoginResponse) => void) & ((email: string, password: string) => Promise<void>);

    const isSuperAdmin = user?.roles?.some((r: any) =>
        typeof r === "string" ? r === "super_admin" : r?.name === "super_admin",
    );

    const can = useCallback(
        (required: PermissionCheck, mode: "all" | "any" = "all"): boolean => {
            if (isSuperAdmin) return true;
            return checkPermission(user?.permissions, required, mode);
        },
        [user, isSuperAdmin],
    );

    const anyCan = useCallback(
        (required: PermissionCheck): boolean => {
            if (isSuperAdmin) return true;
            return checkPermission(user?.permissions, required, "any");
        },
        [user, isSuperAdmin],
    );

    useEffect(() => {
        let mounted = true;

        async function initAuth() {
            if (!tokenStore.hasRefreshToken()) {
                if (mounted) setIsLoading(false);
                return;
            }

            try {
                // Try single flight refresh on initial app load
                const token = await requestSingleFlightRefresh();
                if (token && mounted) {
                    await fetchMe();
                }
            } catch {
                tokenStore.clear();
            } finally {
                if (mounted) setIsLoading(false);
            }
        }

        initAuth();

        const unsubscribe = authBroadcast.subscribe(async (event) => {
            if (event.type === "auth:logout") {
                tokenStore.clear();
                setUser(null);
                queryClient.clear();
            } else if (event.type === "auth:refreshed") {
                await fetchMe();
            } else if (event.type === "auth:user-changed") {
                queryClient.clear();
                await fetchMe();
            }
        });

        return () => {
            mounted = false;
            unsubscribe();
        };
    }, [fetchMe]);

    return (
        <AuthContext.Provider
            value={{
                user,
                isLoading,
                isAuthenticated: !!user,
                can,
                anyCan,
                login,
                logout,
                refetchUser: async () => {
                    await fetchMe();
                },
            }}
        >
            {children}
        </AuthContext.Provider>
    );
}

export function useAuth(): AuthContextType {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error("useAuth must be used within an AuthProvider");
    }
    return context;
}
