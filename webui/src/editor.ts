import { indentWithTab, isolateHistory } from "@codemirror/commands";
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
  applyFormattedSource(source: string): void;
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
        EditorState.tabSize.of(2),
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
    applyFormattedSource(source: string) {
      const previous = editorView.state.doc.toString();
      if (previous === source) {
        return;
      }
      // Keep unchanged text outside the replacement, and make Format a single
      // undo step separate from typing before and after it.
      let from = 0;
      while (
        from < previous.length &&
        from < source.length &&
        previous[from] === source[from]
      ) {
        from++;
      }
      let to = previous.length;
      let end = source.length;
      while (to > from && end > from && previous[to - 1] === source[end - 1]) {
        to--;
        end--;
      }
      editorView.dispatch({
        changes: { from, to, insert: source.slice(from, end) },
        annotations: isolateHistory.of("full"),
      });
    },
    view: editorView,
  };
}
