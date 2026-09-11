import DOMPurify from "dompurify";

/**
 * Basic Markdown to HTML converter with strict DOMPurify XSS sanitization.
 * Used for rendering PDP Biometric Consent documents safely.
 */
export function renderMarkdownSafely(markdownText: string): string {
    if (!markdownText) return "";

    // 1. Escaping basic HTML entities
    const lines = markdownText.split("\n");
    const htmlLines: string[] = [];

    let inList = false;

    for (let i = 0; i < lines.length; i++) {
        const rawLine = lines[i];
        if (rawLine === undefined) continue;
        const trimmed = rawLine.trim();

        if (!trimmed) {
            if (inList) {
                htmlLines.push("</ul>");
                inList = false;
            }
            continue;
        }

        // Headers
        if (trimmed.startsWith("### ")) {
            if (inList) {
                htmlLines.push("</ul>");
                inList = false;
            }
            htmlLines.push(
                `<h3 class="text-base font-semibold text-slate-900 mt-4 mb-2">${parseInline(trimmed.slice(4))}</h3>`,
            );
            continue;
        }
        if (trimmed.startsWith("## ")) {
            if (inList) {
                htmlLines.push("</ul>");
                inList = false;
            }
            htmlLines.push(
                `<h2 class="text-lg font-bold text-slate-900 mt-6 mb-2 border-b pb-1">${parseInline(trimmed.slice(3))}</h2>`,
            );
            continue;
        }
        if (trimmed.startsWith("# ")) {
            if (inList) {
                htmlLines.push("</ul>");
                inList = false;
            }
            htmlLines.push(
                `<h1 class="text-xl font-bold text-slate-900 mt-6 mb-3">${parseInline(trimmed.slice(2))}</h1>`,
            );
            continue;
        }

        // Unordered List items
        if (trimmed.startsWith("- ") || trimmed.startsWith("* ")) {
            if (!inList) {
                htmlLines.push('<ul class="list-disc list-inside space-y-1 my-2 text-slate-700">');
                inList = true;
            }
            htmlLines.push(`<li>${parseInline(trimmed.slice(2))}</li>`);
            continue;
        }

        // Numbered lists
        const matchNum = trimmed.match(/^(\d+)\.\s+(.*)/);
        if (matchNum && matchNum[1] && matchNum[2]) {
            if (inList) {
                htmlLines.push("</ul>");
                inList = false;
            }
            htmlLines.push(
                `<p class="my-1.5 text-slate-700 font-medium">${matchNum[1]}. ${parseInline(matchNum[2])}</p>`,
            );
            continue;
        }

        // Regular paragraph
        if (inList) {
            htmlLines.push("</ul>");
            inList = false;
        }
        htmlLines.push(`<p class="my-2 leading-relaxed text-slate-700 text-sm">${parseInline(trimmed)}</p>`);
    }

    if (inList) {
        htmlLines.push("</ul>");
    }

    const rawHtml = htmlLines.join("\n");

    // Sanitize strictly with DOMPurify
    return DOMPurify.sanitize(rawHtml, {
        ALLOWED_TAGS: [
            "h1",
            "h2",
            "h3",
            "h4",
            "p",
            "ul",
            "ol",
            "li",
            "strong",
            "em",
            "b",
            "i",
            "u",
            "code",
            "span",
            "br",
        ],
        ALLOWED_ATTR: ["class"],
    });
}

function parseInline(text: string): string {
    let res = text;
    // Bold: **bold** or __bold__
    res = res.replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>");
    res = res.replace(/__(.*?)__/g, "<strong>$1</strong>");
    // Italic: *italic* or _italic_
    res = res.replace(/\*(.*?)\*/g, "<em>$1</em>");
    res = res.replace(/_(.*?)_/g, "<em>$1</em>");
    // Inline code: `code`
    res = res.replace(
        /`([^`]+)`/g,
        '<code class="px-1 py-0.5 bg-slate-100 rounded text-xs text-indigo-700">$1</code>',
    );
    return res;
}
