// Human-friendly Indonesian translations for face quality and guidance hints.
// Canonical source: docs/api/hints.json.

export interface HintItem {
  code: string;
  meaning: string;
  blocking: boolean;
  text: string;
}

export const FACE_HINTS: Record<string, HintItem> = {
  no_face: {
    code: "no_face",
    meaning: "Tidak ada wajah terdeteksi",
    blocking: true,
    text: "Wajah tidak terdeteksi. Posisikan wajah di dalam bingkai.",
  },
  multiple_faces: {
    code: "multiple_faces",
    meaning: "Lebih dari satu wajah dalam frame",
    blocking: true,
    text: "Terdeteksi lebih dari satu wajah. Pastikan hanya Anda dalam bingkai.",
  },
  low_detection_confidence: {
    code: "low_detection_confidence",
    meaning: "det_score di bawah ambang",
    blocking: true,
    text: "Wajah kurang jelas. Hadapkan wajah lurus ke kamera.",
  },
  too_blurry: {
    code: "too_blurry",
    meaning: "Variance of Laplacian rendah",
    blocking: true,
    text: "Foto buram. Tahan perangkat agar tidak bergerak.",
  },
  too_dark: {
    code: "too_dark",
    meaning: "Kecerahan crop wajah rendah",
    blocking: true,
    text: "Terlalu gelap. Cari tempat yang lebih terang.",
  },
  too_bright: {
    code: "too_bright",
    meaning: "Kecerahan terlalu tinggi / overexposed",
    blocking: true,
    text: "Terlalu terang. Hindari cahaya langsung dari belakang.",
  },
  face_too_small: {
    code: "face_too_small",
    meaning: "Tinggi bbox / tinggi frame di bawah ambang",
    blocking: true,
    text: "Wajah terlalu jauh. Dekatkan wajah ke kamera.",
  },
  head_turned: {
    code: "head_turned",
    meaning: "Proxy yaw melebihi ambang",
    blocking: true,
    text: "Hadapkan wajah lurus ke kamera.",
  },
  head_tilted: {
    code: "head_tilted",
    meaning: "Proxy pitch melebihi ambang",
    blocking: true,
    text: "Jangan menunduk atau mendongak.",
  },
};

export function getHintText(code: string): string {
  return FACE_HINTS[code]?.text ?? code;
}
