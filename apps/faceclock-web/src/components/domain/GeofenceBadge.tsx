import { Badge } from "../ui/Badge";
import type { GeofenceStatus } from "../../types/api";

export interface GeofenceBadgeProps {
    status?: GeofenceStatus | null;
    isWithin?: boolean;
    distanceMeters?: number | null;
}

export function GeofenceBadge({ status, isWithin, distanceMeters }: GeofenceBadgeProps) {
    if (isWithin !== undefined) {
        if (isWithin) {
            return <Badge variant="success">Dalam Radius</Badge>;
        }
        const distText = distanceMeters != null ? ` (${Math.round(distanceMeters)}m)` : "";
        return <Badge variant="danger">Luar Radius{distText}</Badge>;
    }

    if (!status || status === "unknown") {
        return <Badge variant="neutral">Tidak Diketahui</Badge>;
    }

    if (status === "inside") {
        return <Badge variant="success">Dalam Radius</Badge>;
    }

    return <Badge variant="danger">Luar Radius</Badge>;
}
