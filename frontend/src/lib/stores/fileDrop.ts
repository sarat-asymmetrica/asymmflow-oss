import { writable } from 'svelte/store';

// Global file drop store - allows any screen to subscribe to file drops
interface FileDropEvent {
    x: number;
    y: number;
    paths: string[];
    timestamp: number;
}

function createFileDropStore() {
    const { subscribe, set } = writable<FileDropEvent | null>(null);

    return {
        subscribe,
        // Called by App.svelte when files are dropped
        drop: (x: number, y: number, paths: string[]) => {
            set({ x, y, paths, timestamp: Date.now() });
        },
        // Clear after handling
        clear: () => set(null)
    };
}

export const fileDrop = createFileDropStore();
