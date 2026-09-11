import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "../ui/Button";

export interface PaginationProps {
    page: number;
    perPage?: number;
    total?: number;
    totalPages?: number;
    onPageChange: (page: number) => void;
    onPerPageChange?: (perPage: number) => void;
}

export function Pagination({
    page,
    perPage,
    total,
    totalPages,
    onPageChange,
    onPerPageChange,
}: PaginationProps) {
    const derivedPerPage = perPage ?? 20;
    const derivedTotal = total ?? (totalPages ? totalPages * derivedPerPage : 0);
    const derivedTotalPages = totalPages ?? Math.max(1, Math.ceil(derivedTotal / derivedPerPage));

    if (derivedTotal === 0 && !totalPages) return null;

    const start = (page - 1) * derivedPerPage + 1;
    const end = derivedTotal > 0 ? Math.min(page * derivedPerPage, derivedTotal) : page * derivedPerPage;

    return (
        <div className="flex flex-col sm:flex-row items-center justify-between gap-3 px-4 py-3 bg-white border-t border-gray-100 rounded-b-xl text-xs text-gray-500">
            <div>
                {derivedTotal > 0 ? (
                    <>
                        Menampilkan <span className="font-semibold text-gray-700">{start}</span>–
                        <span className="font-semibold text-gray-700">{end}</span> dari{" "}
                        <span className="font-semibold text-gray-700">{derivedTotal}</span> data
                    </>
                ) : (
                    <span>Halaman {page} dari {derivedTotalPages}</span>
                )}
            </div>

            <div className="flex items-center gap-2">
                {onPerPageChange && (
                    <div className="flex items-center gap-1.5 mr-2">
                        <span>Per halaman:</span>
                        <select
                            value={derivedPerPage}
                            onChange={(e) => onPerPageChange(Number(e.target.value))}
                            className="text-xs border border-gray-200 rounded px-1.5 py-0.5 bg-white text-gray-700"
                        >
                            <option value={10}>10</option>
                            <option value={20}>20</option>
                            <option value={50}>50</option>
                            <option value={100}>100</option>
                        </select>
                    </div>
                )}

                <span className="mr-2">
                    Halaman <span className="font-medium text-gray-700">{page}</span> dari{" "}
                    <span className="font-medium text-gray-700">{Math.max(1, derivedTotalPages)}</span>
                </span>

                <Button
                    size="sm"
                    variant="outline"
                    disabled={page <= 1}
                    onClick={() => onPageChange(page - 1)}
                    className="p-1.5"
                >
                    <ChevronLeft className="w-4 h-4" />
                </Button>

                <Button
                    size="sm"
                    variant="outline"
                    disabled={page >= derivedTotalPages}
                    onClick={() => onPageChange(page + 1)}
                    className="p-1.5"
                >
                    <ChevronRight className="w-4 h-4" />
                </Button>
            </div>
        </div>
    );
}
