// ============================================================
// ENGLISH — common namespace
// Must satisfy CommonTranslations (ID as base schema)
// ============================================================
import type { CommonTranslations } from "../id/common";

export const common = {
    appName: "FaceClock",
    appTagline: "Biometric Attendance & Presence Management System",
    adminPanel: "Admin Panel",

    actions: {
        save: "Save",
        cancel: "Cancel",
        close: "Close",
        delete: "Delete",
        edit: "Edit",
        add: "Add",
        create: "Create",
        update: "Update",
        search: "Search",
        filter: "Filter",
        reset: "Reset",
        refresh: "Refresh",
        export: "Export",
        import: "Import",
        view: "View",
        back: "Back",
        next: "Next",
        previous: "Previous",
        submit: "Submit",
        confirm: "Confirm",
        yes: "Yes",
        no: "No",
        ok: "OK",
        apply: "Apply",
        loading: "Loading...",
        retry: "Retry",
        download: "Download",
        upload: "Upload",
        select: "Select",
        clear: "Clear Filter",
    },

    status: {
        active: "Active",
        inactive: "Inactive",
        pending: "Pending",
        approved: "Approved",
        rejected: "Rejected",
        all: "All",
        enabled: "Enabled",
        disabled: "Disabled",
        success: "Success",
        failed: "Failed",
        loading: "Loading",
        error: "Error",
        empty: "No data found",
        unknown: "Unknown",
    },

    pagination: {
        page: "Page",
        of: "of",
        perPage: "per page",
        showing: "Showing",
        to: "–",
        total: "total",
        first: "First",
        last: "Last",
        entries: "entries",
    },

    time: {
        today: "Today",
        yesterday: "Yesterday",
        thisWeek: "This week",
        thisMonth: "This month",
        date: "Date",
        time: "Time",
        dateTime: "Date & Time",
        createdAt: "Created at",
        updatedAt: "Updated at",
    },

    confirm: {
        deleteTitle: "Confirm Delete",
        deleteMessage: "This action cannot be undone. Continue?",
        yesDelete: "Yes, Delete",
        noCancel: "No, Cancel",
    },

    messages: {
        saveSuccess: "Saved successfully.",
        deleteSuccess: "Deleted successfully.",
        updateSuccess: "Updated successfully.",
        createSuccess: "Created successfully.",
        errorGeneric: "An error occurred. Please try again.",
        noData: "No data found.",
        networkError: "Failed to connect to server.",
    },

    environment: {
        development: "Development",
        production: "Production",
    },

    nav: {
        system: "FaceClock System",
        changePassword: "Change Password & Sessions",
        employeePortal: "Employee Attendance Portal",
        auditLog: "Activity Audit Log",
        logout: "Sign Out",
    },
} satisfies CommonTranslations;
