import { formatPercent } from "../../lib/format";

interface SimilarityBarProps {
  similarity?: number | null;
  score?: number | null;
  threshold?: number | null;
  showText?: boolean;
  className?: string;
}

export function SimilarityBar({
  similarity,
  score,
  threshold,
  showText = true,
  className = "",
}: SimilarityBarProps) {
  const effectiveSim = similarity ?? score;

  if (effectiveSim === null || effectiveSim === undefined) {
    return <span className="text-xs text-gray-400">-</span>;
  }

  const th = threshold ?? 0.42;
  const isPass = effectiveSim >= th;
  const simPct = Math.min(100, Math.max(0, effectiveSim * 100));
  const thPct = Math.min(100, Math.max(0, th * 100));

  return (
    <div className={`space-y-1 ${className}`}>
      {showText && (
        <div className="flex items-center justify-between text-xs">
          <span className={`font-semibold ${isPass ? "text-emerald-700" : "text-rose-600"}`}>
            {formatPercent(effectiveSim)}
          </span>
          <span className="text-gray-400">
            ambang: <span className="font-medium text-gray-600">{formatPercent(th)}</span>
          </span>
        </div>
      )}

      {/* Progress Track */}
      <div className="relative h-2 w-full bg-gray-100 rounded-full overflow-hidden">
        {/* Fill */}
        <div
          className={`h-full transition-all duration-300 rounded-full ${
            isPass ? "bg-emerald-500" : "bg-rose-500"
          }`}
          style={{ width: `${simPct}%` }}
        />

        {/* Threshold marker */}
        <div
          className="absolute top-0 bottom-0 w-0.5 bg-gray-900/60 z-10"
          style={{ left: `${thPct}%` }}
          title={`Ambang batas: ${formatPercent(th)}`}
        />
      </div>
    </div>
  );
}
