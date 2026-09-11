import React from "react";
import { AlertCircle, CheckCircle2, Lightbulb, Info } from "lucide-react";
import { getHintText } from "../../lib/errors/hints";

interface HintCoachProps {
    hints?: string[];
    type?: "error" | "warning" | "info" | "success";
    className?: string;
}

export const HintCoach: React.FC<HintCoachProps> = ({ hints = [], type = "warning", className = "" }) => {
    if (!hints || hints.length === 0) return null;

    let bgClass =
        "bg-amber-50 dark:bg-amber-950/40 border-amber-200 dark:border-amber-900/50 text-amber-900 dark:text-amber-200";
    let icon = <Lightbulb className="w-4 h-4 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />;

    if (type === "error") {
        bgClass =
            "bg-rose-50 dark:bg-rose-950/40 border-rose-200 dark:border-rose-900/50 text-rose-900 dark:text-rose-200";
        icon = <AlertCircle className="w-4 h-4 text-rose-600 dark:text-rose-400 flex-shrink-0 mt-0.5" />;
    } else if (type === "success") {
        bgClass =
            "bg-emerald-50 dark:bg-emerald-950/40 border-emerald-200 dark:border-emerald-900/50 text-emerald-900 dark:text-emerald-200";
        icon = <CheckCircle2 className="w-4 h-4 text-emerald-600 dark:text-emerald-400 flex-shrink-0 mt-0.5" />;
    } else if (type === "info") {
        bgClass =
            "bg-indigo-50 dark:bg-indigo-950/40 border-indigo-200 dark:border-indigo-900/50 text-indigo-900 dark:text-indigo-200";
        icon = <Info className="w-4 h-4 text-indigo-600 dark:text-indigo-400 flex-shrink-0 mt-0.5" />;
    }

    return (
        <div className={`rounded-xl border p-3 text-xs space-y-1.5 ${bgClass} ${className}`}>
            {hints.map((code, idx) => (
                <div key={idx} className="flex items-start gap-2">
                    {icon}
                    <span className="leading-relaxed font-medium">{getHintText(code)}</span>
                </div>
            ))}
        </div>
    );
};
