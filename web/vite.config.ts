import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig, type Plugin } from 'vite';

// Fortune-sheet (tables) inserts rows with `new Function(...)`, which the CSP
// (script-src 'self', no 'unsafe-eval') blocks. The evaluated text is only
// JSON.stringify(row) joined by commas, so JSON.parse does the same. The build
// fails when the code changes (e.g. after an update), instead of silently
// shipping a table that cannot insert rows.
const FORTUNE_CORE = /@fortune-sheet[\\/]core[\\/]dist[\\/]index\.esm\.js$/;
const parsed = 'arr.map(function (s) { return JSON.parse(s); })';
const FORTUNE_PATCHES: [string, string][] = [
  ['new Function("d", "return d.unshift(".concat(arr.join(","), ")"))(d)', `d.unshift.apply(d, ${parsed})`],
  ['new Function("d", "return d.splice(".concat(index, ", 0, ").concat(arr.join(","), ")"))(d)', `d.splice.apply(d, [index, 0].concat(${parsed}))`],
  ['new Function("d", "return d.splice(".concat(index + 1, ", 0, ").concat(arr.join(","), ")"))(d)', `d.splice.apply(d, [index + 1, 0].concat(${parsed}))`]
];
function patchFortuneCore(code: string): string {
  for (const [from, to] of FORTUNE_PATCHES) {
    if (!code.includes(from)) throw new Error(`fortune-sheet patch: code not found: ${from}`);
    code = code.split(from).join(to);
  }
  if (code.includes('new Function')) throw new Error('fortune-sheet patch: unexpected "new Function" left');
  return code;
}
function fortuneSheetNoEval(): Plugin {
  return {
    name: 'fortune-sheet-no-eval',
    enforce: 'pre',
    // production build (Rollup)
    transform: (code, id) => (FORTUNE_CORE.test(id.split('?')[0]) ? { code: patchFortuneCore(code), map: null } : null)
  };
}

export default defineConfig({
  plugins: [fortuneSheetNoEval(), tailwindcss(), sveltekit()],
  optimizeDeps: {
    // dev server: dependencies are pre-bundled by esbuild, without the Rollup transform above
    esbuildOptions: {
      plugins: [
        {
          name: 'fortune-sheet-no-eval',
          setup(build) {
            build.onLoad({ filter: FORTUNE_CORE }, async (args) => {
              // @ts-ignore -- no @types/node in this project (vite.config runs in Node)
              const { readFile } = await import('node:fs/promises');
              return { contents: patchFortuneCore(await readFile(args.path, 'utf8')), loader: 'js' };
            });
          }
        }
      ]
    }
  },
  build: {
    // Never inline fonts as data: URLs (the CSP only allows fonts from 'self');
    // otherwise Vite's default limit (4 KB) applies.
    assetsInlineLimit: (file) => (/\.(woff2?|ttf|otf)$/.test(file) ? false : undefined)
  },
  server: {
    // For development: forward the API to the locally running Go server.
    proxy: { '/api': 'http://localhost:8080', '/healthz': 'http://localhost:8080' }
  }
});
