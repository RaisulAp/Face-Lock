import { useState } from "react";
import { AlertCircle, AlertTriangle, Info, RefreshCw, Copy, Check } from "lucide-react";
import { ApiError } from "../../lib/api";
import { getErrorPresentation } from "../../lib/errors/messages";
import { Button } from "../ui/Button";

interface ErrorStateProps {
  error: unknown;
  onRetry?: () => void;
  className?: string;
}

export function ErrorState({ error, onRetry, className = "" }: ErrorStateProps) {
  const [copied, setCopied] = useState(false);

  let code = "INTERNAL_ERROR";
  let message = "Terjadi kesalahan";
  let requestId: string | undefined;

  if (error instanceof ApiError) {
    code = error.code;
    message = error.message;
    requestId = error.requestId;
  } else if (error instanceof Error) {
    message = error.message;
  }

  const presentation = getErrorPresentation(code);

  const copyRequestId = () => {
    if (requestId) {
      navigator.clipboard.writeText(requestId);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const isWarning = presentation.severity === "warning";
  const isInfo = presentation.severity === "info";

  return (
    <div
      className={`rounded-xl border p-6 my-4 ${
        isInfo
          ? "bg-sky-50/50 border-sky-200"
          : isWarning
            ? "bg-amber-50/50 border-amber-200"
            : "bg-rose-50/50 border-rose-200"
      } ${className}`}
    >
      <div className="flex items-start gap-3">
        <div className="shrink-0 mt-0.5">
          {isInfo ? (
            <Info className="w-5 h-5 text-sky-600" />
          ) : isWarning ? (
            <AlertTriangle className="w-5 h-5 text-amber-600" />
          ) : (
            <AlertCircle className="w-5 h-5 text-rose-600" />
          )}
        </div>

        <div className="flex-1">
          <h4 className="text-sm font-semibold text-gray-900">{presentation.title}</h4>
          <p className="text-xs text-gray-600 mt-1">
            {presentation.body} {message && message !== presentation.title ? `(${message})` : ""}
          </p>

          {presentation.showRequestId && requestId && (
            <div className="mt-3 flex items-center gap-2">
              <span className="text-[11px] font-mono text-gray-500 bg-white px-2 py-0.5 rounded border border-gray-200">
                ID: {requestId}
              </span>
              <button
                type="button"
                onClick={copyRequestId}
                className="inline-flex items-center gap-1 text-[11px] text-gray-600 hover:text-gray-900 cursor-pointer"
              >
                {copied ? <Check className="w-3 h-3 text-emerald-600" /> : <Copy className="w-3 h-3" />}
                {copied ? "Tersalin" : "Salin ID"}
              </button>
            </div>
          )}

          <div className="mt-4 flex items-center gap-2">
            {presentation.action === "relogin" && (
              <Button size="sm" onClick={() => (window.location.href = "/login")}>
                Login Ulang
              </Button>
            )}

            {onRetry && (
              <Button size="sm" variant="secondary" onClick={onRetry}>
                <RefreshCw className="w-3.5 h-3.5 mr-1" /> Coba Lagi
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
