const DATABASE_NAME = "m3dscad_files";
const STORE_NAME = "entries";
const CONTENT_STORE_NAME = "contents";
const METADATA_STORE_NAME = "metadata";
const INITIALIZED_METADATA_KEY = "initialized";
const LEGACY_SOURCE_KEY = "m3dscad_source";
const DATABASE_VERSION = 4;

export type FileSystemEntryKind = "file" | "directory";

export interface FileSystemEntry {
  path: string;
  kind: FileSystemEntryKind;
}

interface StoredFileContent {
  path: string;
  content: Uint8Array;
}

interface MetadataEntry<T> {
  key: string;
  value: T;
}

const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder();

function fileBytes(content: unknown): Uint8Array {
  if (content instanceof Uint8Array) {
    return new Uint8Array(content);
  }
  if (content instanceof ArrayBuffer) {
    return new Uint8Array(content.slice(0));
  }
  if (typeof content === "string") {
    return textEncoder.encode(content);
  }
  return new Uint8Array();
}

function requestResult<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result);
    request.onerror = () =>
      reject(request.error || new Error("IndexedDB request failed"));
  });
}

function transactionComplete(transaction: IDBTransaction): Promise<void> {
  return new Promise((resolve, reject) => {
    transaction.oncomplete = () => resolve();
    transaction.onerror = () =>
      reject(transaction.error || new Error("IndexedDB transaction failed"));
    transaction.onabort = () =>
      reject(transaction.error || new Error("IndexedDB transaction aborted"));
  });
}

function parentPaths(path: string): string[] {
  const parts = path.split("/");
  const result: string[] = [];
  for (let i = 1; i < parts.length; i++) {
    result.push(parts.slice(0, i).join("/"));
  }
  return result;
}

export function normalizePath(input: string): string {
  const parts: string[] = [];
  for (const part of input.replaceAll("\\", "/").split("/")) {
    if (!part || part === ".") {
      continue;
    }
    if (part === "..") {
      if (parts.length === 0) {
        throw new Error("A path cannot leave the project root.");
      }
      parts.pop();
      continue;
    }
    parts.push(part);
  }
  const path = parts.join("/");
  if (!path) {
    throw new Error("Path cannot be empty.");
  }
  return path;
}

export class BrowserFileSystem {
  private constructor(private readonly database: IDBDatabase) {}

  static async open(defaultSource: string): Promise<BrowserFileSystem> {
    const request = indexedDB.open(DATABASE_NAME, DATABASE_VERSION);
    request.onupgradeneeded = (event) => {
      const database = request.result;
      let store: IDBObjectStore;
      if (!database.objectStoreNames.contains(STORE_NAME)) {
        store = database.createObjectStore(STORE_NAME, { keyPath: "path" });
      } else {
        store = request.transaction?.objectStore(STORE_NAME) as IDBObjectStore;
      }
      let contentStore: IDBObjectStore;
      if (!database.objectStoreNames.contains(CONTENT_STORE_NAME)) {
        contentStore = database.createObjectStore(CONTENT_STORE_NAME, {
          keyPath: "path",
        });
      } else {
        contentStore = request.transaction?.objectStore(
          CONTENT_STORE_NAME,
        ) as IDBObjectStore;
      }
      if (event.oldVersion < 4) {
        const cursorRequest = store.openCursor();
        cursorRequest.onsuccess = () => {
          const cursor = cursorRequest.result;
          if (!cursor) {
            return;
          }
          const entry = cursor.value as FileSystemEntry & {
            content?: unknown;
          };
          if (entry.kind === "file") {
            contentStore.put({
              path: entry.path,
              content: fileBytes(entry.content),
            } satisfies StoredFileContent);
            cursor.update({
              path: entry.path,
              kind: entry.kind,
            } satisfies FileSystemEntry);
          }
          cursor.continue();
        };
      }
      let metadataStore: IDBObjectStore;
      if (!database.objectStoreNames.contains(METADATA_STORE_NAME)) {
        metadataStore = database.createObjectStore(METADATA_STORE_NAME, {
          keyPath: "key",
        });
      } else {
        metadataStore = request.transaction?.objectStore(
          METADATA_STORE_NAME,
        ) as IDBObjectStore;
      }
      // Existing databases have already gone through project initialization.
      // Remember that even if the user intentionally deleted every entry.
      if (event.oldVersion > 0) {
        metadataStore.put({
          key: INITIALIZED_METADATA_KEY,
          value: true,
        } satisfies MetadataEntry<boolean>);
      }
    };
    const database = await requestResult(request);
    const result = new BrowserFileSystem(database);
    if (!(await result.getMetadata<boolean>(INITIALIZED_METADATA_KEY))) {
      const legacy = window.localStorage.getItem(LEGACY_SOURCE_KEY);
      if ((await result.list()).length === 0) {
        await result.writeTextFile(
          "main.scad",
          legacy && legacy.trim().length > 0 ? legacy : defaultSource,
        );
      }
      await result.setMetadata(INITIALIZED_METADATA_KEY, true);
    }
    window.localStorage.removeItem(LEGACY_SOURCE_KEY);
    return result;
  }

  async list(): Promise<FileSystemEntry[]> {
    const transaction = this.database.transaction(STORE_NAME, "readonly");
    const entries = await requestResult(
      transaction.objectStore(STORE_NAME).getAll() as IDBRequest<
        FileSystemEntry[]
      >,
    );
    return entries.sort((a, b) => a.path.localeCompare(b.path));
  }

  async readFile(pathInput: string): Promise<Uint8Array> {
    const path = normalizePath(pathInput);
    const transaction = this.database.transaction(
      CONTENT_STORE_NAME,
      "readonly",
    );
    const stored = await requestResult(
      transaction.objectStore(CONTENT_STORE_NAME).get(path) as IDBRequest<
        StoredFileContent | undefined
      >,
    );
    if (!stored) {
      throw new Error(`File not found: ${path}`);
    }
    return fileBytes(stored.content);
  }

  async readTextFile(pathInput: string): Promise<string> {
    return textDecoder.decode(await this.readFile(pathInput));
  }

  async writeFile(pathInput: string, content: Uint8Array): Promise<void> {
    const path = normalizePath(pathInput);
    const data = new Uint8Array(content);
    const transaction = this.database.transaction(
      [STORE_NAME, CONTENT_STORE_NAME],
      "readwrite",
    );
    const store = transaction.objectStore(STORE_NAME);
    for (const parent of parentPaths(path)) {
      store.put({ path: parent, kind: "directory" } satisfies FileSystemEntry);
    }
    store.put({ path, kind: "file" } satisfies FileSystemEntry);
    transaction
      .objectStore(CONTENT_STORE_NAME)
      .put({ path, content: data } satisfies StoredFileContent);
    await transactionComplete(transaction);
  }

  async writeTextFile(pathInput: string, content: string): Promise<void> {
    await this.writeFile(pathInput, textEncoder.encode(content));
  }

  async createDirectory(pathInput: string): Promise<void> {
    const path = normalizePath(pathInput);
    const transaction = this.database.transaction(STORE_NAME, "readwrite");
    const store = transaction.objectStore(STORE_NAME);
    for (const parent of [...parentPaths(path), path]) {
      store.put({ path: parent, kind: "directory" } satisfies FileSystemEntry);
    }
    await transactionComplete(transaction);
  }

  async renameEntry(oldPathInput: string, newPathInput: string): Promise<void> {
    const oldPath = normalizePath(oldPathInput);
    const newPath = normalizePath(newPathInput);
    if (oldPath === newPath) {
      return;
    }
    const entries = await this.list();
    const source = entries.find((entry) => entry.path === oldPath);
    if (!source) {
      throw new Error(`Path not found: ${oldPath}`);
    }
    if (source.kind === "directory" && newPath.startsWith(`${oldPath}/`)) {
      throw new Error("A folder cannot be moved inside itself.");
    }
    const affected = entries.filter(
      (entry) =>
        entry.path === oldPath ||
        (source.kind === "directory" && entry.path.startsWith(`${oldPath}/`)),
    );
    const affectedPaths = new Set(affected.map((entry) => entry.path));
    const moved = affected.map((entry) => ({
      entry,
      newPath: `${newPath}${entry.path.slice(oldPath.length)}`,
    }));
    if (
      moved.some(({ newPath: target }) =>
        entries.some(
          (entry) => entry.path === target && !affectedPaths.has(entry.path),
        ),
      )
    ) {
      throw new Error(`${newPath} already exists.`);
    }

    const transaction = this.database.transaction(
      [STORE_NAME, CONTENT_STORE_NAME],
      "readwrite",
    );
    const store = transaction.objectStore(STORE_NAME);
    const contentStore = transaction.objectStore(CONTENT_STORE_NAME);
    let moveError: Error | null = null;
    for (const entry of affected) {
      store.delete(entry.path);
    }
    for (const parent of parentPaths(newPath)) {
      store.put({ path: parent, kind: "directory" } satisfies FileSystemEntry);
    }
    for (const item of moved) {
      store.put({
        path: item.newPath,
        kind: item.entry.kind,
      } satisfies FileSystemEntry);
      if (item.entry.kind === "file") {
        const contentRequest = contentStore.get(item.entry.path) as IDBRequest<
          StoredFileContent | undefined
        >;
        contentRequest.onsuccess = () => {
          if (!contentRequest.result) {
            moveError = new Error(`File not found: ${item.entry.path}`);
            transaction.abort();
            return;
          }
          contentStore.put({
            path: item.newPath,
            content: contentRequest.result.content,
          } satisfies StoredFileContent);
          contentStore.delete(item.entry.path);
        };
      }
    }
    try {
      await transactionComplete(transaction);
    } catch (error) {
      throw moveError || error;
    }
  }

  async deleteEntry(pathInput: string): Promise<void> {
    const path = normalizePath(pathInput);
    const entries = await this.list();
    const source = entries.find((entry) => entry.path === path);
    if (!source) {
      throw new Error(`Path not found: ${path}`);
    }
    const affected = entries.filter(
      (entry) =>
        entry.path === path ||
        (source.kind === "directory" && entry.path.startsWith(`${path}/`)),
    );
    const transaction = this.database.transaction(
      [STORE_NAME, CONTENT_STORE_NAME],
      "readwrite",
    );
    const store = transaction.objectStore(STORE_NAME);
    const contentStore = transaction.objectStore(CONTENT_STORE_NAME);
    for (const entry of affected) {
      store.delete(entry.path);
      if (entry.kind === "file") {
        contentStore.delete(entry.path);
      }
    }
    await transactionComplete(transaction);
  }

  private async getMetadata<T>(key: string): Promise<T | undefined> {
    const transaction = this.database.transaction(
      METADATA_STORE_NAME,
      "readonly",
    );
    const entry = await requestResult(
      transaction.objectStore(METADATA_STORE_NAME).get(key) as IDBRequest<
        MetadataEntry<T> | undefined
      >,
    );
    return entry?.value;
  }

  private async setMetadata<T>(key: string, value: T): Promise<void> {
    const transaction = this.database.transaction(
      METADATA_STORE_NAME,
      "readwrite",
    );
    transaction
      .objectStore(METADATA_STORE_NAME)
      .put({ key, value } satisfies MetadataEntry<T>);
    await transactionComplete(transaction);
  }
}
