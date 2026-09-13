import { useState, useEffect } from "react";

export interface UseCountdownResult {
    timeLeftSeconds: number;
    totalSeconds: number;
    formattedTime: string;
    isExpired: boolean;
    minutes: number;
    seconds: number;
}

export function useCountdown(
    targetDateOrSeconds: string | Date | number | null | undefined,
    onExpire?: () => void,
): UseCountdownResult {
    const calculateRemaining = (): number => {
        if (!targetDateOrSeconds) return 0;

        if (typeof targetDateOrSeconds === "number") {
            return Math.max(0, targetDateOrSeconds);
        }

        const targetMs =
            typeof targetDateOrSeconds === "string"
                ? new Date(targetDateOrSeconds).getTime()
                : targetDateOrSeconds.getTime();

        const now = Date.now();
        return Math.max(0, Math.floor((targetMs - now) / 1000));
    };

    const [timeLeftSeconds, setTimeLeftSeconds] = useState<number>(calculateRemaining);

    useEffect(() => {
        const interval = setInterval(() => {
            const remaining = calculateRemaining();
            setTimeLeftSeconds(remaining);

            if (remaining <= 0) {
                clearInterval(interval);
                onExpire?.();
            }
        }, 1000);

        return () => clearInterval(interval);
    }, [targetDateOrSeconds, onExpire]);

    const minutes = Math.floor(timeLeftSeconds / 60);
    const seconds = timeLeftSeconds % 60;
    const hours = Math.floor(minutes / 60);

    const formattedTime =
        hours > 0
            ? `${hours.toString().padStart(2, "0")}:${(minutes % 60)
                .toString()
                .padStart(2, "0")}:${seconds.toString().padStart(2, "0")}`
            : `${minutes.toString().padStart(2, "0")}:${seconds.toString().padStart(2, "0")}`;

    return {
        timeLeftSeconds,
        totalSeconds: timeLeftSeconds,
        formattedTime,
        isExpired: timeLeftSeconds <= 0,
        minutes,
        seconds,
    };
}
