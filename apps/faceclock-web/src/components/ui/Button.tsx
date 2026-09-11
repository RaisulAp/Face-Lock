import React from "react";

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: "primary" | "secondary" | "danger" | "ghost" | "outline";
    size?: "sm" | "md" | "lg";
    isLoading?: boolean;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
    (
        {
            children,
            className = "",
            variant = "primary",
            size = "md",
            isLoading = false,
            disabled,
            ...props
        },
        ref,
    ) => {
        const base =
            "inline-flex items-center justify-center font-medium rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer";

        const variants = {
            primary: "bg-indigo-600 text-white hover:bg-indigo-700 focus:ring-indigo-500",
            secondary: "bg-gray-100 text-gray-800 hover:bg-gray-200 focus:ring-gray-400",
            outline: "border border-gray-300 text-gray-700 bg-white hover:bg-gray-50 focus:ring-indigo-500",
            danger: "bg-rose-600 text-white hover:bg-rose-700 focus:ring-rose-500",
            ghost: "text-gray-600 hover:bg-gray-100 hover:text-gray-900 focus:ring-gray-300",
        };

        const sizes = {
            sm: "px-2.5 py-1.5 text-xs gap-1.5",
            md: "px-4 py-2 text-sm gap-2",
            lg: "px-5 py-2.5 text-base gap-2.5",
        };

        return (
            <button
                ref={ref}
                disabled={disabled || isLoading}
                className={`${base} ${variants[variant]} ${sizes[size]} ${className}`}
                {...props}
            >
                {isLoading && (
                    <svg
                        className="animate-spin -ml-1 mr-2 h-4 w-4 text-current"
                        xmlns="http://www.w3.org/2000/svg"
                        fill="none"
                        viewBox="0 0 24 24"
                    >
                        <circle
                            className="opacity-25"
                            cx="12"
                            cy="12"
                            r="10"
                            stroke="currentColor"
                            strokeWidth="4"
                        />
                        <path
                            className="opacity-75"
                            fill="currentColor"
                            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                        />
                    </svg>
                )}
                {children}
            </button>
        );
    },
);

Button.displayName = "Button";
