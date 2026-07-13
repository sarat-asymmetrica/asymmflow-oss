import { writable } from 'svelte/store';

/**
 * A one-line notice shown ON the login/lock surface explaining WHY the user
 * landed there (e.g. an inactivity session timeout).
 *
 * Wave 10 B6 (Article IV.4 + V): a session-expiry is a background state
 * transition, not an action the user just took — so it must NOT arrive as a
 * toast. It belongs on the destination surface (the login screen), which is
 * exactly the kind of "state, not event" routing Article V calls for. Set it
 * when redirecting to login; the login screen displays and then clears it.
 */
export const authNotice = writable<string>('');
