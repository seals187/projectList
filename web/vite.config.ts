import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  build: {
    // Schriften nie als data:-URL einbetten (die CSP erlaubt Schriften nur von 'self');
    // sonst gilt Vites Standardgrenze (4 KB).
    assetsInlineLimit: (file) => (/\.(woff2?|ttf|otf)$/.test(file) ? false : undefined)
  },
  server: {
    // Für die Entwicklung: API an den lokal laufenden Go-Server weiterreichen.
    proxy: { '/api': 'http://localhost:8080', '/healthz': 'http://localhost:8080' }
  }
});
