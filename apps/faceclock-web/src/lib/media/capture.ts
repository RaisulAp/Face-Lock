/**
 * Camera Frame Capture & Adaptive JPEG Compression (D24).
 *
 * CRITICAL ARCHITECTURAL REQUIREMENT:
 * The live preview `<video>` is mirrored via CSS (`transform: scaleX(-1)`) for user intuition.
 * However, the canvas capture MUST NEVER BE MIRRORED (`scale(-1, 1)` must NOT be applied to ctx).
 * Mirroring the captured frame produces inverted facial embeddings that degrade matching accuracy.
 */

export interface CaptureOptions {
    maxBytes?: number; // default 800 KB (819,200 bytes)
    maxWidth?: number; // default 1280
    initialQuality?: number; // default 0.85
    mimeType?: string; // default "image/jpeg"
}

export interface CaptureResult {
    blob: Blob;
    dataUrl: string;
    width: number;
    height: number;
    sizeBytes: number;
}

/**
 * Converts canvas to Blob using a Promise.
 */
function canvasToBlob(canvas: HTMLCanvasElement, mimeType: string, quality: number): Promise<Blob | null> {
    return new Promise((resolve) => {
        canvas.toBlob((blob) => resolve(blob), mimeType, quality);
    });
}

/**
 * Captures the current frame from a `<video>` element with adaptive JPEG compression.
 */
export async function captureVideoFrame(
    videoElement: HTMLVideoElement,
    options: CaptureOptions = {},
): Promise<CaptureResult> {
    const maxBytes = options.maxBytes ?? 800 * 1024; // 800 KB limit
    const maxWidth = options.maxWidth ?? 1280;
    const mimeType = options.mimeType ?? "image/jpeg";
    let quality = options.initialQuality ?? 0.85;

    const naturalWidth = videoElement.videoWidth || 640;
    const naturalHeight = videoElement.videoHeight || 480;

    // Determine initial scale
    let targetWidth = naturalWidth;
    let targetHeight = naturalHeight;

    if (targetWidth > maxWidth) {
        const ratio = maxWidth / targetWidth;
        targetWidth = Math.round(targetWidth * ratio);
        targetHeight = Math.round(targetHeight * ratio);
    }

    const canvas = document.createElement("canvas");
    canvas.width = targetWidth;
    canvas.height = targetHeight;
    const ctx = canvas.getContext("2d", { willReadFrequently: true });

    if (!ctx) {
        throw new Error("Gagal menginisialisasi konteks 2D canvas.");
    }

    // Draw frame directly WITHOUT ANY HORIZONTAL FLIP / MIRRORING!
    ctx.drawImage(videoElement, 0, 0, targetWidth, targetHeight);

    // Step 1: initial quality blob
    let blob = await canvasToBlob(canvas, mimeType, quality);
    if (!blob) {
        throw new Error("Gagal mengonversi canvas ke Blob gambar.");
    }

    // Step 2: Quality degradation ladder if size exceeds maxBytes
    const qualitySteps = [0.75, 0.65, 0.55];
    let stepIdx = 0;

    while (blob.size > maxBytes && stepIdx < qualitySteps.length) {
        quality = qualitySteps[stepIdx] ?? 0.75;
        blob = await canvasToBlob(canvas, mimeType, quality);
        if (!blob) break;
        stepIdx++;
    }

    // Step 3: Dimensional downscaling if still too large
    if (blob && blob.size > maxBytes && targetWidth > 800) {
        const downscaleRatio = 800 / targetWidth;
        const downWidth = Math.round(targetWidth * downscaleRatio);
        const downHeight = Math.round(targetHeight * downscaleRatio);

        const downCanvas = document.createElement("canvas");
        downCanvas.width = downWidth;
        downCanvas.height = downHeight;
        const downCtx = downCanvas.getContext("2d");
        if (downCtx) {
            downCtx.drawImage(canvas, 0, 0, downWidth, downHeight);
            const smallerBlob = await canvasToBlob(downCanvas, mimeType, 0.75);
            if (smallerBlob) {
                blob = smallerBlob;
                targetWidth = downWidth;
                targetHeight = downHeight;
            }
        }
    }

    if (!blob) {
        throw new Error("Gagal menghasilkan file foto.");
    }

    const dataUrl = URL.createObjectURL(blob);

    return {
        blob,
        dataUrl,
        width: targetWidth,
        height: targetHeight,
        sizeBytes: blob.size,
    };
}
