import { Link } from "react-router-dom";
import { FileQuestion, Home } from "lucide-react";
import { Button } from "../../../components/ui/Button";

export function NotFoundPage() {
  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-slate-50">
      <div className="max-w-md w-full bg-white rounded-2xl border border-gray-200/80 p-8 shadow-xs text-center">
        <div className="w-12 h-12 rounded-2xl bg-indigo-50 border border-indigo-200 text-indigo-600 flex items-center justify-center mx-auto mb-4">
          <FileQuestion className="w-6 h-6" />
        </div>

        <h1 className="text-xl font-bold text-gray-900">Halaman Tidak Ditemukan (404)</h1>
        <p className="text-xs text-gray-600 mt-2 leading-relaxed">
          Tautan yang Anda tuju tidak tersedia atau telah dipindahkan. Periksa kembali URL Anda.
        </p>

        <div className="mt-6 flex items-center justify-center">
          <Link to="/">
            <Button variant="primary" size="sm">
              <Home className="w-4 h-4 mr-1" /> Kembali ke Beranda
            </Button>
          </Link>
        </div>
      </div>
    </div>
  );
}
