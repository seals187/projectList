// Sets color scheme and font before the first render (prevents flashing).
// "dim" = dark gray: dark mode with lighter gray tones (class .dim in addition to .dark).
// Font: html[data-font] / html[data-heading], see app.css and $lib/fonts.ts.
(function () {
  var el = document.documentElement;
  var t = null;
  try { t = localStorage.getItem('theme'); } catch (e) {}
  var dark = t === 'dark' || t === 'dim' || (t !== 'light' && window.matchMedia('(prefers-color-scheme: dark)').matches);
  el.classList.toggle('dark', dark);
  el.classList.toggle('dim', t === 'dim');
  try {
    var p = JSON.parse(localStorage.getItem('view-prefs') || '{}');
    if (typeof p.font === 'string' && /^[a-z]+$/.test(p.font) && p.font !== 'inter') el.dataset.font = p.font;
    if (p.headingFont === 'sans') el.dataset.heading = 'sans';
  } catch (e) {}
})();
