# Changelog

Alle nennenswerten Änderungen an ProjectList werden in dieser Datei festgehalten.

Das Format basiert auf [Keep a Changelog](https://keepachangelog.com/de/1.1.0/),
die Versionsnummern folgen [Semantic Versioning](https://semver.org/lang/de/).
Änderungen bis zum nächsten Release sammeln sich unter **Unveröffentlicht**.

## [Unveröffentlicht]

## [0.0.7] – 2026-11-05

### Hinzugefügt

- Checklisten und Templates: Die Breite der Notizspalte lässt sich an ihrem linken Rand mit der Maus nach links
  oder rechts ziehen (Doppelklick: Standardbreite, auch per Pfeiltasten); die Breite wird gespeichert

### Geändert

- Einheitliche Farben für ausgewählt und Maus darüber: Seitenleiste, Ansichts-Umschalter (Karten/Board/Liste/…,
  Kunden Karten/Liste), Reiter in „Meine Aufgaben“, Zoom der Zeitachse, Einstellungen und Filter-Chips.
  Ausgewählt ist immer violett hinterlegt mit violetter Schrift, Maus darüber immer eine neutrale graue Fläche;
  Filter-Chips überall gleich (rund). Suchfeld der Seitenleiste im dunklen Design und auf Hintergründen heller
- Hintergründe: Filter, Suchfeld, Chips und Knöpfe im Kopfband sind helles Glas statt dunkler Kästen; das
  Kopfband ist auf Übersicht, Meine Aufgaben, Kunden und Templates immer gleich hoch; „Neues Template“ steht
  auf einer Glasfläche
- Hintergründe: Board-Spalten, Listen, Tabellen und Kacheln wieder weniger durchsichtig, damit sie auch auf
  unruhigen Bildern gut lesbar sind

### Behoben

- Einige helle Akzentfarben im dunklen Design fehlten (z. B. „Gruppieren“ im Board war kaum lesbar)

## [0.0.6] – 2026-11-05

### Hinzugefügt

- Hintergründe: 25 Farbverläufe oder eigene Bilder (JPG, PNG, WebP, GIF; max. 5 MB, grössere Bilder werden vor
  dem Speichern automatisch verkleinert). Wählbar pro Benutzer für die ganze App (Einstellungen, auf allen
  Geräten gleich), pro Kunde (Knopf auf der Kundenseite) und pro Projekt („…“-Menü der Projektseite); es gilt
  der spezifischste: Projekt vor Kunde vor Benutzer. Hochgeladene Bilder stehen allen zur Auswahl; löschen
  können sie der Hochladende und Admins. Darstellung wie bei Planka: Das Bild liegt hinter der ganzen App,
  Seitenleiste und schmales Kopfband (Titel, Ansichten, Filter, Reiter) sind durchscheinendes dunkles Glas,
  Board-Spalten, Listen, Tabellen, Kalender, Zeitachse und die Kacheln darin durchscheinendes helles Glas;
  dazwischen ist das Bild frei sichtbar. Checklisten-, Template-Editor und Einstellungen stehen auf einer
  deckenden Fläche. Migration 00019

### Geändert

- Meine Aufgaben: Projekte, für die man verantwortlich ist, erscheinen jetzt ebenfalls (mit Markierung
  „Verantwortlich“), auch wenn einem darin keine Checkliste und kein Eintrag zugewiesen ist
- Projektseite: Checklisten lassen sich direkt in der Liste zuweisen und löschen (mit Rückfrage), ohne sie
  zu öffnen
- Kundenseite neu gestaltet: Kopf mit Kürzel, Name, „Bearbeiten“ und „+ Projekt“, Ort/Ansprechpartner/E-Mail
  (E-Mail als Link; Bearbeiten über den Knopf), Notizen, Projekte als Tabelle mit Status (eigener Name und Farbe
  wie im Board), Labels, Fortschritt, Fälligkeit und verantwortlicher Person
- Übersicht, Liste: Spalte „Zuständig“ statt „Team / Zuständig“; in der Projektzeile steht die verantwortliche
  Person statt des Teams, in den Checklisten-Zeilen wie bisher die zugewiesene Person
- Übersicht, Board: „Alle einklappen“ / „Alle ausklappen“ statt des Schalters „Kompakt“ (eingeklappt zeigen die
  Karten nur Kunde, Projektname und Fortschritt)
- Übersicht, Karten: Projektname über die ganze Breite (bis zwei Zeilen), darunter Status, Fortschrittsbalken,
  Anzahl erledigter von allen Einträgen (z. B. „5/10“) und Fälligkeit; die Statusfarbe erscheint nur noch einmal
- Gespeicherte Ansichten lassen sich umbenennen (Stift oder Doppelklick in der Seitenleiste)
- Checklisten: erledigte (durchgestrichene) Einträge sind dunkler und damit besser lesbar

### Behoben

- Seitenleiste: Entfernen/Umbenennen bei Favoriten und gespeicherten Ansichten war auf Geräten ohne Maus
  (Tablet, Handy) nicht erreichbar; die Knöpfe sind dort jetzt immer sichtbar
- Auswahllisten (z. B. Rolle beim Anlegen eines Benutzers) und Datumsfelder zeigten teilweise unsichtbare
  Einträge, wenn das Theme der App und das des Betriebssystems verschieden waren (z. B. App hell, Windows
  dunkel). Die Browser-Bedienelemente folgen jetzt dem Theme der App

## [0.0.5] – 2026-11-04

### Hinzugefügt

- Übersicht: neue Ansichten **Kalender** (Projekte am Fälligkeitsdatum, per Drag & Drop umplanen) und
  **Zeitachse** (Gantt: Balken vom Start bis zur Fälligkeit, ziehen verschiebt, Enden ziehen ändert
  Start/Fälligkeit; Zoom Wochen/Monate); einheitlicher Ansichts-Umschalter Karten · Board · Liste ·
  Kalender · Zeitachse
- Startdatum pro Projekt (Projektseite, Kontextmenü „Zeitraum“, Zeitachse); ohne Startdatum gilt das
  Erstelldatum
- Filterleiste für alle Ansichten der Übersicht: Kunde (mit „Alle“/„Keine“), Label und Verantwortlich als
  Dropdowns; im Board zusätzlich „Gruppieren“ nach Status, Kunde oder Verantwortlich und „Kompakt“.
  Die Filter werden pro Benutzer in der Datenbank gespeichert und sind auf allen Geräten gleich
- Gespeicherte Ansichten: aktuelle Ansicht mit Filtern unter einem Namen speichern; sie erscheinen in
  der Seitenleiste
- Board: ruhigere Karten (Labels als Farbpunkte, max. zwei plus „+N“; Fortschrittsbalken;
  Fälligkeit, Anzahl Checklisten, Verantwortliche/r und Team unten), Schnellaktionen beim Überfahren
  (Verantwortlich, Zeitraum, Favorit, Menü), „Projekt hinzufügen“ am Ende jeder Spalte
- Board: WIP-Limit pro Status-Spalte (z. B. „WIP 2/2“, rot sobald erreicht)
- Status (Planung, Aktiv, Review, Blockiert, Erledigt) für alle umbenennbar, mit eigener Farbe und
  WIP-Limit: direkt im Board per Klick auf den Spaltentitel (nur Admins; zurücksetzbar auf Standard)
- Tags (Labels) für Projekte mit frei wählbarer Farbe (54 vorgegebene Farben oder eine eigene): im
  Projekt zuweisen oder direkt neu anlegen, sichtbar in allen Ansichten der Übersicht sowie in der
  Projektliste des Kunden; Verwaltung (umbenennen, umfärben, löschen) unter Einstellungen → Projekte
- Board und Karten der Übersicht: Rechtsklick auf ein Projekt öffnet ein Menü zum Öffnen, für Checklisten (öffnen oder neu
  anlegen), Umbenennen, Status, Zeitraum, Verantwortliche Person, Tags und Favorit; auf Touch-Geräten
  über den „…“-Knopf der Karte. Rechtsklick auf einen Kunden: Kunde öffnen, neues Projekt, nur diesen
  Kunden anzeigen, ausblenden
- Board: Karten lassen sich auch innerhalb einer Spalte verschieben (Reihenfolge wird gespeichert
  und gilt für alle Benutzer)
- Verantwortliche Person pro Projekt (Projektseite, Board-Karte und Kontextmenü)
- Neue Projektseite: Kopfzeile mit Pfad, Status, Favorit und Aktionen (Duplizieren, Archivieren, Löschen,
  „+ Checkliste“), Fortschritt mit Klartext („37 % · 19 von 31 Einträgen · noch 50 Tage“), Reiter Checklisten ·
  Aktivität · Dateien · Notizen und rechts ein Eigenschaften-Panel (Status, Kunde, Verantwortlich, Start/Ende,
  Labels, Beobachter, Easy Redmine) mit Bearbeitung direkt im Panel sowie „Letzte Aktivität“
- Aktivitätsverlauf pro Projekt: erledigte Einträge (zusammengefasst), abgeschlossene/angelegte/gelöschte
  Checklisten, Status, Zeitraum, Verantwortlich, Labels, Redmine-Ticket und Dateien; bereits erledigte Einträge
  erscheinen rückwirkend
- Dateien pro Projekt hochladen (auch per Drag & Drop, max. 25 MB pro Datei), herunterladen und löschen; die Dateien
  liegen in der Datenbank und sind damit in jedem Backup enthalten
- Projekt duplizieren: Kopie mit Beschreibung, Verantwortlich, Labels und allen Checklisten (Einträge nicht erledigt,
  ohne Fälligkeiten), Status Planung
- Beobachter pro Projekt (Personen mit Zugriff auf den Kunden)
- Easy-Redmine-Ticket pro Projekt; die Redmine-Adresse stellt ein Admin unter Einstellungen → Projekte ein, daraus
  wird der Link gebildet
- Kartenansicht der Übersicht neu: eine Karte pro Kunde mit Ort, Fortschrittsring, Projekten (Status, Fortschritt,
  Fälligkeit) und Fusszeile (überfällige bzw. offene Einträge, nächste Fälligkeit, Team); Kunden mit überfälligen
  Einträgen stehen zuerst und sind rot umrandet; leere Kunden mit „+ Projekt aus Template“. Karten lassen sich
  einzeln oder mit „Alle ein-/ausklappen“ einklappen; eingeklappt bleiben Kopf und Fusszeile als Zusammenfassung
- Listenansicht der Übersicht neu: Kennzahlen (aktive Projekte, offene/überfällige Einträge, fällig in 14 Tagen,
  im Verzug, blockiert), pro Projekt die Checklisten als Unterzeilen (Status „nicht begonnen/läuft/n überfällig/
  fertig“, Fortschritt, Fälligkeit, Zuständig), Spalte **Zeit vs. Fortschritt** (Fortschrittsbalken mit Markierung
  der verstrichenen Zeit, „im Plan“ / „x % im Verzug“ / „überfällig“) und „Aktualisiert“ (letzte Aktivität)

### Geändert

- Seitenleiste: aufklappbarer Abschnitt „Kunden“ entfernt (Kunden über den Menüpunkt „Kunden“)
- Seitenleiste: „Zuletzt besucht“ entfernt (inkl. Server-Schnittstelle `/api/recent` und Tabelle
  `user_recent`, Migration 00017); stattdessen „Gespeicherte Ansichten“
- Lange Status-Namen werden in den Board-Spalten nicht mehr abgeschnitten, sondern umgebrochen
- README auf Englisch
- Übersicht, Meine Aufgaben, Kunden und Templates haben denselben Seitenkopf (Titel, Untertitel, Aktionen rechts)
  und dieselben Abstände; „Meine Aufgaben“ nutzt jetzt die volle Breite wie die anderen Seiten
- Server: Lese-Timeout 3 Minuten statt 30 Sekunden, damit Datei-Uploads auch über langsame Leitungen klappen
- Datenbank-Migrationen 00012–00018; sie werden beim Start automatisch angewendet (vorher ein Backup machen)

## [0.0.4] – 2026-11-04

### Hinzugefügt

- GitHub-Release wird bei jedem Versions-Tag automatisch angelegt, mit den Notizen aus dieser Datei
  (Workflow `release.yml`; optional Ankündigung auf Discord über das Secret `DISCORD_WEBHOOK`)

### Geändert

- GitHub Actions auf aktuelle Versionen (keine Node-20-Warnungen mehr)
- Testdaten und Code-Beispiele verwenden nur noch neutrale Beispielnamen

### Behoben

- Workflows akzeptieren beim manuellen Start das Tag mit oder ohne „v“ (`0.0.3` wie `v0.0.3`)

## [0.0.3] – 2026-11-04

### Hinzugefügt

- Image `:edge` mit dem aktuellen Entwicklungsstand, bei jedem Push auf `main` nach grünen Tests

### Geändert

- `docker-compose.yml` verwendet nur noch das veröffentlichte Image
  `ghcr.io/seals187/projectlist`, ohne `build`/`args`; Version über `PROJECTLIST_VERSION` in der `.env`
  (`latest` = neuestes Release, `edge` = Entwicklungsstand, oder z. B. `0.0.2`)
- Alle Code-Kommentare auf Englisch (Oberflächentexte unverändert)

### Behoben

- Kurzbeschreibungen der Schriften in den Einstellungen erscheinen jetzt in der gewählten Sprache (waren immer Deutsch)
- „Zuletzt besucht“: Besuche innerhalb derselben Millisekunde wurden manchmal in falscher Reihenfolge
  angezeigt; dadurch schlug auch der Test auf GitHub Actions zufällig fehl

## [0.0.2] – 2026-11-04

### Hinzugefügt

- `CHANGELOG.md` mit der Versionsgeschichte
- `.gitattributes`: Zeilenenden einheitlich LF (auch unter Windows), Bilder und Schriften als Binärdateien
- Container-Image auf GitHub (`ghcr.io/seals187/projectlist`) für linux/amd64 und linux/arm64,
  automatisch bei jedem Release-Tag (`:X.Y.Z`, `:X.Y`, `:latest`); bestehende Tags manuell nachveröffentlichbar
- GitHub Actions: Tests (go vet, go test, Frontend-Build) bei jedem Push auf `main` und bei Pull Requests

### Geändert

- Dockerfile baut Frontend und Go nativ und kompiliert für die Zielarchitektur (schnelle Multi-Arch-Builds
  ohne Emulation); OCI-Labels (Quelle, Lizenz, Version) verknüpfen das Image mit dem Repository

## [0.0.1] – 2026-11-04

Erste Version.

### Hinzugefügt

**Grundlagen**

- Selbst gehostete Projektverwaltung für mehrere Benutzer: Kunden → Projekte → Checklisten,
  Templates als Kopiervorlagen
- Ein einzelnes Go-Binary (net/http, SQLite, sqlc, goose) mit eingebettetem SvelteKit-Frontend;
  Docker-Image distroless (~29 MB, im Betrieb ~6 MB RAM), Compose mit 128-MB-Limit und Healthcheck
- Benutzer mit Rollen Admin, Member und Viewer sowie Kundenzuordnung; Argon2id, Sessions,
  CSRF- und Origin-Schutz, strenge Content-Security-Policy
- Backup per Befehl (`projectlist backup`) und als Download für Admins
- Oberfläche auf Englisch, Deutsch und Französisch
- Datenbank-Migrationen 00001–00011

**Übersicht**

- Ansichten Karten, Liste (mit Kennzahlen) und Board (Status-Spalten, Drag & Drop, Kundenfilter);
  Standardansicht in den Einstellungen wählbar, gerade genutzte Ansicht bleibt in der Sitzung erhalten
- Projektstatus Planung, Aktiv, Review, Blockiert, Erledigt und Archiviert; Fälligkeitsdatum,
  Team (zugewiesene Personen) und nächster offener Schritt je Projekt
- Projekte je Kunde einklappbar (Standard: eingeklappt)
- Dialog „Neues Projekt“ mit Kunde, Status, Fälligkeit und optional erster Checkliste aus einem Template

**Seitenleiste**

- Suche (Strg+K) und „+“ für ein neues Projekt
- Favoriten und „Zuletzt besucht“ (Kunden, Projekte, Checklisten, Templates), pro Benutzer in der
  Datenbank gespeichert und damit auf allen Geräten gleich
- Kundenbaum mit Kürzel, Projektanzahl und Status-Punkten; Kunden und Projekte per Klick auf-/zuklappbar
- Abschnitte einklappbar, Seitenleiste in der Breite verstellbar und ganz einklappbar (Strg+\)
- Unten Benutzer (→ Einstellungen), Theme-Umschalter und Abmelden; optionaler Untertitel über `APP_SUBTITLE`

**Kunden**

- Kundenübersicht als Liste (aufklappbar mit Projekten) oder Karten; Suche, Filter (alle, mit aktiven
  Projekten, kritisch, ohne Projekt) und Sortierung (letzte Aktivität, Name, Fortschritt, Fälligkeit,
  offene Aufgaben)
- Status je Kunde: Neu, Im Plan, Kritisch, Abgeschlossen
- Ort, Ansprechpartner und E-Mail je Kunde
- Je Projekt einstellbar, ob es in den Gesamtfortschritt des Kunden einfliesst; ausgenommene
  Projekte werden grau dargestellt, beim Kunden steht „ohne N Projekt(e)“

**Meine Aufgaben**

- Tab „Kunden“: Kunden → Projekte → Checklisten, an denen man beteiligt ist, mit Fälligkeit
  (Gestern/Heute/Morgen)
- Tab „Persönlich“: zugewiesene Einträge zum Abhaken
- Filter Offen, Heute und Erledigt (letzte 30 Tage)

**Checklisten-Editor**

- Blöcke: Überschriften H1–H4 (einklappbar), Text, Aufzählung, Nummerierung, Zitat, Codeblock mit
  Syntax-Highlighting, Trennlinie und Einträge mit Checkbox
- Einträge mit Zuständigen, Fälligkeit, Status-Text (z. B. „Erledigt“) und Notizen;
  Umwandeln eines Blocks in einen Eintrag mit einem Klick
- Abschnitts- und Zeilenfarben in drei Stufen (wie Word/Excel), Text- und Markerfarben,
  Fett/Kursiv/Unterstrichen/Durchgestrichen, Links mit Text und URL
- Rahmenlinien (ein-/ausschaltbar), Zoom 80–160 %, Rückgängig/Wiederholen, Drag & Drop
- Ganze Checklisten einer Person zuweisbar; Checkliste als Template speichern

**Darstellung**

- Themes Hell, Dunkel, Dunkelgrau und System
- 12 eingebettete Schriften (u. a. Inter, Geist, Mona Sans), Überschriften in Serif oder wie der Text,
  Code in JetBrains Mono; keine externen Anfragen
- Checkbox links oder rechts, eckig oder rund; Codeblock-Hintergrund Hellgrau, Dunkelgrau oder Schwarz

[Unveröffentlicht]: https://github.com/seals187/projectList/compare/v0.0.7...HEAD
[0.0.7]: https://github.com/seals187/projectList/compare/v0.0.6...v0.0.7
[0.0.6]: https://github.com/seals187/projectList/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/seals187/projectList/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/seals187/projectList/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/seals187/projectList/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/seals187/projectList/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/seals187/projectList/releases/tag/v0.0.1
