/**
 * Real-time Canvas Luminance Advisory (D25).
 * Non-blocking visual guidance for low-light or back-lit environments.
 */

export interface LuminanceResult {
    averageLuminance: number; // 0 - 255
    advisoryCode: "too_dark" | "too_bright" | null;
    advisoryText: string | null;
}

const SAMPLE_SIZE = 64;

/**
 * Computes average luminance across the current video frame using a small off-screen canvas.
 */
export function evaluateVideoLuminance(
    videoElement: HTMLVideoElement,
    offscreenCanvas?: HTMLCanvasElement,
): LuminanceResult {
    if (
        !videoElement ||
        videoElement.readyState < HTMLMediaElement.HAVE_CURRENT_DATA ||
        videoElement.videoWidth === 0
    ) {
        return {
            averageLuminance: 128,
            advisoryCode: null,
            advisoryText: null,
        };
    }

    const canvas = offscreenCanvas ?? document.createElement("canvas");
    canvas.width = SAMPLE_SIZE;
    canvas.height = SAMPLE_SIZE;

    const ctx = canvas.getContext("2d", { willReadFrequently: true });
    if (!ctx) {
        return {
            averageLuminance: 128,
            advisoryCode: null,
            advisoryText: null,
        };
    }

    // Draw scaled down thumbnail
    ctx.drawImage(videoElement, 0, 0, SAMPLE_SIZE, SAMPLE_SIZE);

    const imgData = ctx.getImageData(0, 0, SAMPLE_SIZE, SAMPLE_SIZE);
    const data = imgData.data;
    let totalLuminance = 0;
    const pixelCount = SAMPLE_SIZE * SAMPLE_SIZE;

    // Sample every pixel in 64x64 (4096 pixels)
    for (let i = 0; i < data.length; i += 4) {
        const r = data[i] ?? 0;
        const g = data[i + 1] ?? 0;
        const b = data[i + 2] ?? 0;
        // ITU-R BT.601 standard relative luminance formula
        const lum = 0.299 * r + 0.587 * g + 0.114 * b;
        totalLuminance += lum;
    }

    const avgLum = Math.round(totalLuminance / pixelCount);

    if (avgLum < 42) {
        return {
            averageLuminance: avgLum,
            advisoryCode: "too_dark",
            advisoryText: "Pencahayaan redup. Cari lokasi yang lebih terang.",
        };
    }

    if (avgLum > 225) {
        return {
            averageLuminance: avgLum,
            advisoryCode: "too_bright",
            advisoryText: "Pencahayaan terlalu silau. Hindari lampu sorot langsung atau backlight.",
        };
    }

    return {
        averageLuminance: avgLum,
        advisoryCode: null,
        advisoryText: null,
    };
}
