export function TableSkeleton({ rows = 5, cols = 4 }: { rows?: number; cols?: number }) {
  return (
    <div className="w-full animate-pulse">
      <div className="h-10 bg-gray-100 rounded-t-lg mb-2" />
      <div className="space-y-2">
        {Array.from({ length: rows }).map((_, i) => (
          <div key={i} className="flex gap-4 p-3 bg-gray-50/70 rounded border border-gray-100">
            {Array.from({ length: cols }).map((_, j) => (
              <div
                key={j}
                className="h-4 bg-gray-200/80 rounded"
                style={{ width: `${Math.max(40, 100 / cols - 5)}%` }}
              />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

export function CardSkeleton() {
  return (
    <div className="animate-pulse bg-white rounded-xl border border-gray-200/80 p-6 space-y-4">
      <div className="h-5 bg-gray-200 rounded w-1/3" />
      <div className="h-4 bg-gray-100 rounded w-2/3" />
      <div className="h-12 bg-gray-50 rounded" />
    </div>
  );
}
