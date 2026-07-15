/* Minimal ambient declarations for the node builtins used by the test harness.
 *
 * The project deliberately ships no `@types/node` (it is a browser/Wails
 * frontend; adding "node" to tsconfig `types` would pollute ambient globals
 * across all ~50 screens). These tripwire/unit tests run in vitest's node
 * environment and read source files off disk. Declaring only the exact surface
 * they use keeps `npm run check` green without a dependency and without leaking
 * node globals into screen code — only files that `import 'node:*'` see these. */
declare module 'node:fs' {
  export function readFileSync(path: string, encoding: 'utf8'): string
  export function readdirSync(path: string): string[]
}
declare module 'node:path' {
  export function join(...parts: string[]): string
  export function dirname(path: string): string
}
declare module 'node:url' {
  export function fileURLToPath(url: string): string
}
