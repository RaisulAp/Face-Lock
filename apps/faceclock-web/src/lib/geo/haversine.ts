// EarthRadiusMeters is the mean radius of Earth in meters (WGS84 / IUGG recommendation).
export const EarthRadiusMeters = 6371008.8;

/**
 * Calculates the great-circle distance between two points on Earth in meters.
 * Coordinates are in decimal degrees. Longitude wrapping around the 180th meridian is handled.
 * Exactly matches backend apps/faceclock-api/internal/geo/haversine.go.
 */
export function haversineDistance(lat1: number, lon1: number, lat2: number, lon2: number): number {
    // Zero distance optimization
    if (lat1 === lat2 && lon1 === lon2) {
        return 0;
    }

    const phi1 = (lat1 * Math.PI) / 180.0;
    const phi2 = (lat2 * Math.PI) / 180.0;
    const deltaPhi = ((lat2 - lat1) * Math.PI) / 180.0;

    // Handle longitude difference wrapping between -180 and 180 degrees
    let deltaLambdaDeg = lon2 - lon1;
    while (deltaLambdaDeg > 180) {
        deltaLambdaDeg -= 360;
    }
    while (deltaLambdaDeg < -180) {
        deltaLambdaDeg += 360;
    }
    const deltaLambda = (deltaLambdaDeg * Math.PI) / 180.0;

    let a =
        Math.sin(deltaPhi / 2) * Math.sin(deltaPhi / 2) +
        Math.cos(phi1) * Math.cos(phi2) * Math.sin(deltaLambda / 2) * Math.sin(deltaLambda / 2);

    // Clamp 'a' to [0, 1] to avoid NaN from floating-point rounding errors
    if (a < 0) {
        a = 0;
    } else if (a > 1) {
        a = 1;
    }

    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    return EarthRadiusMeters * c;
}

export interface OfficeLocationTarget {
    id: string;
    name: string;
    latitude: number;
    longitude: number;
    radius_meters: number;
}

export interface NearestOfficeResult {
    office: OfficeLocationTarget | null;
    distanceMeters: number;
    isInside: boolean;
}

/**
 * Finds the nearest office location given user's latitude and longitude.
 */
export function findNearestOffice(
    userLat: number,
    userLng: number,
    offices: OfficeLocationTarget[],
): NearestOfficeResult {
    if (!offices || offices.length === 0) {
        return {
            office: null,
            distanceMeters: Infinity,
            isInside: false,
        };
    }

    let nearest: OfficeLocationTarget | null = null;
    let minDistance = Infinity;

    for (const office of offices) {
        const dist = haversineDistance(userLat, userLng, office.latitude, office.longitude);
        if (dist < minDistance) {
            minDistance = dist;
            nearest = office;
        }
    }

    const isInside = nearest ? minDistance <= nearest.radius_meters : false;

    return {
        office: nearest,
        distanceMeters: minDistance,
        isInside,
    };
}
