import type { AuditTranslations } from "../id/audit";

export const audit = {
    page: {
        title: "Activity Audit Log",
        subtitle: "Track all important activities in the system.",
        loading: "Loading audit log...",
        noData: "No audit log entries found.",
        searchPlaceholder: "Search activity, user, or resource...",
    },
    filters: {
        allActions: "All Actions",
        allActors: "All Users",
        dateFrom: "From Date",
        dateTo: "To Date",
        resource: "Resource",
    },
    table: {
        timestamp: "Timestamp",
        actor: "User",
        action: "Action",
        resource: "Resource",
        resourceId: "Resource ID",
        ipAddress: "IP Address",
        userAgent: "Browser/Device",
        details: "Details",
        status: "Status",
    },
    actions: {
        view: "View Detail",
        export: "Export Log",
    },
    detail: {
        title: "Audit Log Detail",
        before: "Before",
        after: "After",
        metadata: "Metadata",
        noChanges: "No changes recorded.",
    },
    messages: {
        exportSuccess: "Audit log exported successfully.",
        exportError: "Failed to export audit log.",
    },
} satisfies AuditTranslations;
