import { useEffect, useState } from "react";
import { useAuthBlob } from "./useAuthBlob";
import { ImageOff, RotateCw } from "lucide-react";

interface AuthImageProps {
  src?: string | null;
  alt: string;
  className?: string;
  aspectRatio?: "square" | "video" | "auto";
}

export function AuthImage({ src, alt, className = "", aspectRatio = "square" }: AuthImageProps) {
  const { data, isLoading, isError, refetch } = useAuthBlob(src);
  const [objectUrl, setObjectUrl] = useState<string | null>(null);

  useEffect(() => {
    if (data?.blob) {
      const url = URL.createObjectURL(data.blob);
      setObjectUrl(url);
      return () => {
        URL.revokeObjectURL(url);
        setObjectUrl(null);
      };
    } else {
      setObjectUrl(null);
    }
  }, [data?.blob]);

  const aspectClass =
    aspectRatio === "square" ? "aspect-square" : aspectRatio === "video" ? "aspect-video" : "";

  if (!src) {
    return (
      <div
        className={`flex items-center justify-center bg-gray-100 text-gray-400 rounded-lg ${aspectClass} ${className}`}
      >
        <ImageOff className="w-8 h-8" />
      </div>
    );
  }

  if (isLoading) {
    return (
      <div
        className={`animate-pulse bg-gray-200 rounded-lg flex items-center justify-center ${aspectClass} ${className}`}
      >
        <span className="text-xs text-gray-400">Memuat foto...</span>
      </div>
    );
  }

  if (isError || (data && data.status !== 200)) {
    const errorText = data?.errorText ?? "Gagal memuat foto.";
    return (
      <div
        className={`flex flex-col items-center justify-center p-4 bg-gray-50 border border-gray-200 text-center rounded-lg ${aspectClass} ${className}`}
      >
        <ImageOff className="w-6 h-6 text-gray-400 mb-2" />
        <p className="text-xs text-gray-600 mb-2">{errorText}</p>
        {data?.status !== 410 && data?.status !== 404 && data?.status !== 403 && (
          <button
            type="button"
            onClick={() => refetch()}
            className="inline-flex items-center gap-1 text-xs text-indigo-600 hover:text-indigo-800 font-medium"
          >
            <RotateCw className="w-3 h-3" /> Coba lagi
          </button>
        )}
      </div>
    );
  }

  if (!objectUrl) {
    return (
      <div
        className={`flex items-center justify-center bg-gray-100 text-gray-400 rounded-lg ${aspectClass} ${className}`}
      >
        <ImageOff className="w-8 h-8" />
      </div>
    );
  }

  return <img src={objectUrl} alt={alt} className={`object-cover rounded-lg ${aspectClass} ${className}`} />;
}
