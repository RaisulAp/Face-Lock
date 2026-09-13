import React, { useEffect } from "react";
import { Clock, AlertTriangle } from "lucide-react";
import { useCountdown } from "../../../hooks/useCountdown";

interface EnrollmentSessionTimerProps {
    expiresAt: string;
    onExpired?: () => void;
}

export const EnrollmentSessionTimer: React.FC<EnrollmentSessionTimerProps> = ({ expiresAt, onExpired }) => {
    const expiryTimestamp = new Date(expiresAt).getTime();
    const { minutes, seconds, isExpired, totalSeconds } = useCountdown(expiryTimestamp);

    useEffect(() => {
        if (isExpired && onExpired) {
            onExpired();
        }
    }, [isExpired, onExpired]);

    const isUrgent = totalSeconds < 120; // less than 2 minutes

    if (isExpired) {
        return (
            <div className="inline-flex items-center space-x-1.5 px-3 py-1 rounded-full bg-rose-100 text-rose-800 text-xs font-semibold">
                <AlertTriangle className="w-3.5 h-3.5 text-rose-600" />
                <span>Sesi Berakhir</span>
            </div>
        );
    }

    return (
        <div
            className={`inline-flex items-center space-x-1.5 px-3 py-1 rounded-full text-xs font-semibold border transition-colors ${isUrgent
                    ? "bg-rose-50 text-rose-700 border-rose-200 animate-pulse"
                    : "bg-slate-100 text-slate-700 border-slate-200"
                }`}
        >
            <Clock className={`w-3.5 h-3.5 ${isUrgent ? "text-rose-600" : "text-slate-500"}`} />
            <span>
                Sisa Waktu Sesi:{" "}
                <strong className="font-mono">
                    {String(minutes).padStart(2, "0")}:{String(seconds).padStart(2, "0")}
                </strong>
            </span>
        </div>
    );
};
