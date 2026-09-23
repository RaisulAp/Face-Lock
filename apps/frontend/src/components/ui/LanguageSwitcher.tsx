import { useTranslation } from "react-i18next";
import { changeLanguage, type SupportedLanguage } from "../../i18n";
import { Globe } from "lucide-react";

const LANGUAGES: { code: SupportedLanguage; label: string; short: string }[] = [
    { code: "id", label: "Bahasa Indonesia", short: "ID" },
    { code: "en", label: "English", short: "EN" },
];

export function LanguageSwitcher() {
    const { i18n } = useTranslation();
    const currentLang = (i18n.language?.slice(0, 2) || "id") as SupportedLanguage;

    const handleChange = (lang: SupportedLanguage) => {
        if (lang !== currentLang) {
            changeLanguage(lang);
        }
    };

    return (
        <div className="flex items-center gap-1 border border-gray-200 rounded-lg p-0.5 bg-gray-50">
            <Globe className="w-3.5 h-3.5 text-gray-400 ml-1.5" />
            {LANGUAGES.map((lang) => (
                <button
                    key={lang.code}
                    type="button"
                    onClick={() => handleChange(lang.code)}
                    title={lang.label}
                    className={`px-2 py-1 text-[11px] font-semibold rounded-md transition-colors cursor-pointer ${currentLang === lang.code
                            ? "bg-indigo-600 text-white shadow-sm"
                            : "text-gray-500 hover:text-gray-800 hover:bg-gray-100"
                        }`}
                >
                    {lang.short}
                </button>
            ))}
        </div>
    );
}
