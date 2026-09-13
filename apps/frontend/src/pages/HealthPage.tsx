import { useQuery } from "@tanstack/react-query";
import { api, ApiError } from "../lib/api";
import type { ReadyzData } from "../types/api";

// Fase 0 § 2.7: "Satu halaman `/` yang memanggil GET /readyz dan
// menampilkan status — pembuktian bahwa proxy & CORS jalan." Nothing about
// this page is meant to survive past Fase 5 wiring the real app shell.
export function HealthPage() {
  const { data, error, isLoading } = useQuery<ReadyzData, ApiError>({
    queryKey: ["readyz"],
    queryFn: () => api.get<ReadyzData>("/readyz"),
    refetchInterval: 5000,
  });

  return (
    <main className="min-h-screen bg-slate-950 text-slate-100 flex items-center justify-center p-6">
      <div className="max-w-md w-full space-y-4">
        <h1 className="text-2xl font-semibold">Faceclock — Fase 0</h1>
        <p className="text-slate-400 text-sm">
          Halaman ini memanggil <code className="text-slate-300">GET /readyz</code> lewat proxy Vite,
          membuktikan CORS dan wiring compose berjalan.
        </p>

        {isLoading && <p className="text-slate-400">Memeriksa status...</p>}

        {error && (
          <div className="rounded border border-red-800 bg-red-950/50 p-4 text-red-200">
            <p className="font-medium">Gagal memuat status</p>
            <p className="text-sm">{error.message}</p>
          </div>
        )}

        {data && (
          <div
            className={`rounded border p-4 ${
              data.status === "ok"
                ? "border-emerald-800 bg-emerald-950/50 text-emerald-200"
                : "border-amber-800 bg-amber-950/50 text-amber-200"
            }`}
          >
            <p className="font-medium">
              Status: <span className="uppercase">{data.status}</span>
            </p>
            <ul className="mt-2 text-sm space-y-1">
              <li>
                database: {data.checks.database.status}
                {data.checks.database.latency_ms !== undefined && ` (${data.checks.database.latency_ms}ms)`}
              </li>
              <li>
                inference: {data.checks.inference.status}
                {data.checks.inference.latency_ms !== undefined && ` (${data.checks.inference.latency_ms}ms)`}
              </li>
            </ul>
          </div>
        )}
      </div>
    </main>
  );
}
