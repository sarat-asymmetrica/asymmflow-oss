
import { writable } from 'svelte/store';

function createQGIFStore() {
    const { subscribe, set, update } = writable({
        activeAnimation: null,
        playbackSpeed: 1.0,
        isPlaying: false
    });

    return {
        subscribe,
        play: (animName) => update(s => ({ ...s, activeAnimation: animName, isPlaying: true })),
        stop: () => update(s => ({ ...s, isPlaying: false })),
        setSpeed: (speed) => update(s => ({ ...s, playbackSpeed: speed })),
        reset: () => set({ activeAnimation: null, playbackSpeed: 1.0, isPlaying: false })
    };
}

export const qgifStore = createQGIFStore();
