import { useState } from "react";
import { AlertTriangle, ShieldAlert, Info } from "lucide-react";
import { Dialog } from "../ui/Dialog";
import { Button } from "../ui/Button";
import { Input } from "../ui/Input";

interface ConfirmDialogProps {
  isOpen?: boolean;
  open?: boolean;
  onClose: () => void;
  onConfirm: () => void | Promise<void>;
  title: string;
  consequenceText?: string;
  description?: string;
  confirmLabel?: string;
  confirmText?: string;
  cancelLabel?: string;
  variant?: "danger" | "warning" | "info";
  isLoading?: boolean;
  requiredConfirmationText?: string;
}

export function ConfirmDialog({
  isOpen,
  open,
  onClose,
  onConfirm,
  title,
  consequenceText,
  description,
  confirmLabel,
  confirmText,
  cancelLabel = "Batal",
  variant = "warning",
  isLoading = false,
  requiredConfirmationText,
}: ConfirmDialogProps) {
  const [typedConfirm, setTypedConfirm] = useState("");

  const activeOpen = open ?? isOpen ?? false;
  const activeDesc = description ?? consequenceText ?? "";
  const activeConfirmLabel = confirmText ?? confirmLabel ?? "Konfirmasi";

  const isConfirmed = !requiredConfirmationText || typedConfirm === requiredConfirmationText;

  const handleConfirm = async () => {
    if (!isConfirmed) return;
    await onConfirm();
  };

  return (
    <Dialog isOpen={activeOpen} onClose={onClose} title={title}>
      <div className="space-y-4">
        <div className="flex items-start gap-3 p-3.5 rounded-xl bg-gray-50 border border-gray-100">
          {variant === "danger" ? (
            <ShieldAlert className="w-5 h-5 text-rose-600 shrink-0 mt-0.5" />
          ) : variant === "warning" ? (
            <AlertTriangle className="w-5 h-5 text-amber-600 shrink-0 mt-0.5" />
          ) : (
            <Info className="w-5 h-5 text-sky-600 shrink-0 mt-0.5" />
          )}
          <p className="text-sm text-gray-700 leading-relaxed">{activeDesc}</p>
        </div>

        {requiredConfirmationText && (
          <div className="pt-2">
            <p className="text-xs text-gray-500 mb-1.5">
              Ketik <span className="font-mono font-bold text-gray-800">{requiredConfirmationText}</span>{" "}
              untuk mengonfirmasi:
            </p>
            <Input
              value={typedConfirm}
              onChange={(e) => setTypedConfirm(e.target.value)}
              placeholder={requiredConfirmationText}
            />
          </div>
        )}

        <div className="flex items-center justify-end gap-2 pt-3 border-t border-gray-100">
          <Button type="button" variant="outline" onClick={onClose} disabled={isLoading}>
            {cancelLabel}
          </Button>
          <Button
            type="button"
            variant={variant === "danger" ? "danger" : "primary"}
            onClick={handleConfirm}
            isLoading={isLoading}
            disabled={!isConfirmed || isLoading}
          >
            {activeConfirmLabel}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
