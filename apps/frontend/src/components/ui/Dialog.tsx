import React, { useEffect } from "react";
import { X } from "lucide-react";

export interface DialogProps {
  isOpen?: boolean;
  open?: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  children: React.ReactNode;
  maxWidth?: "sm" | "md" | "lg" | "xl" | "2xl" | "3xl" | "4xl" | "5xl" | "6xl";
  size?: "sm" | "md" | "lg" | "xl" | "2xl" | "3xl" | "4xl" | "5xl" | "6xl" | string;
}

export function Dialog({
  isOpen,
  open,
  onClose,
  title,
  description,
  children,
  maxWidth,
  size = "md",
}: DialogProps) {
  const activeOpen = open ?? isOpen ?? false;
  const activeSize = (maxWidth ?? size) as "sm" | "md" | "lg" | "xl" | "2xl" | "3xl" | "4xl" | "5xl" | "6xl";

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape" && activeOpen) {
        onClose();
      }
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [activeOpen, onClose]);

  if (!activeOpen) return null;

  const maxWClass =
    activeSize === "sm"
      ? "max-w-sm"
      : activeSize === "lg"
        ? "max-w-lg"
        : activeSize === "xl"
          ? "max-w-xl"
          : activeSize === "2xl"
            ? "max-w-2xl"
            : activeSize === "3xl"
              ? "max-w-3xl"
              : activeSize === "4xl"
                ? "max-w-4xl"
                : activeSize === "5xl"
                  ? "max-w-5xl"
                  : activeSize === "6xl"
                    ? "max-w-6xl"
                    : "max-w-md";

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/40 backdrop-blur-xs transition-opacity" onClick={onClose} />

      <div className="flex min-h-full items-center justify-center p-4 text-center sm:p-0">
        <div
          className={`relative transform overflow-hidden rounded-2xl bg-white text-left shadow-xl transition-all w-full ${maxWClass} my-8`}
          onClick={(e) => e.stopPropagation()}
        >
          {title && (
            <div className="flex items-center justify-between border-b border-gray-100 px-6 py-4">
              <div>
                <h3 className="text-base font-semibold leading-6 text-gray-900">{title}</h3>
                {description && <p className="text-xs text-gray-500 mt-0.5">{description}</p>}
              </div>
              <button
                type="button"
                onClick={onClose}
                className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-500 cursor-pointer"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
          )}
          <div className={title ? "p-6" : ""}>{children}</div>
        </div>
      </div>
    </div>
  );
}

export function DialogHeader({ children, className = "", ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={`px-6 py-4 border-b border-gray-100 flex items-center justify-between ${className}`}
      {...props}
    >
      {children}
    </div>
  );
}

export function DialogTitle({
  children,
  className = "",
  ...props
}: React.HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h3 className={`text-base font-semibold text-gray-900 leading-6 ${className}`} {...props}>
      {children}
    </h3>
  );
}

export function DialogContent({ children, className = "", ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={`p-6 ${className}`} {...props}>
      {children}
    </div>
  );
}

export function DialogFooter({ children, className = "", ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={`px-6 py-4 bg-gray-50 border-t border-gray-100 flex items-center justify-end gap-2.5 ${className}`}
      {...props}
    >
      {children}
    </div>
  );
}
