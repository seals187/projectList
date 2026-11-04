# ProjectList

Schlankes, selbst gehostetes Projektmanagement-Tool: **Kunden → Projekte → Checklisten**.
Checklisten lassen sich aus wiederverwendbaren Templates erzeugen (als unabhängige Kopie).
Änderungen pro Version: siehe [CHANGELOG.md](CHANGELOG.md).

- Ein einziges Go-Binary mit eingebettetem SvelteKit-Frontend, SQLite (pure Go, kein CGO)
- Ein Container, Ziel < 50 MB RAM
- Multiuser mit Rollen (`admin`, `member`, `viewer`) und Kundenzuordnung

## Schnellstart (Docker)

```bash
cp .env.example .env        # ADMIN_EMAIL / ADMIN_PASSWORD anpassen!
docker compose up -d
```

Danach unter <http://localhost:8080> mit den Admin-Zugangsdaten aus der `.env` anmelden.
Der erste Admin wird nur beim allerersten Start angelegt (leere Datenbank).

Status prüfen:

```bash
docker compose ps                     # Healthcheck-Status
curl http://localhost:8080/healthz    # {"status":"ok"}
docker compose logs -f projectlist    # strukturierte JSON-Logs
```

### Fertiges Image von GitHub (ghcr.io)

Zu jedem Release gibt es ein Image für **linux/amd64** und **linux/arm64** (z. B. Raspberry Pi):

```bash
docker pull ghcr.io/seals187/projectlist:latest   # oder eine feste Version, z. B. :0.0.2
```

Ohne eigenen Build: in `docker-compose.yml` die Zeile `build:` (samt `context`/`args`) entfernen und
`image: ghcr.io/seals187/projectlist:latest` setzen, dann `docker compose up -d`. Oder direkt:

```bash
docker run -d --name projectlist -p 8080:8080 --env-file .env \
  -v projectlist-data:/data --memory 128m ghcr.io/seals187/projectlist:latest
```

Veröffentlicht wird automatisch per GitHub Actions (`.github/workflows/docker.yml`), sobald ein Tag
`vX.Y.Z` gepusht wird – mit den Tags `:X.Y.Z`, `:X.Y` und `:latest`. Ein bestehendes Tag lässt sich unter
*Actions → CI und Container-Image → Run workflow* nachträglich veröffentlichen.

## Umgebungsvariablen

| Variable         | Standard               | Beschreibung |
|------------------|------------------------|--------------|
| `ADMIN_EMAIL`    | –                      | E-Mail des ersten Admins (nur bei leerer DB) |
| `ADMIN_PASSWORD` | –                      | Passwort des ersten Admins (min. 8 Zeichen) |
| `APP_SUBTITLE`   | –                      | Optionaler Untertitel unter „ProjectList“ in der Seitenleiste (z. B. Firmen-/Abteilungsname) |
| `ADMIN_NAME`     | `Admin`                | Anzeigename des ersten Admins |
| `PORT`           | `8080`                 | Host-Port (nur docker compose) |
| `LISTEN_ADDR`    | `:8080`                | Listen-Adresse im Container |
| `DB_PATH`        | `/data/projectlist.db` | Pfad der SQLite-Datei |
| `COOKIE_SECURE`  | `true`                 | Session-Cookie nur über HTTPS. Auf `http://localhost` funktioniert das in Chrome/Edge/Firefox; für Zugriff per HTTP über IP/Hostname `false` setzen |
| `SESSION_TTL`    | `720h`                 | Gültigkeit einer Session (wird bei Nutzung verlängert) |
| `LOG_LEVEL`      | `info`                 | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT`     | `json`                 | `json` oder `text` |

Für den Produktivbetrieb empfiehlt sich ein Reverse-Proxy mit TLS (Caddy, Traefik, nginx) vor dem Container.
Der Proxy muss den `Host`-Header unverändert weitergeben (Origin-Prüfung).

## Rollen & Rechte

| Rolle    | Sichtbarkeit             | Rechte |
|----------|--------------------------|--------|
| `admin`  | alle Kunden              | alles inkl. Benutzerverwaltung (*Settings → Users*), Kunden anlegen/löschen, Backup |
| `member` | nur zugeordnete Kunden   | Kunden bearbeiten; Projekte, Checklisten, Items anlegen/ändern/löschen/abhaken; Templates verwalten |
| `viewer` | nur zugeordnete Kunden   | nur lesen |

Alle Rechte werden serverseitig in jedem Endpunkt geprüft. Objekte fremder Kunden liefern `404`.
Fehler liefert die API als sprachneutrale Codes (z. B. `{"error":"validation","fields":{"name":"required"}}`);
die Oberfläche übersetzt sie.

## Bedienung

- **Sprache** (English, Deutsch, Français), **Darstellung** (System/Hell/Dunkel/Dunkelgrau) und **Schrift** unter *Settings*:
  Inter (Standard, mit optischer Größe), Geist, Mona Sans, Instrument Sans, Figtree, Onest, Plus Jakarta Sans, Manrope, DM Sans,
  Schibsted Grotesk, IBM Plex Sans oder System; Überschriften in
  Serif (Source Serif 4) oder wie der Text; Code in JetBrains Mono. Alle Schriften sind eingebettet (keine externen Anfragen);
  die Sprache wird beim ersten Besuch aus dem Browser übernommen
- **Übersicht** in drei Ansichten; die **Standardansicht** ist unter *Settings* einstellbar (Karten, Liste, Board oder „Zuletzt
  genutzte“), die gerade genutzte Ansicht bleibt bis zum Schließen der App erhalten; in *Karten* und *Liste* sind die Projekte je Kunde
  einklappbar (Standard: eingeklappt, „Alle ausklappen“ oben rechts, Zustand wird pro Browser gemerkt):
  - *Karten*: je Kunde seine Projekte mit Status, Team, Fortschritt je Checkliste, Fälligkeit und nächstem offenen Schritt
  - *Liste*: Kennzahlen (aktive Projekte, offene Aufgaben, in 14 Tagen fällig, blockiert) und eine nach Kunden gruppierte Tabelle
  - *Board*: Spalten Planung · Aktiv · Review · Blockiert · Erledigt, Projekte per Drag & Drop verschieben, Filter nach Kunden
- **Meine Aufgaben**: Tab *Kunden* (Kunden → Projekte → Checklisten, an denen man beteiligt ist, mit Kundenfortschritt und
  Fälligkeit „Gestern/Heute/Morgen“) und Tab *Persönlich* (zugewiesene Einträge zum Abhaken); Filter Offen / Heute / Erledigt
  (erledigt = letzte 30 Tage)
- **Kundenfortschritt**: Pro Projekt schaltet das Tortendiagramm-Symbol, ob es in den Gesamtfortschritt des Kunden einfliesst
  (Admins und Members; Viewer sehen nur den Zustand). Ausgenommene Projekte werden grau dargestellt, beim Kunden steht
  „ohne N Projekt(e)“; gilt gleich in Meine Aufgaben, Kundenliste und Übersicht
- **Kunden**-Übersicht als Liste (aufklappbar mit Projekten, Checklisten-Fortschritt, Fälligkeit) oder Karten; Suche,
  Filter (alle, mit aktiven Projekten, kritisch, ohne Projekt), Sortierung (letzte Aktivität, Name, Fortschritt, Fälligkeit, offene
  Aufgaben); Status je Kunde: *Neu* (keine Projekte), *Kritisch* (Projekt blockiert oder in ≤ 3 Tagen fällig/überfällig),
  *Abgeschlossen*, sonst *Im Plan*. Ort, Ansprechpartner und E-Mail auf der Kundenseite bearbeitbar
- **Projektstatus** (Planung, Aktiv, Review, Blockiert, Erledigt, Archiviert) und **Fälligkeitsdatum** oben auf der Projektseite;
  „Neues Projekt“ in der Übersicht legt optional gleich die erste Checkliste aus einer Vorlage an
- **Seitenleiste**: Suche und **+** (neues Projekt), **Favoriten** (Stern am Projekt in der Seitenleiste oder auf der Projektseite),
  **Zuletzt** besuchte Kunden/Projekte/Checklisten/Templates mit Art und Zugehörigkeit, „Leeren“ (beides pro Benutzer in der Datenbank,
  also auf allen Geräten gleich); Abschnitte Favoriten/Zuletzt/Kunden einklappbar; Kunden mit Kürzel und Projektanzahl
  (aufklappbar, Projekte mit Status-Punkt, „+ Neues Projekt“ je Kunde); unten Benutzer (→ Einstellungen), Theme-Umschalter, Abmelden
- **Strg+K** / **⌘K** öffnet die Suche (Kunden, Projekte, Checklisten)
- Titel von Kunden, Projekten und Checklisten anklicken zum Bearbeiten – **Enter** speichert, **Esc** bricht ab

### Block-Editor (Checklisten & Templates)

Checklisten bestehen aus **Blöcken**, die wie in einem Dokument bearbeitet werden. Gespeichert wird
automatisch (Anzeige „saved“ oben rechts).

| Blocktyp | Kürzel beim Tippen | Hinweis |
|---|---|---|
| Checklisten-Eintrag | `[] ` | Checkbox, Zuständige, Fälligkeit – nur diese zählen zum Fortschritt |
| Überschrift 1–4 | `# ` … `#### ` | H1 bildet Abschnitte mit Zähler (auch in der Sidebar) |
| Normaler Text | – | |
| Aufzählung / nummerierte Liste | `- ` / `1. ` | |
| Zitat | `> ` | |
| Codeblock | ` ``` ` | Syntax-Highlighting (Bash, PowerShell, SQL, JSON, YAML, JS, Python), Zeilennummern, *Copy*, Hintergrund Hellgrau/Dunkelgrau/Schwarz (unter *Settings → Checklisten*) |
| Trennlinie | `---` | |

- **`/`** am Zeilenanfang öffnet das Befehlsmenü (z. B. `/hea` → Überschriften); **+** am Zeilenrand
  fügt darunter einen Block ein und öffnet das Menü
- **Enter** teilt den Block bzw. legt einen neuen gleichen Typs an; Enter in einem leeren Listeneintrag
  beendet die Liste. **Backspace** am Zeilenanfang wandelt in Text um bzw. verbindet mit dem Block davor.
  **Tab / Shift+Tab** rückt ein/aus (z. B. Codeblock unter einem Eintrag). **↑/↓** wechseln den Block.
- **Toolbar**: Rückgängig/Wiederholen (**Strg+Z / Strg+Y**), Blocktyp, H1–H4, **fett** (Strg+B),
  *kursiv* (Strg+I), unterstrichen (Strg+U), durchgestrichen, `Inline-Code` (Strg+E), Textfarbe,
  Hervorhebung, Listen, Zitat, Codeblock, Link, Trennlinie, Formatierung entfernen; rechts **+ Heading** / **+ Item**
- Im Codeblock: **Tab** rückt ein, **Esc** oder **Strg+Enter** springt zum nächsten Block
- Mehrzeiliges Einfügen erzeugt pro Zeile einen eigenen Block
- **Einklappen**: Der Pfeil vor einer Überschrift klappt ihren Abschnitt ein (bis zur nächsten Überschrift
  gleicher oder höherer Ebene; H2 klappt also seine H3/H4 mit ein). Eingeklappte Überschriften zeigen
  Fortschritt und die Zahl der ausgeblendeten Blöcke; oben rechts gibt es *Alle einklappen / ausklappen*.
  Der Zustand wird pro Browser gespeichert (nicht in der Datenbank), verschiebt man eine eingeklappte
  Überschrift, wandert der Abschnitt mit
- **Links**: blau dargestellt. Klick auf einen Link (oder der Link-Button) öffnet einen Dialog für
  **Text** und **Adresse (URL)**, mit *Öffnen* und *Link entfernen*; ohne markierten Text wird ein neuer
  Link an der Cursorposition eingefügt. Strg+Klick öffnet den Link direkt
- **Zeilenfarbe**: über das Tropfen-Symbol rechts in jeder Zeile (beim Überfahren); hat Vorrang vor der
  Abschnittsfarbe. In einem farbigen Abschnitt gibt es zusätzlich *Wie Abschnitt* (Farbe erben) und
  *Ohne Farbe* (Zeile bleibt weiß, auch im farbigen Abschnitt)
- **Farbstufen** wie in Word/Excel: jede Farbe (Grau, Gelb, Grün, Blau, Rot, Lila) gibt es hell, mittel
  und kräftig (Textmarker-Farben wie `#FFFF00`) – für Zeilen, Abschnitte, Textfarbe und Hervorhebung,
  überall im gleichen Raster (Farbtöne nebeneinander, Stufen untereinander)
- **In Checklistenpunkt umwandeln**: Checkbox-Symbol rechts in Text-, Listen-, Zitat- und
  Überschriftszeilen (beim Überfahren) macht daraus einen Punkt mit Checkbox (Strg+Z macht es rückgängig)
- **Status je Eintrag**: rechts neben der Checkbox ein freies Textfeld (max. 60 Zeichen) mit Vorschlägen
  wie *Erfüllt*, *Erreicht*, *Teilweise*; bereits verwendete Werte der Checkliste werden ebenfalls
  vorgeschlagen. Der Status wird nicht in Templates übernommen
- **Rahmenlinien** um jede Zeile wie in einer Tabelle, ein/aus über das Tabellen-Symbol in der Toolbar
  oder unter Einstellungen → Checklisten (Standard: an). Die Notizspalte bleibt immer ohne Abschnittsfarbe
- **Zuständig für eine ganze Checkliste**: im Kopf der Checkliste über *Zuweisen* (nur Benutzer mit
  Zugriff auf den Kunden). Zugewiesene Checklisten erscheinen unter *Meine Aufgaben → Meine Checklisten*
  und auf der Projektseite mit Avatar
- **Seitenleiste einklappen**: Symbol neben dem Logo oder **Strg+\\** (wird pro Browser gemerkt)
- **Seitenleiste breiter/schmaler**: am rechten Rand ziehen (200–480 px, Doppelklick = Standardbreite;
  wird pro Browser gemerkt)
- **Darstellung** (persönlich, pro Browser): Checkbox **rechts oder links** und **eckig oder rund**
  (Standard: rechts, eckig) unter *Einstellungen → Checklisten*; die **Größe** der Checkliste
  (80–160 % in 5 %-Schritten) über **− / +** in der Toolbar (Klick auf die Prozentzahl setzt auf 100 % zurück)
- **Abschnittsfarbe**: Über das Tropfen-Symbol rechts an einer Überschrift bekommt der ganze Abschnitt
  einen Hintergrund (Grau, Gelb, Grün, Blau, Rot, Lila). Die Farbe wird gespeichert, gilt für alle
  Benutzer und wird in Templates übernommen; ein verschachtelter farbiger Abschnitt hat Vorrang
- Rechts steht die **Notizspalte** (ebenfalls formatierbar); ein Klick auf den farbigen Balken wechselt
  die Farbe (grau → grün → orange → rot → indigo)
- Zuständige und Fälligkeit: beim Überfahren einer Zeile über die Symbole 👤/📅; „today“ bzw.
  überfällige Daten werden rot markiert
- Blöcke und Checklisten über den Griff (⋮⋮) per Drag & Drop sortieren (auch per Touch)
- Formatierungen werden als kleines HTML-Subset gespeichert und serverseitig per Whitelist bereinigt
  (`b i u s code a span[data-color] mark[data-bg] br`)
- „Aus Template“ zeigt eine Vorschau; die Einträge werden **kopiert** – spätere Template-Änderungen
  beeinflussen bestehende Checklisten nicht
- „Save as template“ im Kopf einer Checkliste speichert sie als neues Template
- *My tasks* listet alle offenen, dir zugewiesenen Einträge nach Fälligkeit

## Backup & Restore

**Backup per CLI** (konsistent auch im laufenden Betrieb, via `VACUUM INTO`):

```bash
docker compose exec projectlist /projectlist backup
# → /data/backups/projectlist-YYYYMMDD-HHMMSS.db (im Volume)

# Backup auf den Host kopieren:
docker compose cp projectlist:/data/backups/. ./backups/
```

Optional mit eigenem Ziel: `/projectlist backup /data/backups/vor-update.db`.

**Backup per Browser:** Als Admin unter *Settings → Backup herunterladen*
(Endpunkt `GET /api/admin/backup`).

**Restore:**

Das Laufzeit-Image enthält keine Shell (distroless). Daher wird die Datei über einen kurzlebigen
Hilfscontainer ins Volume kopiert (Backup-Datei liegt in `./backups/` auf dem Host):

```bash
docker compose stop projectlist
docker run --rm -v projectlist_projectlist-data:/data -v "$PWD/backups:/backup:ro" alpine sh -c \
  "cp /backup/projectlist-XXXX.db /data/projectlist.db && rm -f /data/projectlist.db-wal /data/projectlist.db-shm && chown 65532:65532 /data/projectlist.db"
docker compose start projectlist
```

Die alten `-wal`/`-shm`-Dateien müssen dabei entfernt werden, sonst würde SQLite sie auf die
wiederhergestellte Datei anwenden.

## Update

```bash
git pull
docker compose exec projectlist /projectlist backup   # vorher sichern
docker compose up -d --build
```

Datenbank-Migrationen laufen beim Start automatisch (goose, eingebettet). Daten liegen im Volume
`projectlist-data` und bleiben bei Updates erhalten.

## Entwicklung

Voraussetzungen: Go 1.27+, Node 26+ – oder nur Docker (alle Schritte laufen auch in Containern).

```bash
# Backend-Tests (In-Memory-SQLite)
go test ./...
# … oder ohne lokale Go-Installation:
docker build --target test .

# Frontend
cd web && npm install && npm run dev     # Vite-Dev-Server, /api → localhost:8080
npm run check                             # svelte-check
npm run build                             # → web/build (wird ins Binary eingebettet)

# Server lokal
ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=passwort-123 COOKIE_SECURE=false \
  LOG_FORMAT=text go run ./cmd/projectlist

# SQL-Queries geändert? sqlc-Code neu erzeugen:
docker run --rm -v "$PWD:/src" -w /src sqlc/sqlc:1.31.1 generate
```

Neue Migrationen als `migrations/0000N_name.sql` im goose-Format anlegen.

### Projektstruktur

```
cmd/projectlist/      Einstieg: serve | backup | migrate | healthcheck | version
internal/config/      ENV-Konfiguration
internal/db/          SQLite öffnen (WAL, FK, busy_timeout), Migrationen, Transaktionen
internal/db/queries/  SQL-Quellen für sqlc
internal/db/sqlc/     generierter, typsicherer Query-Code
internal/auth/        Argon2id, Sessions, Login-Limiter, Admin-Bootstrap
internal/authz/       Rollen & Kundenzuordnung
internal/service/     Template anwenden / Checkliste als Template (Transaktionen)
internal/richtext/    Whitelist-Bereinigung der Inline-Formatierung
internal/httpapi/     JSON-API, Middleware, Validierung
internal/static/      Auslieferung des eingebetteten Frontends (br/gzip, Cache-Header)
migrations/           goose-Migrationen (eingebettet)
web/                  SvelteKit + Tailwind (adapter-static)
```

## Technische Details

- **Sicherheit:** Argon2id (19 MiB, t=2, OWASP), serverseitige Sessions (nur SHA-256 des Tokens in der DB),
  Cookie `HttpOnly; Secure; SameSite=Lax`, CSRF-Token-Header + Origin-Prüfung, Login-Rate-Limit,
  CSP mit Script-Hashes, distroless-Image als `nonroot`
- **Performance:** Projektansicht lädt alle Checklisten samt Items mit zwei Abfragen; Fortschritt in Sidebar
  und Kundenansicht per SQL-Aggregation; Assets beim Build vorkomprimiert (Brotli/Gzip), gehashte Dateien
  mit `Cache-Control: immutable`
- **Betrieb:** Graceful Shutdown (SIGTERM), WAL-Checkpoint beim Beenden, strukturierte Logs (`log/slog`),
  `/healthz`, Speicherlimit 128 MB + `GOMEMLIMIT` im Compose-File

## API (Kurzüberblick)

Alle Endpunkte unter `/api`, JSON. Schreibende Requests benötigen den Header `X-CSRF-Token`
(aus `POST /api/auth/login` bzw. `GET /api/auth/me`).

| Bereich     | Endpunkte |
|-------------|-----------|
| Auth        | `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`, `PUT /auth/password` |
| Benutzer    | `GET/POST /users`, `PATCH/DELETE /users/{id}`, `PUT /users/{id}/customers` (Admin) |
| Kunden      | `GET/POST /customers`, `GET/PATCH/DELETE /customers/{id}` |
| Projekte    | `GET/POST /customers/{id}/projects`, `GET/PATCH/DELETE /projects/{id}` |
| Checklisten | `GET/POST /projects/{id}/checklists`, `POST /projects/{id}/checklists/from-template`, `PUT /projects/{id}/checklists/order`, `GET/PATCH/DELETE /checklists/{id}`, `POST /checklists/{id}/save-as-template` |
| Items       | `POST /checklists/{id}/items`, `PUT /checklists/{id}/items/order`, `PATCH/DELETE /items/{id}`, `POST /items/{id}/done` |
| Templates   | `GET/POST /templates`, `GET/PATCH/DELETE /templates/{id}`, `POST /templates/{id}/items`, `PUT /templates/{id}/items/order`, `PATCH/DELETE /template-items/{id}` |
| Seitenleiste | `PUT/DELETE /favorites/{projectId}`, `POST/DELETE /recent` (`kind` = `customer`, `project`, `checklist` oder `template`, dazu `id`); beide Listen kommen mit `GET /sidebar` |
| Sonstiges   | `GET /sidebar`, `GET /overview`, `GET /my-tasks`, `GET /my-checklists`, `GET /search?q=`, `GET /admin/backup`, `GET /healthz` |

## Lizenz

[GNU Affero General Public License v3.0](LICENSE)
