import { indentWithTab } from "@codemirror/commands";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { basicSetup } from "codemirror";

export const DEFAULT_SOURCE = `// Example
difference() {
  cube(2, center=true);
  translate([1, 1, 1]) sphere(1);
}
`;

interface EditorOptions {
  parent: HTMLElement;
  initialSource: string;
  onSave?: () => void;
  onChange?: (source: string) => void;
}

export interface SourceEditor {
  getSource(): string;
  setSource(source: string): void;
  view: EditorView;
}

export function createEditor({
  parent,
  initialSource,
  onSave,
  onChange,
}: EditorOptions): SourceEditor {
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
          onChange?.(editorView.state.doc.toString());
        }),
      ],
    }),
    parent,
  });

  return {
    getSource() {
      return editorView.state.doc.toString();
    },
    setSource(source: string) {
      editorView.dispatch({
        changes: {
          from: 0,
          to: editorView.state.doc.length,
          insert: source,
        },
      });
    },
    view: editorView,
  };
}
