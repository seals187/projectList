import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    // Reine SPA: Go liefert index.html als Fallback für alle Routen aus.
    adapter: adapter({ pages: 'build', assets: 'build', fallback: 'index.html', precompress: true }),
    paths: { relative: false },
    csp: {
      mode: 'hash',
      directives: {
        'default-src': ['self'],
        'script-src': ['self'],
        'style-src': ['self', 'unsafe-inline'],
        'img-src': ['self', 'data:'],
        'connect-src': ['self'],
        'object-src': ['none'],
        'base-uri': ['self']
      }
    }
  }
};

export default config;
