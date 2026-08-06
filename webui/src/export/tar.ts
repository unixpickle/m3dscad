const TAR_BLOCK_SIZE = 512;
const textEncoder = new TextEncoder();

export interface TarEntry {
  name: string;
  kind: "file" | "directory";
  read?: () => Promise<Uint8Array>;
}

function encodedLength(value: string): number {
  return textEncoder.encode(value).byteLength;
}

function writeString(
  target: Uint8Array,
  offset: number,
  length: number,
  value: string,
): void {
  const bytes = textEncoder.encode(value);
  if (bytes.byteLength > length) {
    throw new Error(`Tar field is too long: ${value}`);
  }
  target.set(bytes, offset);
}

function writeOctal(
  target: Uint8Array,
  offset: number,
  length: number,
  value: number,
): void {
  const octal = Math.max(0, Math.floor(value)).toString(8);
  if (octal.length > length - 1) {
    throw new Error(`Tar numeric field is too large: ${value}`);
  }
  writeString(target, offset, length - 1, octal.padStart(length - 1, "0"));
  target[offset + length - 1] = 0;
}

function splitTarPath(path: string): { name: string; prefix: string } {
  if (!path || path.includes("\0")) {
    throw new Error(`Invalid tar path: ${path || "<empty>"}`);
  }
  if (encodedLength(path) <= 100) {
    return { name: path, prefix: "" };
  }
  for (let index = path.lastIndexOf("/"); index > 0; ) {
    const prefix = path.slice(0, index);
    const name = path.slice(index + 1);
    if (encodedLength(prefix) <= 155 && encodedLength(name) <= 100) {
      return { name, prefix };
    }
    index = path.lastIndexOf("/", index - 1);
  }
  throw new Error(`Path is too long for a tar archive: ${path}`);
}

function tarHeader(
  name: string,
  size: number,
  directory: boolean,
): Uint8Array<ArrayBuffer> {
  const header = new Uint8Array(TAR_BLOCK_SIZE);
  const fields = splitTarPath(name);
  writeString(header, 0, 100, fields.name);
  writeOctal(header, 100, 8, directory ? 0o755 : 0o644);
  writeOctal(header, 108, 8, 0);
  writeOctal(header, 116, 8, 0);
  writeOctal(header, 124, 12, directory ? 0 : size);
  writeOctal(header, 136, 12, Math.floor(Date.now() / 1000));
  header.fill(0x20, 148, 156);
  header[156] = directory ? "5".charCodeAt(0) : "0".charCodeAt(0);
  writeString(header, 257, 6, "ustar\0");
  writeString(header, 263, 2, "00");
  writeString(header, 345, 155, fields.prefix);

  let checksum = 0;
  for (const byte of header) {
    checksum += byte;
  }
  const checksumOctal = checksum.toString(8).padStart(6, "0");
  if (checksumOctal.length > 6) {
    throw new Error("Tar header checksum overflow");
  }
  writeString(header, 148, 6, checksumOctal);
  header[154] = 0;
  header[155] = 0x20;
  return header;
}

export async function buildTar(entries: TarEntry[]): Promise<Blob> {
  const parts: BlobPart[] = [];
  for (const entry of entries) {
    if (entry.kind === "directory") {
      parts.push(tarHeader(entry.name, 0, true));
      continue;
    }
    if (!entry.read) {
      throw new Error(`Missing content reader for ${entry.name}`);
    }
    const source = await entry.read();
    const contents = new Uint8Array(source.byteLength);
    contents.set(source);
    parts.push(tarHeader(entry.name, contents.byteLength, false), contents);
    const padding =
      (TAR_BLOCK_SIZE - (contents.byteLength % TAR_BLOCK_SIZE)) %
      TAR_BLOCK_SIZE;
    if (padding > 0) {
      parts.push(new Uint8Array(padding));
    }
  }
  parts.push(new Uint8Array(TAR_BLOCK_SIZE * 2));
  return new Blob(parts, { type: "application/x-tar" });
}
