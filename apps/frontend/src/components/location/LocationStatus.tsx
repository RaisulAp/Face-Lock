import React from "react";
import { MapPin, Navigation, AlertCircle, RefreshCw } from "lucide-react";
import type { NearestOfficeResult } from "../../lib/geo/haversine";
import type { GeolocationErrorDetails } from "../../hooks/useGeolocation";

interface LocationStatusProps {
    isLoading: boolean;
    error: GeolocationErrorDetails | null;
    nearest: NearestOfficeResult;
    onRefresh: () => void;
}

export const LocationStatus: React.FC<LocationStatusProps> = ({ isLoading, error, nearest, onRefresh }) => {
    if (isLoading) {
        return (
            <div className="flex items-center gap-2 px-3 py-2 rounded-xl bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-300 text-xs">
                <div className="w-3.5 h-3.5 rounded-full border-2 border-indigo-600 border-t-transparent animate-spin" />
                <span>Mendeteksi koordinat GPS...</span>
            </div>
        );
    }

    if (error) {
        return (
            <div className="flex items-center justify-between gap-2 px-3 py-2 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900/50 text-rose-700 dark:text-rose-300 text-xs">
                <div className="flex items-center gap-1.5 overflow-hidden">
                    <AlertCircle className="w-4 h-4 flex-shrink-0 text-rose-500" />
                    <span className="truncate">{error.message}</span>
                </div>
                <button
                    type="button"
                    onClick={onRefresh}
                    className="flex-shrink-0 p-1 hover:bg-rose-100 dark:hover:bg-rose-900/40 rounded transition-colors"
                    title="Perbarui Lokasi"
                >
                    <RefreshCw className="w-3.5 h-3.5" />
                </button>
            </div>
        );
    }

    if (!nearest.office) {
        return (
            <div className="flex items-center justify-between gap-2 px-3 py-2 rounded-xl bg-amber-50 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-900/50 text-amber-800 dark:text-amber-300 text-xs">
                <div className="flex items-center gap-1.5">
                    <MapPin className="w-4 h-4 flex-shrink-0 text-amber-500" />
                    <span>Tidak ada kantor aktif yang terdaftar.</span>
                </div>
                <button
                    type="button"
                    onClick={onRefresh}
                    className="flex-shrink-0 p-1 hover:bg-amber-100 dark:hover:bg-amber-900/40 rounded"
                >
                    <RefreshCw className="w-3.5 h-3.5" />
                </button>
            </div>
        );
    }

    const distM = Math.round(nearest.distanceMeters);

    if (nearest.isInside) {
        return (
            <div className="flex items-center justify-between gap-2 px-3.5 py-2 rounded-xl bg-emerald-50 dark:bg-emerald-950/40 border border-emerald-200 dark:border-emerald-900/50 text-emerald-800 dark:text-emerald-300 text-xs">
                <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse flex-shrink-0" />
                    <span className="font-medium truncate">
                        Di area kantor: <strong>{nearest.office.name}</strong> ({distM}m / radius{" "}
                        {nearest.office.radius_meters}m)
                    </span>
                </div>
                <button
                    type="button"
                    onClick={onRefresh}
                    className="flex-shrink-0 p-1 hover:bg-emerald-100 dark:hover:bg-emerald-900/40 rounded"
                    title="Perbarui Lokasi"
                >
                    <RefreshCw className="w-3.5 h-3.5 text-emerald-700 dark:text-emerald-400" />
                </button>
            </div>
        );
    }

    return (
        <div className="flex items-center justify-between gap-2 px-3.5 py-2 rounded-xl bg-amber-50 dark:bg-amber-950/40 border border-amber-200 dark:border-amber-900/50 text-amber-800 dark:text-amber-300 text-xs">
            <div className="flex items-center gap-2">
                <Navigation className="w-4 h-4 flex-shrink-0 text-amber-600 dark:text-amber-400" />
                <span className="truncate">
                    Di luar radius: <strong>{distM}m</strong> dari {nearest.office.name} (maks{" "}
                    {nearest.office.radius_meters}m)
                </span>
            </div>
            <button
                type="button"
                onClick={onRefresh}
                className="flex-shrink-0 p-1 hover:bg-amber-100 dark:hover:bg-amber-900/40 rounded"
                title="Perbarui Lokasi"
            >
                <RefreshCw className="w-3.5 h-3.5 text-amber-700 dark:text-amber-400" />
            </button>
        </div>
    );
};
