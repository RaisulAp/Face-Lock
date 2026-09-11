import { useState, useEffect, useCallback, useMemo } from "react";
import { findNearestOffice, type OfficeLocationTarget, type NearestOfficeResult } from "../lib/geo/haversine";

export type GeolocationErrorKind =
    | "NOT_SUPPORTED"
    | "NOT_SECURE"
    | "PERMISSION_DENIED"
    | "POSITION_UNAVAILABLE"
    | "TIMEOUT"
    | "ACCURACY_LOW"
    | "UNKNOWN";

export interface GeolocationErrorDetails {
    kind: GeolocationErrorKind;
    message: string;
    originalError?: unknown;
}

export interface GeolocationCoords {
    latitude: number;
    longitude: number;
    accuracy: number;
    timestamp: number;
}

export interface UseGeolocationOptions {
    offices?: OfficeLocationTarget[];
    maxAccuracyMeters?: number; // warn/error if accuracy > maxAccuracyMeters
    autoFetch?: boolean;
}

export interface UseGeolocationResult {
    coords: GeolocationCoords | null;
    isLoading: boolean;
    error: GeolocationErrorDetails | null;
    nearest: NearestOfficeResult;
    refreshLocation: () => Promise<GeolocationCoords | null>;
    clearError: () => void;
}

export function useGeolocation(options: UseGeolocationOptions = {}): UseGeolocationResult {
    const { offices = [], maxAccuracyMeters = 150, autoFetch = true } = options;

    const [coords, setCoords] = useState<GeolocationCoords | null>(null);
    const [isLoading, setIsLoading] = useState<boolean>(autoFetch);
    const [error, setError] = useState<GeolocationErrorDetails | null>(null);

    const fetchPosition = useCallback(async (): Promise<GeolocationCoords | null> => {
        if (typeof navigator === "undefined" || !navigator.geolocation) {
            const err: GeolocationErrorDetails = {
                kind: "NOT_SUPPORTED",
                message: "Browser Anda tidak mendukung layanan lokasi (Geolocation).",
            };
            setError(err);
            setIsLoading(false);
            return null;
        }

        setIsLoading(true);
        setError(null);

        return new Promise<GeolocationCoords | null>((resolve) => {
            navigator.geolocation.getCurrentPosition(
                (position) => {
                    const lat = position.coords.latitude;
                    const lng = position.coords.longitude;
                    const acc = position.coords.accuracy;
                    const ts = position.timestamp;

                    const newCoords: GeolocationCoords = {
                        latitude: lat,
                        longitude: lng,
                        accuracy: acc,
                        timestamp: ts,
                    };

                    if (maxAccuracyMeters && acc > maxAccuracyMeters) {
                        setError({
                            kind: "ACCURACY_LOW",
                            message: `Akurasi GPS rendah (±${Math.round(acc)}m). Pindahlah ke luar ruangan atau aktifkan GPS berakurasi tinggi.`,
                        });
                    }

                    setCoords(newCoords);
                    setIsLoading(false);
                    resolve(newCoords);
                },
                (geoErr) => {
                    let kind: GeolocationErrorKind = "UNKNOWN";
                    let message = "Gagal mengambil koordinat lokasi perangkat.";

                    switch (geoErr.code) {
                        case geoErr.PERMISSION_DENIED:
                            kind = "PERMISSION_DENIED";
                            message = "Izin akses lokasi ditolak. Harap berikan izin akses lokasi di browser.";
                            break;
                        case geoErr.POSITION_UNAVAILABLE:
                            kind = "POSITION_UNAVAILABLE";
                            message = "Informasi lokasi saat ini tidak tersedia pada perangkat Anda.";
                            break;
                        case geoErr.TIMEOUT:
                            kind = "TIMEOUT";
                            message = "Waktu permintaan lokasi habis (timeout). Coba segarkan kembali.";
                            break;
                    }

                    const details: GeolocationErrorDetails = {
                        kind,
                        message,
                        originalError: geoErr,
                    };
                    setError(details);
                    setIsLoading(false);
                    resolve(null);
                },
                {
                    enableHighAccuracy: true,
                    timeout: 10000,
                    maximumAge: 0,
                },
            );
        });
    }, [maxAccuracyMeters]);

    useEffect(() => {
        if (autoFetch) {
            fetchPosition();
        }
    }, [autoFetch, fetchPosition]);

    const nearest = useMemo(() => {
        if (!coords) {
            return {
                office: null,
                distanceMeters: Infinity,
                isInside: false,
            };
        }
        return findNearestOffice(coords.latitude, coords.longitude, offices);
    }, [coords, offices]);

    return {
        coords,
        isLoading,
        error,
        nearest,
        refreshLocation: fetchPosition,
        clearError: () => setError(null),
    };
}
