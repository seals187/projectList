// Undo/redo for the block editor. Entries are pairs of undo/redo actions;
// several actions can be bundled into one step (batch).

export interface Step {
  undo: () => void | Promise<void>;
  redo: () => void | Promise<void>;
  /** For merging rapid text input in the same field. */
  mergeKey?: string;
  at?: number;
}

export class History {
  undoStack = $state<Step[]>([]);
  redoStack = $state<Step[]>([]);
  private batch: Step[] | null = null;
  private replaying = false;

  get canUndo() {
    return this.undoStack.length > 0;
  }
  get canRedo() {
    return this.redoStack.length > 0;
  }

  push(step: Step) {
    if (this.replaying) return;
    if (this.batch) {
      this.batch.push(step);
      return;
    }
    const last = this.undoStack[this.undoStack.length - 1];
    const now = Date.now();
    if (step.mergeKey && last?.mergeKey === step.mergeKey && now - (last.at ?? 0) < 2000) {
      // merge text input: keep the old undo, take the new redo
      this.undoStack[this.undoStack.length - 1] = { ...last, redo: step.redo, at: now };
    } else {
      this.undoStack.push({ ...step, at: now });
      if (this.undoStack.length > 200) this.undoStack.shift();
    }
    this.redoStack = [];
  }

  /** Runs fn and merges all steps recorded during it. */
  group(fn: () => void) {
    const outer = this.batch;
    this.batch = outer ?? [];
    try {
      fn();
    } finally {
      if (!outer) {
        const steps = this.batch!;
        this.batch = null;
        if (steps.length === 1) this.push(steps[0]);
        else if (steps.length > 1)
          this.push({
            undo: async () => {
              for (const s of [...steps].reverse()) await s.undo();
            },
            redo: async () => {
              for (const s of steps) await s.redo();
            }
          });
      }
    }
  }

  async undo() {
    const s = this.undoStack.pop();
    if (!s) return;
    this.replaying = true;
    try {
      await s.undo();
    } finally {
      this.replaying = false;
    }
    this.redoStack.push(s);
  }

  async redo() {
    const s = this.redoStack.pop();
    if (!s) return;
    this.replaying = true;
    try {
      await s.redo();
    } finally {
      this.replaying = false;
    }
    this.undoStack.push({ ...s, mergeKey: undefined });
  }
}
