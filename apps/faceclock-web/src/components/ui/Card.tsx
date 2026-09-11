import type { ReactNode, HTMLAttributes } from "react";

export interface CardProps extends Omit<HTMLAttributes<HTMLDivElement>, "title"> {
  title?: ReactNode;
  subtitle?: ReactNode;
  action?: ReactNode;
  padding?: "none" | "sm" | "md" | "lg" | string;
}

export function Card({ title, subtitle, action, children, padding, className = "", ...props }: CardProps) {
  const padClass =
    padding === "none" ? "p-0" : padding === "sm" ? "p-4" : padding === "lg" ? "p-8" : padding ? "p-4" : "";

  return (
    <div
      className={`bg-white rounded-xl border border-gray-200/80 shadow-xs overflow-hidden ${padClass} ${className}`}
      {...props}
    >
      {(title || action) && (
        <div className="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
          <div>
            {title && <h3 className="text-base font-semibold text-gray-900">{title}</h3>}
            {subtitle && <p className="text-xs text-gray-500 mt-0.5">{subtitle}</p>}
          </div>
          {action && <div>{action}</div>}
        </div>
      )}
      {children}
    </div>
  );
}

export function CardHeader({ children, className = "", ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={`px-5 py-3.5 border-b border-gray-100 flex items-center justify-between ${className}`}
      {...props}
    >
      {children}
    </div>
  );
}

export function CardTitle({ children, className = "", ...props }: React.HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h3 className={`text-sm font-semibold text-gray-900 tracking-tight ${className}`} {...props}>
      {children}
    </h3>
  );
}

export function CardContent({ children, className = "", ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={`p-5 ${className}`} {...props}>
      {children}
    </div>
  );
}
