import React from "react";

export interface SelectOption {
    value: string;
    label: string;
}

export interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
    label?: string;
    error?: string;
    helperText?: string;
    options?: SelectOption[];
}

export const Select = React.forwardRef<HTMLSelectElement, SelectProps>(
    ({ label, error, helperText, options, children, className = "", id, ...props }, ref) => {
        const selectId = id || (label ? label.toLowerCase().replace(/\s+/g, "-") : undefined);

        return (
            <div className="w-full">
                {label && (
                    <label htmlFor={selectId} className="block text-sm font-medium text-gray-700 mb-1">
                        {label}
                    </label>
                )}
                <select
                    id={selectId}
                    ref={ref}
                    className={`block w-full rounded-lg border px-3 py-2 text-sm shadow-sm transition-colors bg-white
            focus:outline-none focus:ring-2 focus:ring-offset-1 disabled:bg-gray-50 disabled:text-gray-500
            ${error
                            ? "border-rose-400 text-rose-900 focus:border-rose-500 focus:ring-rose-400"
                            : "border-gray-300 text-gray-900 focus:border-indigo-500 focus:ring-indigo-500"
                        } ${className}`}
                    {...props}
                >
                    {options
                        ? options.map((opt) => (
                            <option key={opt.value} value={opt.value}>
                                {opt.label}
                            </option>
                        ))
                        : children}
                </select>
                {error ? (
                    <p className="mt-1 text-xs text-rose-600 font-medium">{error}</p>
                ) : helperText ? (
                    <p className="mt-1 text-xs text-gray-500">{helperText}</p>
                ) : null}
            </div>
        );
    },
);

Select.displayName = "Select";
