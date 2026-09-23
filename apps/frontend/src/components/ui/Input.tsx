import React, { useId, useState } from "react";
import { Eye, EyeOff } from "lucide-react";

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: React.ReactNode;
  error?: string;
  helperText?: string;
  /**
   * Show a "reveal password" button inside the field.
   * Only meaningful for password inputs; ignored for every other type.
   */
  showPasswordToggle?: boolean;
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  (
    {
      label,
      error,
      helperText,
      className = "",
      id,
      type = "text",
      showPasswordToggle,
      ...props
    },
    ref,
  ) => {
    const reactId = useId();
    const inputId =
      id || (label && typeof label === "string" ? label.toLowerCase().replace(/\s+/g, "-") : reactId);

    // A password field with the toggle enabled flips between hidden and plain
    // text. `isRevealed` is the only state we own here, so callers keep full
    // control of the value itself.
    const toggleEnabled = showPasswordToggle && type === "password";
    const [isRevealed, setIsRevealed] = useState(false);
    const resolvedType = toggleEnabled && isRevealed ? "text" : type;
    // Reserve room on the right so long values never slide under the button.
    const paddingClass = toggleEnabled && !className.includes("pr-") ? "pr-10" : "";

    return (
      <div className="w-full">
        {label && (
          <label htmlFor={inputId} className="block text-sm font-medium text-gray-700 mb-1">
            {label}
          </label>
        )}
        <div className="relative">
          <input
            id={inputId}
            ref={ref}
            type={resolvedType}
            className={`block w-full rounded-lg border px-3 py-2 text-sm shadow-sm transition-colors
              focus:outline-none focus:ring-2 focus:ring-offset-1 disabled:bg-gray-50 disabled:text-gray-500
              ${error
                ? "border-rose-400 text-rose-900 focus:border-rose-500 focus:ring-rose-400"
                : "border-gray-300 text-gray-900 focus:border-indigo-500 focus:ring-indigo-500"
              } ${paddingClass} ${className}`}
            {...props}
          />
          {toggleEnabled && (
            <button
              type="button"
              onClick={() => setIsRevealed((prev) => !prev)}
              // Not a tab stop: keyboard users can reach the field itself, and
              // the aria-label still exposes the control to screen readers.
              tabIndex={-1}
              disabled={props.disabled}
              aria-label={isRevealed ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
              aria-pressed={isRevealed}
              title={isRevealed ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-gray-400
                transition-colors hover:text-gray-600 focus:outline-none focus:ring-2
                focus:ring-indigo-400 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {isRevealed ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
            </button>
          )}
        </div>
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

