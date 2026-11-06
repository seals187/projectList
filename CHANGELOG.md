# Changelog

Alle nennenswerten Änderungen an ProjectList werden in dieser Datei festgehalten.

Das Format basiert auf [Keep a Changelog](https://keepachangelog.com/de/1.1.0/),
die Versionsnummern folgen [Semantic Versioning](https://semver.org/lang/de/).
Änderungen bis zum nächsten Release sammeln sich unter **Unveröffentlicht**.

## [Unveröffentlicht]

## [0.1.2] – 2026-11-07

### Hinzugefügt

- Übersicht „Kalender“: Kalenderwochen (ISO) in einer schmalen ersten Spalte „KW“

### Geändert

- Übersicht „Kalender“: Der aktuelle Monat hebt sich deutlicher ab – die Tage der Nachbarmonate sind im hellen
  Design grauer, im dunklen Design dunkler, die Tage des Monats im dunklen Design etwas heller; Tageszahlen des
  Monats kräftiger. Gilt auch mit Hintergrundbild
- Übersicht „Board“: Projektnamen auf den Karten etwas kleiner (13 px), zugeklappt und aufgeklappt gleich gross
- Übersicht „Kalender“: passt mit Zurück-Leiste und Kalenderwochen wieder ganz ins Fenster (kein Scrollen)
- Zurück-Leiste: Die Übersicht zeigt im Verlauf das Icon und den Namen der damaligen Ansicht (z. B.
  „Übersicht · Kalender“), im Pfad nur „Übersicht“ mit dem Ansichts-Icon; ein Ansichtswechsel aktualisiert den
  Eintrag

## [0.1.1] – 2026-11-07

### Hinzugefügt

- Einheitliche Typ-Icons und -Farben für Kunden (grün, Gebäude), Projekte (blau, Ordner), Checklisten (violett,
  Liste) und Templates (orange, Dokument) – in Seitenleiste, Suche (Ctrl K), Breadcrumbs, Kopfbereich der
  Seiten, Übersicht (Karten, Liste, Board), Kundenliste, Meine Aufgaben, Aktivitätslog, Template-Bibliothek und
  Template-Auswahl; im dunklen Design eigene Farben
- Favoriten neu: zweizeilige Einträge mit Typ-Icon, Name und „Typ · Kontext“ (Projekt → Kunde, Checkliste →
  Projekt, Kunde → Anzahl Projekte, Template → Anzahl Einträge); eigene Gruppen mit Lesezeichen-Icon und Anzahl,
  einklappbar (pro Benutzer gespeichert); Rechtsklick bzw. langes Drücken: Öffnen, In neuem Tab öffnen, In
  Gruppe verschieben (auch in eine neue Gruppe), Aus Favoriten entfernen; Gruppen umbenennen und löschen
  (Einträge wandern nach „Ohne Gruppe“); Drag & Drop innerhalb und zwischen Gruppen
- Templates lassen sich als Favorit speichern (Stern im Kopf der Template-Seite)
- Zurück-Leiste über jeder Seite: Zurück und Vorwärts sowie der Pfad der aktuellen Seite mit Typ-Icons (z. B.
  Übersicht / Kunden / Kunde / Projekt / Checkliste, jeweils anklickbar; die Übersicht als Startseite steht immer
  vorne). Langes Drücken auf ← oder Alt+← zeigt den Verlauf
  („Zurück zu“ mit Typ bzw. „Seite“), bedienbar auch mit Pfeiltasten und Enter, mit Papierkorb zum Löschen des
  Verlaufs; der Verlauf zeigt alle besuchten Seiten, auch die Hauptseiten und die Einstellungen mit Unterseite
  (z. B. „Einstellungen · Darstellung“); er bleibt beim Neuladen
  des Tabs erhalten. Die bisherigen Pfadangaben in den Seitenköpfen entfallen
- Checklisten als Tabs: Jede geöffnete Checkliste erscheint als Tab über der Checkliste (Typ-Icon, Name,
  Fortschritt, roter Punkt bei überfälligen Einträgen). Klick wechselt, ✕ oder mittlere Maustaste schliesst
  (danach der Nachbar-Tab), „+“ sucht und öffnet eine weitere Checkliste, Tabs lassen sich per Drag & Drop
  umsortieren; die Tabs bleiben pro Benutzer im Browser gespeichert (max. 12)
- Kunden: Telefonnummern (mehrere, jeweils mit Bezeichnung wie „Zentrale“ oder „Mobil“) auf der Kundenseite;
  im Bearbeiten-Modus hinzufügen, ändern und entfernen, sonst als anklickbare Telefon-Links (Migration 00023)
- Einstellungen → „Über“ (ganz unten in der Navigation): aktuelle Version, Link zum GitHub-Repository und zu
  den Release-Notizen

### Geändert

- Favoriten werden nur noch über den Stern im Kopfbereich von Kunde, Projekt, Checkliste und Template gesetzt
  (nicht mehr auf Board-Karten und in den Kontextmenüs); in der Favoritenliste gibt es keinen Stern mehr
- „Kunden“ in der Seitenleiste und im Kopf der Kundenliste: Gebäude-Icon (ohne Farbe) wie das Kunden-Typ-Icon
- Kunden zeigen überall das Kunden-Icon statt des Kürzels („AB“): Kopf der Kundenseite, Übersicht (Karten,
  Liste, Board, Kalender, Zeitachse), Meine Aufgaben, Kundenliste (Liste und Karten), Kundenfilter,
  Kontextmenüs, Projektseite und Benutzerverwaltung
- Checklisten: Trennlinien deutlich dunkler (wie Rahmenlinien in Excel), auch zur Notizen-Spalte
- Ein zweiter Klick auf den eigenen Namen unten links schliesst die Einstellungen wieder und führt zurück zur
  vorher geöffneten Seite
- Übersicht „Zeitachse“: Ansichten „Diese Woche“ (Mo–So), „14 Tage“ und „30 Tage“ (ab heute) sowie „Monate“;
  die Tage nutzen die ganze Breite, die Pfeile springen um eine Ansichtslänge, die gewählte Ansicht bleibt
  gespeichert (pro Browser). Balken, die vor dem sichtbaren Bereich beginnen, zeigen ihren Namen am linken Rand
- Projektseite: „Letzte Aktivität“ zeigt nur den neuesten Eintrag und lässt sich aufklappen (dann die letzten
  fünf); das Feld „Easy Redmine“ ist aus den Eigenschaften entfernt (eine gespeicherte Ticket-Nummer bleibt in
  der Datenbank erhalten)

### Behoben

- Übersicht: Die Kästen (Karten, Liste, Board-Spalten) waren runder als auf Meine Aufgaben, Kunden und Templates;
  jetzt haben alle dieselben Ecken
- Mit Hintergrundbild und Glas-Effekt waren die farbigen Typ-Icons in Seitenleiste und Leisten zu kräftig; sie
  sind dort jetzt dezent getönt
- Dunkles Design mit Hintergrundbild: Checklisten, Template-Editor und Einstellungen waren leicht
  durchscheinend; ihr Inhaltsbereich ist jetzt deckend
- Eingabefelder in Menüs (z. B. „Neue Gruppe …“ bei den Favoriten) bekamen beim Öffnen keinen Fokus
- Hintergrundbild mit ausgeschaltetem Glas-Effekt: Seitenleiste, Zurück-Leiste und Kopfband von Kunden- und
  Projektseiten waren auch im hellen Design dunkel; sie sehen jetzt aus wie ohne Hintergrundbild (hell bzw. dunkel)
- Kopfband (Übersicht, Meine Aufgaben, Kunden, Templates): Bei einer Schriftgrösse ungleich 100 % waren die
  Bänder mit Schnellfilter-Chips (Meine Aufgaben, Kunden) etwas höher als die anderen; jetzt sind alle gleich hoch
- Übersicht (und Meine Aufgaben, Kunden, Templates): Nach dem Wechsel zurück auf eine Standardeinstellung (z. B.
  von „Zeitachse“ zurück auf „Karten“) blieb die alte Einstellung in der Adresse stehen; ein Neuladen der Seite
  sprang deshalb wieder auf die vorherige Ansicht
- Checklisten mit Trennlinien: Die senkrechte Linie zur Notizen-Spalte lag bei Überschriften (H1–H4) 1 px weiter
  rechts als bei Zeilen mit Checkbox
- Gruppen und Reihenfolge der Favoriten liegen jetzt in der Datenbank (Migration 00022); bestehende Ordner
  werden automatisch übernommen

- Checklisten und Templates: Die Bedienelemente einer Zeile (links „+“ und Verschiebe-Griff, rechts Status,
  Person, Fälligkeit, Farbe) haben im hellen und dunklen Design deutlich mehr Kontrast; der Griff ist etwas
  grösser

## [0.1.0] – 2026-11-06

### Hinzugefügt

- Backup wiederherstellen direkt in der App: *Einstellungen → Backup & Restore → Wiederherstellen*. Die Datei
  wird zuerst geprüft (SQLite, Integrität, ProjectList-Datenbank, nicht neuer als die laufende Version,
  mindestens ein Admin); danach legt die App eine Sicherheitskopie des aktuellen Stands im Datenverzeichnis ab
  (`pre-restore-<Zeit>.db`) und übernimmt das Backup ohne Neustart. Ältere Backups werden automatisch
  migriert. Der Admin bleibt angemeldet, wenn er im Backup Admin ist, sonst meldet er sich neu an
- Gespeicherte Ansichten: Ordner und Reihenfolge wie bei den Favoriten – per Drag & Drop verschieben, per
  Rechtsklick Ordner anlegen, umbenennen, löschen und Ansichten in Ordner verschieben; gilt auf allen Geräten

### Geändert

- Einstellungen: „Backup“ heisst jetzt „Backup & Restore“

## [0.0.9] – 2026-11-06

### Hinzugefügt

- Einstellungen neu aufgebaut: links eine eigene Navigation mit Suche („Einstellung suchen …“) und den Gruppen
  „Persönlich“ (Profil & Konto, Darstellung, Schrift, Checklisten, Übersicht) und „Arbeitsbereich“ (Tags,
  Benutzer & Rechte, Backup; nur Admins), rechts die Seite mit Karten und Zeilen
- Darstellung: Farbschema „System“ (folgt dem Betriebssystem), Akzentfarbe (Violett, Blau, Petrol, Grün, Orange,
  Rot, Pink), Glas-Effekt ein/aus, Hintergrundbild abdunkeln, Zeilenabstand (Kompakt/Normal/Luftig) und
  Schriftgrösse der ganzen App (80–120 %). Sprache, Farbschema und Darstellung gelten jetzt auf allen Geräten
  des Benutzers
- Schrift & Checklisten mit Live-Vorschau einer Checkliste
- Favoriten: Neben Projekten lassen sich jetzt auch Kunden und Checklisten als Favorit speichern (Stern neben dem
  Titel der Kunden-, Projekt- oder Checklisten-Seite, bei Kunden und Projekten auch im Kontextmenü). In der
  Seitenleiste stehen sie in der Reihenfolge, in der sie hinzugefügt wurden (Checklisten nur mit ihrem Namen).
  Bestehende Projekt-Favoriten bleiben erhalten
- Favoriten ordnen: per Drag & Drop nach oben oder unten schieben; per Rechtsklick in der Favoritenliste Ordner
  anlegen (umbenennen per Rechtsklick oder Doppelklick, auf- und zuklappen, löschen – die Favoriten bleiben dann
  erhalten). Favoriten lassen sich in Ordner ziehen oder per Rechtsklick „In Ordner verschieben“, Ordner selbst
  ebenfalls verschieben. Reihenfolge und Ordner gelten auf allen Geräten des Benutzers
- Tags: Filter, „+ Tag“, Menü „…“ pro Tag mit Bearbeiten (Name und Farbe) und Löschen
- Benutzer & Rechte: Suche, Rollen-Filter, Spalte „Zuletzt aktiv“, Rolle direkt in der Tabelle änderbar,
  Detailbereich mit Rolle, Kunden-Zugriff („Alle auswählen“), „Passwort zurücksetzen“ und „Benutzer entfernen“
- Benutzer einladen: Admins laden per E-Mail ein; der Link (7 Tage gültig) führt auf eine Seite zum Festlegen
  des Passworts. „Passwort zurücksetzen“ schickt ebenso einen Link (24 Stunden gültig). Ohne eingerichteten
  Mailversand zeigt die App den Link zum Kopieren an. Neue Umgebungsvariablen `APP_URL` und `SMTP_*`
  (siehe README)

### Geändert

- Benutzer werden nicht mehr mit einem vom Admin gesetzten Passwort angelegt, sondern eingeladen
- Übersicht „Board“: Zugeklappte Karten sind kompakter (weniger Abstand zwischen und in den Karten, Projektname
  etwas kleiner); Aufbau und Design bleiben gleich
- Checklisten: Das Statusfeld (z. B. „Erledigt“, „Terminée“, „Effectué“) steht direkt rechts neben der Checkbox
  und ist auch leer als „Status“ sichtbar; vorher war es ohne Inhalt unsichtbar, bis man über die Zeile fuhr.
  Bei Checkbox links steht es zwischen Checkbox und Text
- Die Grösse der Checklisten-Ansicht (100 % …) wird nur noch in der Checklisten-Werkzeugleiste eingestellt;
  in den Einstellungen gibt es dafür die Schriftgrösse der ganzen App

## [0.0.8] – 2026-11-06

### Hinzugefügt

- Einheitlicher Kopfbereich für Übersicht, Meine Aufgaben, Kunden und Templates (`PageHeader`): graue Kopf-Karte
  mit Seiten-Icon, Titel, Untertitel, Ansicht-Umschalter und Hauptaktion; darunter ein Streifen mit Suche,
  Filtern, Schnellfiltern, „Ansicht speichern“, Sortierung und „Alle einklappen / aufklappen“. Bei wenig Platz
  werden die Filter zu „Filter (n)“ zusammengefasst, danach zeigt der Ansicht-Umschalter nur noch Icons
- Neu: Suche in Übersicht, Meine Aufgaben und Templates; Sortierung in Übersicht, Meine Aufgaben und Templates;
  Filter „Kunde“ und „Fälligkeit“ in Meine Aufgaben, Filter „Status“ bei Kunden
- Ansicht, Filter und Sortierung stehen in der URL (teilbare Links) und werden pro Seite und Benutzer gespeichert;
  „Ansicht speichern“ funktioniert jetzt auf allen vier Seiten
- Frontend-Tests mit Vitest (laufen im Docker-Build mit)
- Übersicht „Liste“ neu: pro Kunde eine Karte mit Kürzel, Name, Zusammenfassung („3 Projekte · 5/51 Schritte ·
  46 offen · nächste 13.11.“) und Fortschrittsring; Projektzeilen mit Status, Fortschritt, Schritten, Fälligkeit
  und Zuständig, ganze Zeile klickbar. Bei breitem Fenster Schritte als eigene Spalte und Tags neben dem Namen,
  sonst Schritte neben dem Prozentwert und Tags darunter; Tags werden auf die verfügbare Breite gekürzt („+N“)
- Knopf „Anzeige“ im Kopfbereich aller Seiten: Schalter pro Seite und Ansicht (z. B. Schritte, Fortschritt in %,
  alle Tags, Fälligkeit, Zuständig), pro Benutzer auf dem Server gespeichert (Ersatz: Browser)

### Geändert

- Inhalte (Karten, Listen, Kacheln) sind mit Hintergrundbild jetzt deckend weiss bzw. dunkelgrau; nur der
  Seitenhintergrund und die Kopf-Karte bleiben farbig/durchscheinend
- Standardfarben der Status (ohne eigene Farbe): In Arbeit grün, Abnahme orange, Erledigt dunkelgrün
- Deutsch: „Zuständig“ statt „Verantwortlich“ (Filter, Spalten, Projekt)
- Übersicht „Kalender“: passt samt Liste „Ohne Fälligkeit“ ohne Scrollen ins Fenster (Wochenzeilen teilen sich
  die Höhe, volle Tage scrollen innerhalb der Zelle); kompaktere Tageszahlen und Einträge
- Übersicht „Karten“: Projektzeilen neu angeordnet – Name mit Ordner-Icon und Status-Pill rechts, darunter die
  Labels (so viele wie Platz haben, dann „+N“), darunter Fortschrittsbalken mit Prozent und Fälligkeit (dd.mm.)
- Ein Klick auf „Übersicht“ in der Seitenleiste setzt die Filter (Kunde, Label, Verantwortlich) auf „Alle“
  zurück; gespeicherte Ansichten wenden weiterhin ihre eigenen Filter an
- Übersicht: „Gruppieren nach“ im Board ist jetzt ein Dropdown im selben Stil wie die Filter Kunde, Label und
  Verantwortlich (gleicher Knopf, gleiches Menü); hervorgehoben nur, wenn nicht nach Status gruppiert wird
- Status-Badges (Übersicht „Liste“, Kunden, Projekt, Meine Aufgaben) haben für jeden Status denselben Stil:
  Hintergrund in der Board-Farbe des Status, Schrift immer schwarz (in beiden Designs)
- Kundenliste: Bei aufgeklappten Projekten steht der Projektstatus in der Spalte „Status“ (statt unter
  „Projekte“), der Prozentwert neben dem Fortschrittsbalken
- Nur noch zwei Designs: Hell und Dunkel („Dunkelgrau“ wird zu Dunkel; ohne Auswahl entscheidet einmalig die
  Systemeinstellung). Der Knopf in der Seitenleiste wechselt direkt zwischen Hell und Dunkel
- Dropdown- und Kontextmenüs (z. B. Kunde, Label, Verantwortlich, Gruppieren nach) sind mit Hintergrundbild im
  hellen Design nicht mehr dunkel, sondern folgen dem gewählten Design
- Mit Hintergrundbild haben die Kacheln in Übersicht und „Meine Aufgaben“ dieselbe Helligkeit wie bei Kunden
  und Templates
- Mit Hintergrundbild sind Seitenleiste und Kopfband im dunklen Design jetzt dunkel (fast deckend) statt
  halbtransparent wie im hellen Design
- Dropdown- und Kontextmenüs sind im hellen Design nicht mehr reinweiss, sondern leicht durchscheinend
  (Milchglas wie die Kacheln); im dunklen Design ebenfalls leicht durchscheinend
- Mit Hintergrundbild hat der Inhaltsblock von Checklisten, Template-Editor und Einstellungen im dunklen Design
  denselben Farbton wie die Seitenleiste (vorher deckend schwarz)
- Dunkles Design ohne Hintergrundbild: Der Inhaltsbereich (Übersicht, Meine Aufgaben, Einstellungen usw.) hat
  dieselbe Farbe wie die Seitenleiste statt fast schwarz
- Native Auswahllisten sind im aufgeklappten Zustand auch mit Hintergrundbild und im dunklen Design gut lesbar
- Favoriten in der Seitenleiste zeigen das Kundenkürzel nicht mehr zusätzlich rechts an

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

[Unveröffentlicht]: https://github.com/seals187/projectList/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/seals187/projectList/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/seals187/projectList/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/seals187/projectList/compare/v0.0.9...v0.1.0
[0.0.9]: https://github.com/seals187/projectList/compare/v0.0.8...v0.0.9
[0.0.8]: https://github.com/seals187/projectList/compare/v0.0.7...v0.0.8
[0.0.7]: https://github.com/seals187/projectList/compare/v0.0.6...v0.0.7
[0.0.6]: https://github.com/seals187/projectList/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/seals187/projectList/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/seals187/projectList/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/seals187/projectList/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/seals187/projectList/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/seals187/projectList/releases/tag/v0.0.1
