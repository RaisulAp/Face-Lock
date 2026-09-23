// ============================================================
// i18n configuration — i18next + react-i18next
// Language stored in localStorage under key: faceclock_language
// Default/fallback language: id (Bahasa Indonesia)
// ============================================================
import i18n from "i18next";
import { initReactI18next } from "react-i18next";

// ID locale resources
import * as id from "./locales/id";

// EN locale resources
import * as en from "./locales/en";

const LANGUAGE_KEY = "faceclock_language";
const DEFAULT_LANGUAGE = "id";

const savedLanguage = localStorage.getItem(LANGUAGE_KEY) || DEFAULT_LANGUAGE;

i18n.use(initReactI18next).init({
    lng: savedLanguage,
    fallbackLng: DEFAULT_LANGUAGE,
    defaultNS: "common",
    ns: [
        "common",
        "nav",
        "auth",
        "role",
        "user",
        "attendance",
        "employee",
        "location",
        "settings",
        "audit",
        "face",
    ],
    resources: {
        id: {
            common: id.common,
            nav: id.nav,
            auth: id.auth,
            role: id.role,
            user: id.user,
            attendance: id.attendance,
            employee: id.employee,
            location: id.location,
            settings: id.settings,
            audit: id.audit,
            face: id.face,
        },
        en: {
            common: en.common,
            nav: en.nav,
            auth: en.auth,
            role: en.role,
            user: en.user,
            attendance: en.attendance,
            employee: en.employee,
            location: en.location,
            settings: en.settings,
            audit: en.audit,
            face: en.face,
        },
    },
    interpolation: {
        escapeValue: false, // React already escapes
    },
});

// Helper to change language and persist to localStorage
export function changeLanguage(lang: "id" | "en") {
    i18n.changeLanguage(lang);
    localStorage.setItem(LANGUAGE_KEY, lang);
}

export type SupportedLanguage = "id" | "en";
export { LANGUAGE_KEY, DEFAULT_LANGUAGE };
export default i18n;
