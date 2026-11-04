import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
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
