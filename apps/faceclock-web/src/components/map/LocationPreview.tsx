import { useEffect, useRef } from "react";
import L from "leaflet";

interface LocationPreviewProps {
    attendanceLat?: number | null;
    attendanceLng?: number | null;
    officeLat?: number | null;
    officeLng?: number | null;
    officeRadiusMeter?: number | null;
    officeRadius?: number | null;
    isInside?: boolean;
    className?: string;
}

export function LocationPreview({
    attendanceLat,
    attendanceLng,
    officeLat,
    officeLng,
    officeRadiusMeter,
    officeRadius,
    isInside = true,
    className = "h-56 w-full rounded-xl overflow-hidden border border-gray-200",
}: LocationPreviewProps) {
    const containerRef = useRef<HTMLDivElement | null>(null);
    const mapRef = useRef<L.Map | null>(null);
    const effectiveRadius = officeRadius ?? officeRadiusMeter ?? 50;

    useEffect(() => {
        if (!containerRef.current) return;

        const hasAttendance = attendanceLat != null && attendanceLng != null;
        const hasOffice = officeLat != null && officeLng != null;

        if (!hasAttendance && !hasOffice) return;

        const centerLat = attendanceLat ?? officeLat ?? -6.2088;
        const centerLng = attendanceLng ?? officeLng ?? 106.8456;

        const map = L.map(containerRef.current, {
            zoomControl: false,
            attributionControl: false,
        }).setView([centerLat, centerLng], 15);

        L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
            maxZoom: 19,
        }).addTo(map);

        const bounds = L.latLngBounds([]);

        // Draw office if present
        if (hasOffice) {
            const officePos = L.latLng(officeLat!, officeLng!);
            bounds.extend(officePos);

            // Office geofence circle
            L.circle(officePos, {
                radius: effectiveRadius,
                color: "#4f46e5",
                fillColor: "#6366f1",
                fillOpacity: 0.15,
                weight: 1.5,
            }).addTo(map);

            // Office marker
            const officeIcon = L.divIcon({
                className: "custom-div-icon",
                html: `<div style="background-color: #4f46e5; color: white; border-radius: 50%; width: 24px; height: 24px; display: flex; align-items: center; justify-content: center; font-size: 11px; font-weight: bold; border: 2px solid white; box-shadow: 0 2px 4px rgba(0,0,0,0.3);">K</div>`,
                iconSize: [24, 24],
                iconAnchor: [12, 12],
            });
            L.marker(officePos, { icon: officeIcon }).addTo(map).bindPopup("Lokasi Kantor");
        }

        // Draw attendance position
        if (hasAttendance) {
            const attPos = L.latLng(attendanceLat!, attendanceLng!);
            bounds.extend(attPos);

            const color = isInside ? "#10b981" : "#ef4444";
            const attIcon = L.divIcon({
                className: "custom-div-icon",
                html: `<div style="background-color: ${color}; color: white; border-radius: 50%; width: 24px; height: 24px; display: flex; align-items: center; justify-content: center; font-size: 11px; font-weight: bold; border: 2px solid white; box-shadow: 0 2px 4px rgba(0,0,0,0.3);">A</div>`,
                iconSize: [24, 24],
                iconAnchor: [12, 12],
            });
            L.marker(attPos, { icon: attIcon }).addTo(map).bindPopup("Titik Absensi");
        }

        if (hasOffice && hasAttendance) {
            // Connect with dashed line
            L.polyline([[officeLat!, officeLng!], [attendanceLat!, attendanceLng!]], {
                color: "#9ca3af",
                dashArray: "4, 6",
                weight: 1.5,
            }).addTo(map);

            map.fitBounds(bounds, { padding: [30, 30] });
        }

        mapRef.current = map;

        return () => {
            map.remove();
            mapRef.current = null;
        };
    }, [attendanceLat, attendanceLng, officeLat, officeLng, officeRadiusMeter, isInside]);

    return <div ref={containerRef} className={className} />;
}
