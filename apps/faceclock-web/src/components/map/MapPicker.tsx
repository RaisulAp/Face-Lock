import { useEffect, useRef } from "react";
import L from "leaflet";

// Fix default Leaflet icon paths in bundler
// @ts-expect-error leaflet internal
delete L.Icon.Default.prototype._getIconUrl;
L.Icon.Default.mergeOptions({
    iconRetinaUrl: "https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon-2x.png",
    iconUrl: "https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon.png",
    shadowUrl: "https://unpkg.com/leaflet@1.9.4/dist/images/marker-shadow.png",
});

interface MapPickerProps {
    lat?: number;
    lng?: number;
    latitude?: number;
    longitude?: number;
    radiusMeter?: number;
    radiusMeters?: number;
    onChange: (lat: number, lng: number) => void;
    className?: string;
}

export function MapPicker({
    lat,
    lng,
    latitude,
    longitude,
    radiusMeter,
    radiusMeters,
    onChange,
    className = "h-80 w-full rounded-xl overflow-hidden border border-gray-200",
}: MapPickerProps) {
    const effectiveLat = lat ?? latitude ?? -6.2088;
    const effectiveLng = lng ?? longitude ?? 106.8456;
    const effectiveRadius = radiusMeter ?? radiusMeters ?? 100;

    const containerRef = useRef<HTMLDivElement | null>(null);
    const mapRef = useRef<L.Map | null>(null);
    const markerRef = useRef<L.Marker | null>(null);
    const circleRef = useRef<L.Circle | null>(null);

    // Initialize Map
    useEffect(() => {
        if (!containerRef.current || mapRef.current) return;

        const initialLat = isNaN(effectiveLat) || effectiveLat === 0 ? -6.2088 : effectiveLat; // Default Jakarta
        const initialLng = isNaN(effectiveLng) || effectiveLng === 0 ? 106.8456 : effectiveLng;

        const map = L.map(containerRef.current).setView([initialLat, initialLng], 16);

        L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
            attribution:
                '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
            maxZoom: 19,
        }).addTo(map);

        const marker = L.marker([initialLat, initialLng], { draggable: true }).addTo(map);
        const circle = L.circle([initialLat, initialLng], {
            radius: effectiveRadius,
            color: "#4f46e5",
            fillColor: "#6366f1",
            fillOpacity: 0.2,
            weight: 2,
        }).addTo(map);

        marker.on("dragend", () => {
            const pos = marker.getLatLng();
            circle.setLatLng(pos);
            onChange(pos.lat, pos.lng);
        });

        map.on("click", (e: L.LeafletMouseEvent) => {
            marker.setLatLng(e.latlng);
            circle.setLatLng(e.latlng);
            onChange(e.latlng.lat, e.latlng.lng);
        });

        mapRef.current = map;
        markerRef.current = marker;
        circleRef.current = circle;

        return () => {
            map.remove();
            mapRef.current = null;
        };
    }, []);

    // Update marker and circle when external props change
    useEffect(() => {
        if (!mapRef.current || !markerRef.current || !circleRef.current) return;

        const targetLat = effectiveLat;
        const targetLng = effectiveLng;
        const currentPos = markerRef.current.getLatLng();
        if (Math.abs(currentPos.lat - targetLat) > 0.000001 || Math.abs(currentPos.lng - targetLng) > 0.000001) {
            const newPos = L.latLng(targetLat, targetLng);
            markerRef.current.setLatLng(newPos);
            circleRef.current.setLatLng(newPos);
            mapRef.current.panTo(newPos);
        }
    }, [effectiveLat, effectiveLng]);

    useEffect(() => {
        if (circleRef.current) {
            circleRef.current.setRadius(effectiveRadius);
        }
    }, [effectiveRadius]);

    return <div ref={containerRef} className={className} />;
}
