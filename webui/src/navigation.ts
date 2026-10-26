import { type FileSystemEntryKind, normalizePath } from "./filesystem";

export function pathHash(path: string, kind: FileSystemEntryKind): string {
  const encoded = path.split("/").map(encodeURIComponent).join("/");
  return `#/${encoded}${path && kind === "directory" ? "/" : ""}`;
}

export function hashPath(hash: string): string {
  const path = decodeURIComponent(hash.replace(/^#/, ""));
  return !path || /^\/+$/u.test(path) ? "" : normalizePath(path);
}

export function parentDirectory(path: string): string {
  return path.split("/").slice(0, -1).join("/");
}
