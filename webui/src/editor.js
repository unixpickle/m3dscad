import { indentWithTab } from "@codemirror/commands";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { basicSetup } from "codemirror";

const SOURCE_STORAGE_KEY = "m3dscad_source";

const DEFAULT_SOURCE = `// Example
difference() {
  cube(2, center=true);
  translate([1, 1, 1]) sphere(1);
}
`;

export function loadInitialSource() {
  const storedSource = window.localStorage.getItem(SOURCE_STORAGE_KEY);
  return storedSource && storedSource.trim().length > 0
    ? storedSource
    : DEFAULT_SOURCE;
}

export function createEditor({ parent, initialSource, onSave }) {
  const editorView = new EditorView({
    state: EditorState.create({
      doc: initialSource,
      extensions: [
        basicSetup,
        keymap.of([
          {
            key: "Mod-s",
            run: () => {
              onSave?.();
              return true;
            },
          },
          indentWithTab,
        ]),
        EditorView.lineWrapping,
        EditorView.updateListener.of((update) => {
          if (!update.docChanged) {
            return;
          }
          window.localStorage.setItem(
            SOURCE_STORAGE_KEY,
            editorView.state.doc.toString(),
          );
        }),
      ],
    }),
    parent,
  });

  return {
    getSource() {
      return editorView.state.doc.toString();
    },
    view: editorView,
  };
}
