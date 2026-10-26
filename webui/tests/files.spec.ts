import { type Page, expect, test } from "@playwright/test";

// Exercise the real editor, IndexedDB, history and WebGL renderer without
// requiring a Go/WASM build. Delayed worker replies cover file-switch races.
test.beforeEach(async ({ context }) => {
  await context.addInitScript(() => {
    const settings = window as unknown as {
      compileDelay: number;
      formatDelay: number;
    };
    settings.compileDelay = 0;
    settings.formatDelay = 0;
    window.Worker = class {
      onmessage?: (event: { data: unknown }) => void;
      postMessage(message: { type: string; id: number; filename: string }) {
        if (message.type === "init") {
          setTimeout(() => this.onmessage?.({ data: { type: "ready" } }), 0);
        } else if (message.type === "compile") {
          setTimeout(
            () =>
              this.onmessage?.({
                data: {
                  type: "result",
                  id: message.id,
                  ok: true,
                  positions: [0, 0, 0, 2, 0, 0, 0, 2, 0],
                  normals: [0, 0, 1, 0, 0, 1, 0, 0, 1],
                  bounds: { min: [0, 0, 0], max: [2, 2, 0] },
                },
              }),
            settings.compileDelay,
          );
        } else if (message.type === "format") {
          setTimeout(
            () =>
              this.onmessage?.({
                data: {
                  type: "format_result",
                  id: message.id,
                  ok: true,
                  code: "cube(1);\n",
                },
              }),
            settings.formatDelay,
          );
        }
      }
      // Intentionally allow late messages to exercise the client's stale-worker guard.
      terminate() {}
    } as unknown as typeof Worker;
  });
});

async function seed(page: Page, files: Record<string, string>) {
  await page.goto("/");
  await expect(page).toHaveURL(/#\/main\.scad$/);
  await expect(page.locator("#editorFilename")).toHaveText("main.scad");
  await page.evaluate(async (files) => {
    const modulePath = "/src/filesystem.ts";
    const { BrowserFileSystem } = await import(modulePath);
    const fs = await BrowserFileSystem.open("");
    for (const [path, source] of Object.entries(files)) {
      await fs.writeTextFile(path, source);
    }
  }, files);
}

async function openHash(page: Page, hash: string) {
  await page.evaluate((hash) => {
    window.location.hash = hash;
  }, hash);
}

test("first visit, folder breadcrumbs, browser history, and reload stay in sync", async ({
  page,
}) => {
  await seed(page, { "parts/one.scad": "cube(1);" });
  await expect(page).toHaveURL(/#\/main\.scad$/);
  await page.getByRole("button", { name: "Files", exact: true }).click();
  await expect(page).toHaveURL(/#\/$/);
  await page.locator('a[href="#/parts/"]').click();
  await expect(page.locator("#fileBreadcrumbs")).toHaveText("Project/parts");
  await page.locator('a[href="#/parts/one.scad"]').click();
  await expect(page.locator("#editorFilename")).toHaveText("parts/one.scad");
  await page.getByRole("button", { name: "Files", exact: true }).click();
  await expect(page).toHaveURL(/#\/parts\/$/);
  await page.goBack();
  await expect(page.locator("#app")).toBeVisible();
  await expect(page.locator("#editorFilename")).toHaveText("parts/one.scad");
  await page.goBack();
  await expect(page.locator("#fileBrowser")).toBeVisible();
  await page.goBack();
  await expect(page).toHaveURL(/#\/$/);
  await page.goForward();
  await expect(page.locator("#fileBreadcrumbs")).toHaveText("Project/parts");
  await page.reload();
  await expect(page.locator("#fileBrowser")).toBeVisible();
  await expect(page.locator("#fileBreadcrumbs")).toHaveText("Project/parts");
  await page.getByRole("link", { name: "Project", exact: true }).click();
  await expect(page).toHaveURL(/#\/$/);
});

test("file links open independent tabs and encode special names", async ({
  page,
  context,
}) => {
  const name = "parts/雪 # 100%.scad";
  await seed(page, { [name]: "sphere(2);" });
  await openHash(page, "#/parts/");
  const fileLink = page.locator(".file-row-open", {
    hasText: "雪 # 100%.scad",
  });
  await expect(fileLink).toHaveAttribute(
    "href",
    "#/parts/%E9%9B%AA%20%23%20100%25.scad",
  );
  const [other] = await Promise.all([
    context.waitForEvent("page"),
    fileLink.click({ modifiers: ["ControlOrMeta"] }),
  ]);
  await expect(other.locator("#editorFilename")).toHaveText(name);
  await openHash(page, "#/main.scad");
  await expect(page.locator("#editorFilename")).toHaveText("main.scad");
  await other.reload();
  await expect(other.locator("#editorFilename")).toHaveText(name);
  await page.reload();
  await expect(page.locator("#editorFilename")).toHaveText("main.scad");
  await expect(other.locator(".cm-content")).toHaveText("sphere(2);");
});

test("switching files clears undo, redo and the rendered/downloadable preview", async ({
  page,
}) => {
  await seed(page, { "other.scad": "sphere(2);" });
  const editor = page.locator(".cm-content");
  await editor.fill("cube(3);");
  const preview = () =>
    page
      .locator("#preview")
      .screenshot({ mask: [page.locator("#axisIndicator")] });
  const blank = await preview();
  await page.getByRole("button", { name: "Compile", exact: true }).click();
  await expect(page.locator("#download")).toBeEnabled();
  expect((await preview()).equals(blank)).toBe(false);
  await page.getByRole("button", { name: "Files", exact: true }).click();
  await page.locator('a[href="#/other.scad"]').click();
  await expect(editor).toHaveText("sphere(2);");
  await expect(page.locator("#download")).toBeDisabled();
  expect((await preview()).equals(blank)).toBe(true);
  await editor.press("ControlOrMeta+z");
  await expect(editor).toHaveText("sphere(2);");
  await editor.press("ControlOrMeta+Shift+z");
  await expect(editor).toHaveText("sphere(2);");
  await editor.fill("sphere(4);");
  await editor.press("ControlOrMeta+z");
  await expect(editor).toHaveText("sphere(2);");
  await page.reload();
  await expect(editor).toHaveText("sphere(2);");
});

test("new files update the hash; renames and deletes keep navigation valid", async ({
  page,
}) => {
  await seed(page, { "parts/one.scad": "cube(1);" });
  await openHash(page, "#/parts/");
  page.once("dialog", (dialog) => dialog.accept("new part"));
  await page.getByRole("button", { name: "New .scad file" }).click();
  await expect(page).toHaveURL(/#\/parts\/new%20part\.scad$/);
  await expect(page.locator("#editorFilename")).toHaveText(
    "parts/new part.scad",
  );
  await page.getByRole("button", { name: "Files", exact: true }).click();
  page.once("dialog", (dialog) => dialog.accept("renamed"));
  await page
    .getByRole("button", { name: "Rename new part.scad", exact: true })
    .click();
  await expect(page.locator("#fileBrowserStatus")).toContainText("Renamed");
  await page.getByRole("button", { name: "Back to editor" }).click();
  await expect(page).toHaveURL(/#\/parts\/renamed\.scad$/);
  await page.getByRole("button", { name: "Files", exact: true }).click();
  page.once("dialog", (dialog) => dialog.accept());
  await page
    .getByRole("button", { name: "Delete renamed.scad", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Back to editor" }),
  ).toBeDisabled();
  await page.goBack();
  await expect(page.locator("#fileBrowserStatus")).toContainText(
    "Path not found: parts/renamed.scad",
  );
  await expect(page).toHaveURL(/#\/parts\/$/);
});

test("duplicate uploads can be skipped, retried and replaced without losing other uploads", async ({
  page,
}) => {
  await seed(page, { "other.scad": "sphere(2);" });
  await openHash(page, "#/other.scad");
  await expect(page.locator("#editorFilename")).toHaveText("other.scad");
  await page.getByRole("button", { name: "Files", exact: true }).click();
  const replacement = {
    name: "other.scad",
    mimeType: "text/plain",
    buffer: Buffer.from("cube(4);"),
  };
  page.once("dialog", async (dialog) => {
    expect(dialog.message()).toContain("Replace other.scad?");
    await dialog.dismiss();
  });
  await page.locator("#fileUploadInput").setInputFiles([
    replacement,
    {
      name: "new.scad",
      mimeType: "text/plain",
      buffer: Buffer.from("cube(5);"),
    },
  ]);
  await expect(page.locator("#fileBrowserStatus")).toHaveText(
    "1 file uploaded. 1 skipped.",
  );
  await expect(page.locator('a[href="#/new.scad"]')).toBeVisible();
  await page.getByRole("button", { name: "Back to editor" }).click();
  await expect(page.locator(".cm-content")).toHaveText("sphere(2);");
  await page.getByRole("button", { name: "Files", exact: true }).click();
  page.once("dialog", (dialog) => dialog.accept());
  await page.locator("#fileUploadInput").setInputFiles(replacement);
  await expect(page.locator("#fileBrowserStatus")).toHaveText(
    "1 file uploaded.",
  );
  await page.getByRole("button", { name: "Back to editor" }).click();
  await expect(page.locator(".cm-content")).toHaveText("cube(4);");
  await page.locator(".cm-content").press("ControlOrMeta+z");
  await expect(page.locator(".cm-content")).toHaveText("cube(4);");
  await page.reload();
  await expect(page.locator(".cm-content")).toHaveText("cube(4);");
});

test("invalid and missing paths recover to a surviving folder", async ({
  page,
}) => {
  await seed(page, { "parts/one.scad": "cube(1);" });
  await page.goto("/#/parts/missing/deleted.scad");
  await expect(page.locator("#fileBrowserStatus")).toContainText(
    "Path not found",
  );
  await expect(page).toHaveURL(/#\/parts\/$/);
  await openHash(page, "#/%ZZ");
  await expect(page.locator("#fileBrowserStatus")).toContainText(
    "URI malformed",
  );
  await expect(page).toHaveURL(/#\/$/);
});

test("late compilation and formatting cannot repopulate a different file", async ({
  page,
}) => {
  await seed(page, { "other.scad": "sphere(2);" });
  await page.evaluate(() => {
    Object.assign(window, { compileDelay: 300, formatDelay: 300 });
  });
  await page.getByRole("button", { name: "Compile", exact: true }).click();
  await expect(page.locator("#status")).toHaveText("Compiling...");
  await openHash(page, "#/other.scad");
  await expect(page.locator("#editorFilename")).toHaveText("other.scad");
  await page.waitForTimeout(400);
  await expect(page.locator("#download")).toBeDisabled();
  await expect(page.locator("#status")).toHaveText("");
  await page.getByRole("button", { name: "Format", exact: true }).click();
  await expect(page.locator("#status")).toHaveText("Formatting...");
  await openHash(page, "#/main.scad");
  await expect(page.locator("#editorFilename")).toHaveText("main.scad");
  await page.waitForTimeout(400);
  await expect(page.locator(".cm-content")).toContainText("// Example");
  await expect(page.locator("#status")).toHaveText("");
});

test("a slow file load cannot override a newer navigation", async ({
  page,
}) => {
  await seed(page, { "slow.scad": "cube(1);", "fast.scad": "sphere(2);" });
  await page.evaluate(async () => {
    const modulePath = "/src/filesystem.ts";
    const { BrowserFileSystem } = await import(modulePath);
    const read = BrowserFileSystem.prototype.readTextFile;
    BrowserFileSystem.prototype.readTextFile = async function (path: string) {
      const source = await read.call(this, path);
      if (path === "slow.scad") {
        Object.assign(window, { slowReadStarted: true });
        await new Promise((resolve) => setTimeout(resolve, 300));
      }
      return source;
    };
  });
  await openHash(page, "#/slow.scad");
  await page.waitForFunction(() => Reflect.get(window, "slowReadStarted"));
  await openHash(page, "#/fast.scad");
  await expect(page.locator("#editorFilename")).toHaveText("fast.scad");
  await page.waitForTimeout(400);
  await expect(page.locator(".cm-content")).toHaveText("sphere(2);");
  await expect(page).toHaveURL(/#\/fast\.scad$/);
});
