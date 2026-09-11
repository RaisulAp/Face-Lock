import { Badge } from "../ui/Badge";
import type { AttendanceStatus, AttendanceType, VerificationMethod } from "../../types/api";

export function AttendanceStatusBadge({ status }: { status: AttendanceStatus }) {
  switch (status) {
    case "approved":
      return <Badge variant="success">Disetujui</Badge>;
    case "pending_review":
      return <Badge variant="warning">Menunggu Review</Badge>;
    case "rejected":
      return <Badge variant="danger">Ditolak</Badge>;
    default:
      return <Badge variant="neutral">{status}</Badge>;
  }
}

export function AttendanceTypeBadge({ type }: { type: AttendanceType }) {
  if (type === "checkin") {
    return <Badge variant="info">Masuk</Badge>;
  }
  return <Badge variant="purple">Pulang</Badge>;
}

export function VerificationMethodBadge({ method }: { method: VerificationMethod }) {
  switch (method) {
    case "face":
      return <Badge variant="success">Wajah</Badge>;
    case "fallback":
      return <Badge variant="warning">Fallback</Badge>;
    case "manual":
      return <Badge variant="neutral">Manual</Badge>;
    default:
      return <Badge variant="neutral">{method}</Badge>;
  }
}
