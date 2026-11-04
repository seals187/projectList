// Setzt Farbschema und Schrift vor dem ersten Rendern (verhindert Aufblitzen).
// "dim" = Dunkelgrau: Dunkelmodus mit helleren Grautönen (Klasse .dim zusätzlich zu .dark).
// Schrift: html[data-font] / html[data-heading], siehe app.css und $lib/fonts.ts.
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
