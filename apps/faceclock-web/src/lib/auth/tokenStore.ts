// Token storage implementing Decision D23:
// - Access token in JavaScript memory variable (never in localStorage to prevent long-term XSS theft)
// - Refresh token in localStorage to survive browser refresh
// - Mutating operations clear state cleanly

const REFRESH_TOKEN_KEY = "faceclock_rt";

let inMemoryAccessToken: string | null = null;

export const tokenStore = {
    getAccessToken(): string | null {
        return inMemoryAccessToken;
    },

    setAccessToken(token: string | null): void {
        inMemoryAccessToken = token;
    },

    getRefreshToken(): string | null {
        try {
            return localStorage.getItem(REFRESH_TOKEN_KEY);
        } catch {
            return null;
        }
    },

    setRefreshToken(token: string | null): void {
        try {
            if (token) {
                localStorage.setItem(REFRESH_TOKEN_KEY, token);
            } else {
                localStorage.removeItem(REFRESH_TOKEN_KEY);
            }
        } catch {
            // Storage unavailable or disabled
        }
    },

    hasRefreshToken(): boolean {
        return !!this.getRefreshToken();
    },

    clear(): void {
        inMemoryAccessToken = null;
        try {
            localStorage.removeItem(REFRESH_TOKEN_KEY);
        } catch {
            // Ignore storage errors
        }
    },
};
