import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.0.0/index.js";

export function generateId(size) {
    return uuidv4().replace(/-/g, '').substring(0, size);
}

export function Span(opts = {}) {
    // Required
    opts.traceId = typeof opts.traceId === "undefined" ? generateId(14) : opts.traceId;
    opts.id = typeof opts.id === "undefined" ? generateId(16) : opts.id;
    
    // Optional
    Object.keys(opts).forEach(key => {
        if (key !== "id" && key !== "traceId") {
            opts[key] = opts[key]
        }
    });

    return opts;
}