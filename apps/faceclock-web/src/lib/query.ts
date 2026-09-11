import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "./api";

export const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            staleTime: 30_000,
            gcTime: 5 * 60_000,
            refetchOnWindowFocus: false,
            retry: (count, error) => {
                // Never retry 4xx errors
                if (error instanceof ApiError) {
                    if (error.status < 500) return false;
                }
                return count < 2;
            },
        },
        mutations: {
            retry: false,
        },
    },
});
