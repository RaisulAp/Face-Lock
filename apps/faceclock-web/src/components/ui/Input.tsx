import React from "react";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
    label?: string;
    error?: string;
    helperText?: string;
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
    ({ label, error, helperText, className = "", id, ...props }, ref) => {
        const inputId = id || (label ? label.toLowerCase().replace(/\s+/g, "-") : undefined);

        return (
            <div className="w-full">
                {label && (
                    <label htmlFor={inputId} className="block text-sm font-medium text-gray-700 mb-1">
                        {label}
                    </label>
                )}
                <input
                    id={inputId}
                    ref={ref}
                    className={`block w-full rounded-lg border px-3 py-2 text-sm shadow-sm transition-colors
            focus:outline-none focus:ring-2 focus:ring-offset-1 disabled:bg-gray-50 disabled:text-gray-500
            ${error
                            ? "border-rose-400 text-rose-900 focus:border-rose-500 focus:ring-rose-400"
                            : "border-gray-300 text-gray-900 focus:border-indigo-500 focus:ring-indigo-500"
                        } ${className}`}
                    {...props}
                />
                {error ? (
                    <p className="mt-1 text-xs text-rose-600 font-medium">{error}</p>
                ) : helperText ? (
                    <p className="mt-1 text-xs text-gray-500">{helperText}</p>
                ) : null}
            </div>
        );
    },
);

Input.displayName = "Input";
