import React from "react";

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: "success" | "warning" | "danger" | "info" | "neutral" | "purple" | "outline";
  size?: "sm" | "md";
}

export function Badge({ children, variant = "neutral", size = "md", className = "", ...props }: BadgeProps) {
  const variants = {
    success: "bg-emerald-50 text-emerald-700 border-emerald-200",
    warning: "bg-amber-50 text-amber-700 border-amber-200",
    danger: "bg-rose-50 text-rose-700 border-rose-200",
    info: "bg-sky-50 text-sky-700 border-sky-200",
    neutral: "bg-gray-100 text-gray-700 border-gray-200",
    purple: "bg-purple-50 text-purple-700 border-purple-200",
    outline: "bg-white text-gray-700 border-gray-300",
  };

  const sizes = {
    sm: "px-1.5 py-0.5 text-xs font-medium border rounded",
    md: "px-2.5 py-0.5 text-xs font-semibold border rounded-full",
  };

  return (
    <span
      className={`inline-flex items-center gap-1 leading-none ${variants[variant]} ${sizes[size]} ${className}`}
      {...props}
    >
      {children}
    </span>
  );
}
