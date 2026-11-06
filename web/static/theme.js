// Sets color scheme, font and appearance before the first render (prevents flashing).
// Theme: light, dark or system (old value "dim" = dark; no choice = system).
// Font: html[data-font] / html[data-heading]; accent: html[data-accent];
// density: html[data-density]; app font size: html font-size; glass off: html.no-glass
// (see app.css, $lib/fonts.ts and $lib/appearance.ts).
(function () {
  var el = document.documentElement;
  var t = null;
  try { t = localStorage.getItem('theme'); } catch (e) {}
  var dark = t === 'dark' || t === 'dim' || (t !== 'light' && window.matchMedia('(prefers-color-scheme: dark)').matches);
  el.classList.toggle('dark', dark);
  try {
    var p = JSON.parse(localStorage.getItem('view-prefs') || '{}');
    if (typeof p.font === 'string' && /^[a-z]+$/.test(p.font) && p.font !== 'inter') el.dataset.font = p.font;
    if (p.headingFont === 'sans') el.dataset.heading = 'sans';
    if (/^(blue|teal|green|orange|red|pink)$/.test(p.accent)) el.dataset.accent = p.accent;
    if (p.density === 'compact' || p.density === 'airy') el.dataset.density = p.density;
    if (typeof p.scale === 'number' && p.scale >= 0.8 && p.scale <= 1.2 && p.scale !== 1) el.style.fontSize = p.scale * 100 + '%';
    if (p.glass === false) el.classList.add('no-glass');
  } catch (e) {}
})();
