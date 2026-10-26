import { type TarEntry, buildTar } from "../export/tar";
import {
  type BrowserFileSystem,
  type FileSystemEntry,
  normalizePath,
} from "../filesystem";
import { pathHash } from "../navigation";

interface FileBrowserElements {
  browser: HTMLElement;
  app: HTMLElement;
  closeButton: HTMLButtonElement;
  openButton: HTMLButtonElement;
  newFileButton: HTMLButtonElement;
  newDirectoryButton: HTMLButtonElement;
  uploadButton: HTMLButtonElement;
  downloadDirectoryButton: HTMLButtonElement;
  uploadInput: HTMLInputElement;
  breadcrumbs: HTMLElement;
  fileList: HTMLElement;
  status: HTMLElement;
}

interface FileBrowserOptions {
  fileSystem: BrowserFileSystem;
  elements: FileBrowserElements;
  onOpen: () => void;
  onClose: () => void;
  onEditFile: (path: string) => Promise<void>;
  onWriteFile: (path: string) => Promise<void>;
  onRenameEntry: (
    oldPath: string,
    newPath: string,
    kind: FileSystemEntry["kind"],
  ) => void;
  onDeleteEntry: (path: string, kind: FileSystemEntry["kind"]) => Promise<void>;
}

export interface FileBrowserController {
  show(directory: string, error?: string): Promise<void>;
  hide(): void;
  refresh(): Promise<void>;
}

function joinPath(directory: string, name: string): string {
  return normalizePath(directory ? `${directory}/${name}` : name);
}

function validateName(input: string): string {
  const name = input.trim();
  if (!name || name === "." || name === ".." || /[\\/]/.test(name)) {
    throw new Error("Enter a single file or folder name without slashes.");
  }
  return name;
}

export function createFileBrowser({
  fileSystem,
  elements,
  onOpen,
  onClose,
  onEditFile,
  onWriteFile,
  onRenameEntry,
  onDeleteEntry,
}: FileBrowserOptions): FileBrowserController {
  let currentDirectory = "";

  const setStatus = (message: string, error = false): void => {
    elements.status.textContent = message;
    elements.status.classList.toggle("error", error);
  };

  const immediateChildren = (entries: FileSystemEntry[]): FileSystemEntry[] => {
    const prefix = currentDirectory ? `${currentDirectory}/` : "";
    return entries.filter((entry) => {
      if (!entry.path.startsWith(prefix)) {
        return false;
      }
      return !entry.path.slice(prefix.length).includes("/");
    });
  };

  const renderBreadcrumbs = (): void => {
    elements.breadcrumbs.replaceChildren();
    const parts = currentDirectory ? currentDirectory.split("/") : [];
    const names = ["Project", ...parts];
    for (let i = 0; i < names.length; i++) {
      if (i > 0) {
        const separator = document.createElement("span");
        separator.textContent = "/";
        separator.className = "breadcrumb-separator";
        elements.breadcrumbs.append(separator);
      }
      const link = document.createElement("a");
      link.textContent = names[i];
      link.href = pathHash(parts.slice(0, i).join("/"), "directory");
      elements.breadcrumbs.append(link);
    }
  };

  const renderEntry = (entry: FileSystemEntry): HTMLElement => {
    const row = document.createElement("div");
    row.className = "file-row";
    const pathParts = entry.path.split("/");
    const name = pathParts[pathParts.length - 1] || entry.path;
    const editable =
      entry.kind === "file" && entry.path.toLowerCase().endsWith(".scad");
    const openButton =
      entry.kind === "directory" || editable
        ? Object.assign(document.createElement("a"), {
            href: pathHash(entry.path, entry.kind),
          })
        : Object.assign(document.createElement("button"), { disabled: true });
    openButton.className = "file-row-open";
    const icon = document.createElement("span");
    icon.className = "file-row-icon";
    icon.textContent =
      entry.kind === "directory" ? "▸" : editable ? "SCAD" : "FILE";
    const label = document.createElement("span");
    label.className = "file-row-name";
    label.textContent = name;
    const detail = document.createElement("span");
    detail.className = "file-row-detail";
    detail.textContent =
      entry.kind === "directory" ? "Folder" : editable ? "Edit" : "";
    openButton.append(icon, label, detail);
    row.append(openButton);
    const actions = document.createElement("div");
    actions.className = "file-row-actions";
    const renameButton = document.createElement("button");
    renameButton.className = "file-row-rename";
    renameButton.textContent = "✎";
    renameButton.title = `Rename ${name}`;
    renameButton.setAttribute("aria-label", `Rename ${name}`);
    renameButton.addEventListener("click", () => {
      const input = window.prompt(`Rename ${name}:`, name);
      if (!input) {
        return;
      }
      void (async () => {
        try {
          let newName = validateName(input);
          if (
            entry.kind === "file" &&
            name.toLowerCase().endsWith(".scad") &&
            !newName.toLowerCase().endsWith(".scad")
          ) {
            newName += ".scad";
          }
          const newPath = joinPath(currentDirectory, newName);
          await fileSystem.renameEntry(entry.path, newPath);
          onRenameEntry(entry.path, newPath, entry.kind);
          await refresh();
          setStatus(`Renamed ${name} to ${newName}.`);
        } catch (error) {
          setStatus(
            error instanceof Error ? error.message : String(error),
            true,
          );
        }
      })();
    });
    const deleteButton = document.createElement("button");
    deleteButton.className = "file-row-delete";
    deleteButton.textContent = "×";
    deleteButton.title = `Delete ${name}`;
    deleteButton.setAttribute("aria-label", `Delete ${name}`);
    deleteButton.addEventListener("click", () => {
      const message =
        entry.kind === "directory"
          ? `Delete ${name} and everything inside it? This cannot be undone.`
          : `Delete ${name}? This cannot be undone.`;
      if (!window.confirm(message)) {
        return;
      }
      void (async () => {
        try {
          await fileSystem.deleteEntry(entry.path);
          await onDeleteEntry(entry.path, entry.kind);
          await refresh();
          setStatus(`Deleted ${name}.`);
        } catch (error) {
          setStatus(
            error instanceof Error ? error.message : String(error),
            true,
          );
        }
      })();
    });
    actions.append(renameButton, deleteButton);
    row.append(actions);
    return row;
  };

  let refreshRevision = 0;
  async function refresh(): Promise<void> {
    const revision = ++refreshRevision;
    try {
      const entries = await fileSystem.list();
      if (revision !== refreshRevision) {
        return;
      }
      const children = immediateChildren(entries);
      renderBreadcrumbs();
      elements.fileList.replaceChildren(...children.map(renderEntry));
      if (children.length === 0) {
        const empty = document.createElement("div");
        empty.className = "file-list-empty";
        empty.textContent = "This folder is empty.";
        elements.fileList.append(empty);
      }
      setStatus("");
    } catch (error) {
      if (revision === refreshRevision) {
        setStatus(error instanceof Error ? error.message : String(error), true);
      }
    }
  }

  elements.openButton.addEventListener("click", onOpen);
  elements.closeButton.addEventListener("click", onClose);
  elements.newFileButton.addEventListener("click", () => {
    const input = window.prompt("Name for the new .scad file:", "library.scad");
    if (!input) {
      return;
    }
    void (async () => {
      try {
        const validName = validateName(input);
        const name = validName.toLowerCase().endsWith(".scad")
          ? validName
          : `${validName}.scad`;
        const path = joinPath(currentDirectory, name);
        if ((await fileSystem.list()).some((entry) => entry.path === path)) {
          throw new Error(`${path} already exists.`);
        }
        await fileSystem.writeTextFile(path, "");
        await onEditFile(path);
      } catch (error) {
        setStatus(error instanceof Error ? error.message : String(error), true);
      }
    })();
  });
  elements.newDirectoryButton.addEventListener("click", () => {
    const name = window.prompt("Name for the new folder:", "lib");
    if (!name) {
      return;
    }
    void (async () => {
      try {
        const path = joinPath(currentDirectory, validateName(name));
        if ((await fileSystem.list()).some((entry) => entry.path === path)) {
          throw new Error(`${path} already exists.`);
        }
        await fileSystem.createDirectory(path);
        await refresh();
      } catch (error) {
        setStatus(error instanceof Error ? error.message : String(error), true);
      }
    })();
  });
  elements.uploadButton.addEventListener("click", () =>
    elements.uploadInput.click(),
  );
  elements.uploadInput.addEventListener("change", () => {
    const files = [...(elements.uploadInput.files || [])];
    const directory = currentDirectory;
    // Reset even after a cancellation or failure so the same file can be retried.
    elements.uploadInput.value = "";
    void (async () => {
      let uploaded = 0;
      let skipped = 0;
      let failure = "";
      try {
        for (const file of files) {
          const path = joinPath(directory, file.name);
          const existing = (await fileSystem.list()).find(
            (entry) => entry.path === path,
          );
          if (existing?.kind === "directory") {
            window.alert(
              `Cannot upload ${file.name}: a folder with that name already exists.`,
            );
            skipped++;
            continue;
          }
          if (
            existing &&
            !window.confirm(
              `Replace ${file.name}? A file with that name already exists in this folder.`,
            )
          ) {
            skipped++;
            continue;
          }
          await fileSystem.writeFile(
            path,
            new Uint8Array(await file.arrayBuffer()),
          );
          uploaded++;
          await onWriteFile(path);
        }
      } catch (error) {
        failure = error instanceof Error ? error.message : String(error);
      }
      if (currentDirectory === directory) {
        await refresh();
        setStatus(
          failure ||
            `${uploaded} file${uploaded === 1 ? "" : "s"} uploaded.${skipped ? ` ${skipped} skipped.` : ""}`,
          Boolean(failure),
        );
      }
    })();
  });
  elements.downloadDirectoryButton.addEventListener("click", () => {
    void (async () => {
      elements.downloadDirectoryButton.disabled = true;
      try {
        setStatus("Creating tar archive...");
        const prefix = currentDirectory ? `${currentDirectory}/` : "";
        const entries = (await fileSystem.list()).filter(
          (entry) =>
            entry.path.startsWith(prefix) && entry.path !== currentDirectory,
        );
        const tarEntries: TarEntry[] = entries.map((entry) => ({
          name: entry.path.slice(prefix.length),
          kind: entry.kind,
          ...(entry.kind === "file"
            ? { read: () => fileSystem.readFile(entry.path) }
            : {}),
        }));
        const archive = await buildTar(tarEntries);
        const directoryName = currentDirectory.split("/").pop();
        const archiveName = `${directoryName || "project"}.tar`;
        const url = URL.createObjectURL(archive);
        const link = document.createElement("a");
        link.href = url;
        link.download = archiveName;
        document.body.appendChild(link);
        link.click();
        link.remove();
        URL.revokeObjectURL(url);
        const fileCount = entries.filter(
          (entry) => entry.kind === "file",
        ).length;
        setStatus(
          `Downloaded ${archiveName} with ${fileCount} file${fileCount === 1 ? "" : "s"}.`,
        );
      } catch (error) {
        setStatus(error instanceof Error ? error.message : String(error), true);
      } finally {
        elements.downloadDirectoryButton.disabled = false;
      }
    })();
  });

  const controller: FileBrowserController = {
    async show(directory, error) {
      currentDirectory = directory;
      elements.app.hidden = true;
      elements.browser.hidden = false;
      await refresh();
      if (error && currentDirectory === directory) {
        setStatus(error, true);
      }
    },
    hide() {
      refreshRevision++;
      elements.browser.hidden = true;
      elements.app.hidden = false;
    },
    refresh,
  };
  return controller;
}
