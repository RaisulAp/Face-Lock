import { formatDateTime, formatTime, formatDate } from "../../lib/datetime";

interface LocalTimeProps {
    value?: string | null;
    mode?: "datetime" | "time" | "date";
    format?: "datetime" | "time" | "date";
    className?: string;
}

export function LocalTime({ value, mode = "datetime", format, className = "" }: LocalTimeProps) {
    if (!value) return <span className="text-gray-400">-</span>;

    const actualMode = format || mode;
    let formatted = "-";
    if (actualMode === "datetime") {
        formatted = formatDateTime(value);
    } else if (actualMode === "time") {
        formatted = formatTime(value);
    } else {
        formatted = formatDate(value);
    }

    return (
        <span className={className} title={`UTC: ${value}`}>
            {formatted}
        </span>
    );
}
