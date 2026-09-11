import { useQuery } from "@tanstack/react-query";
import { env } from "../../lib/env";
import { tokenStore } from "../../lib/auth/tokenStore";
import { requestSingleFlightRefresh } from "../../lib/auth/refreshLock";

export interface AuthBlobResult {
    blob?: Blob;
    status: number;
    errorText?: string;
}

async function fetchAuthBlob(path: string): Promise<AuthBlobResult> {
    const fullUrl = path.startsWith("http") ? path : `${env.apiBaseUrl}${path}`;
    let token = tokenStore.getAccessToken();

    let response = await fetch(fullUrl, {
        method: "GET",
        credentials: "include",
        headers: {
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
    });

    if (response.status === 401) {
        token = await requestSingleFlightRefresh();
        if (token) {
            response = await fetch(fullUrl, {
                method: "GET",
                credentials: "include",
                headers: {
                    Authorization: `Bearer ${token}`,
                },
            });
        }
    }

    if (response.status === 404) {
        return { status: 404, errorText: "Foto tidak ditemukan." };
    }

    if (response.status === 403) {
        return { status: 403, errorText: "Anda tidak punya izin melihat foto ini." };
    }

    if (response.status === 410) {
        return { status: 410, errorText: "Foto sudah dihapus sesuai kebijakan retensi." };
    }

    if (!response.ok) {
        return { status: response.status, errorText: `Gagal memuat foto (${response.status})` };
    }

    const blob = await response.blob();
    return { blob, status: 200 };
}

export function useAuthBlob(path?: string | null) {
    return useQuery({
        queryKey: ["authBlob", path],
        queryFn: () => (path ? fetchAuthBlob(path) : Promise.resolve(null)),
        enabled: !!path,
        gcTime: 5 * 60_000,
        staleTime: 5 * 60_000,
    });
}
