# Kapitel 5: Blöcke verstehen

Blöcke sind die Bausteine deiner Panels — buchstäblich. Jeder Button, Slider, Touchpad und jede Anzeige auf deinem Panel ist ein Block. Dieses Kapitel erklärt, wie Blöcke funktionieren und wie du sie effektiv nutzt.

## Was ist ein Block?

Ein Block ist ein einzelnes UI-Element auf deinem Panel. Stell dir Blöcke wie LEGO-Teile vor — jedes macht etwas Bestimmtes, und du kombinierst sie, um dein Bedienpanel zu bauen.

## Block-Themes

Jeder Block kann mit einem von vier Themes gestaltet werden. Themes bestimmen das Aussehen — Schriftarten, Farben, Gradienten und Spezialeffekte. Du kannst ein **Standard-Theme für das gesamte Panel** festlegen und **es pro Block überschreiben**.

### Verfügbare Themes

**Standard** — Sauberes, neutrales Design, das zu jedem Spiel passt. Verwendet Systemschriftarten und einfache Schatten.

**Star Citizen** — Sci-Fi-Stil mit abgeschrägten Ecken, cyanfarbenen Leuchteffekten und der Rajdhani-Schriftart. Perfekt für Weltraum-Simulationen.

**KDE Breeze** — Flaches, klares Design basierend auf dem KDE Plasma Breeze Dark Desktop-Theme. Volle Farben, keine Leuchteffekte, dezente Hover-Änderungen und der Breeze-Blau-Akzent. Verwendet die Noto Sans-Schriftart.

**Windows 11** — Fluent Design mit flachen Oberflächen, abgerundeten Ecken, der Segoe UI-Schriftart und der Windows-Akzentfarbe.

### Panel-Theme festlegen

In den Workspace-Einstellungen (unten rechts im Editor) findest du ein **Panel Theme**-Dropdown. Dies setzt das Standard-Theme für alle neuen Blöcke, die du zum Panel hinzufügst.

### Theme eines Blocks ändern

1. Wähle einen Block auf der Arbeitsfläche aus
2. Öffne den **Properties**-Tab (rechte Seitenleiste)
3. Finde die **Theme**-Gruppe oben
4. Wähle ein anderes Theme aus dem Dropdown
5. Der Block aktualisiert sich sofort

Du kannst Themes im selben Panel mischen. Zum Beispiel: Star Citizen für Flugsteuerungen und KDE Breeze für Mediensteuerungen.

## Die Block-Bibliothek

Die Block-Bibliothek befindet sich in der linken Seitenleiste des Editors. Sie zeigt alle verfügbaren Block-Typen, organisiert nach Kategorie:

- **Input Controls** — Button, Toggle, Emergency Button
- **Sliders** — Slider, Slider Vertical, Power Slider
- **Pointing** — Mousepad, Touch Pad
- **Commands** — Command Block
- **Speech** — Speech Command, Push to Talk
- **Data Display** — Data Display
- **Containers** — Paged Container, Section Group

Jeder Block-Typ erscheint einmal — du wählst sein Theme im Properties-Panel, nachdem du ihn hinzugefügt hast.

## Blöcke zu deinem Panel hinzufügen

1. Finde den gewünschten Block in der linken Seitenleiste
2. Klicke und ziehe ihn auf die Arbeitsfläche
3. Lass los, um ihn zu platzieren
4. Der Block rastet am Raster ein

## Blöcke konfigurieren

Jeder Block hat ein **Zahnrad-Symbol**, das seine Einstellungen öffnet. Die Einstellungen variieren je nach Block-Typ, aber häufige Optionen sind:

### Häufige Einstellungen

- **Label** — Text, der auf oder über dem Block angezeigt wird
- **Farben** — Hintergrund-, Text-, Rahmen- und Akzentfarben
- **Schriftgröße** — Größe des Label-Textes
- **Joystick Index** — Welcher virtuelle Controller genutzt werden soll (0 bis 9)

### Joystick Index und IDs

Diese Einstellungen sagen OmniPanel-go, welche virtuelle Controller-Eingabe genutzt werden soll:

- **Joystick Index (0-9)** — Welcher virtuelle Controller. Wenn du 4 Joysticks hast, nutze 0, 1, 2 oder 3
- **Button ID (0-15)** — Welcher Button auf diesem Controller (jeder Joystick hat 16 Buttons)
- **Slider/Achsen ID (0-7)** — Welche Achse auf diesem Controller (jeder Joystick hat 8 Achsen)

> **Beispiel:** Ein Button mit Joystick Index 0 und Button ID 3 steuert den 4. Button des 1. virtuellen Joysticks.

## Blöcke verschieben und skalieren

### Verschieben

- Greife das **Kreuz-Symbol** (Verschiebe-Griff) auf dem Block
- Ziehe ihn an eine neue Position
- Lass los, um am Raster einrasten zu lassen

### Skalieren

- Greife den Griff in der **unteren rechten Ecke**
- Ziehe, um die Größe zu ändern
- Rastet in Raster-Schritten ein

## Blöcke duplizieren

Du brauchst mehrere ähnliche Buttons? Statt jeden einzeln zu erstellen:

1. Finde den Block im **Ebenenbaum** (meist auf der rechten Seite des Editors)
2. Klicke auf das **Duplizieren-Symbol** neben dem Block
3. Eine Kopie erscheint auf der Arbeitsfläche
4. Konfiguriere die Kopie mit anderen Einstellungen

## Blöcke löschen

So entfernst du einen Block:

1. Wähle den Block auf der Arbeitsfläche aus
2. Drücke die **Entfernen**-Taste auf deiner Tastatur
3. Oder nutze die Löschoption im Ebenenbaum

## Das Rastersystem

Die Arbeitsfläche nutzt ein **12x12 prozentbasiertes Raster**:

- 12 Spalten horizontal
- 12 Reihen vertikal
- Position und Größe jedes Blocks werden in Rastereinheiten gemessen

### Warum ein Raster?

- Hält alles ausgerichtet und ordentlich
- Macht Panels professionell
- Stellt sicher, dass Blöcke auf verschiedenen Bildschirmgrößen richtig skalieren
- Verhindert überlappende Steuerelemente

### Raster-Tipps

- Kleine Blöcke (1-2 Rastereinheiten) — gut für einfache Buttons
- Mittlere Blöcke (3-4 Rastereinheiten) — gut für beschriftete Steuerelemente
- Große Blöcke (5+ Rastereinheiten) — gut für Touchpads und Anzeigen

## Eingabe-Zuordnungsansicht

Der Editor zeigt, welche Joystick-Buttons und -Achsen bereits verwendet werden. Das hilft dir zu vermeiden, dieselbe Eingabe mehreren Blöcken zuzuordnen.

Suche nach dem Bereich **Eingabe-Zuordnungen** im Editor — er zeigt eine Liste verwendeter Joystick-Eingaben, sodass du ungenutzte für neue Blöcke auswählen kannst.

## Block-Einstellungen im Detail

Wenn du auf das Zahnrad-Symbol eines Blocks klickst, siehst du Einstellungen, die spezifisch für diesen Block-Typ sind. Hier ist, was die häufigsten Einstellungen bedeuten:

### Farben

Die meisten Blöcke lassen dich anpassen:

- **Hintergrundfarbe** — die Hauptfüllfarbe
- **Textfarbe** — die Farbe des Label-Textes
- **Rahmenfarbe** — die Umrissfarbe
- **Akzentfarbe** — Spezialeffekte (Fortschrittsbalken, aktive Zustände, etc.)

Klicke auf ein beliebiges Farbfeld, um eine Farbauswahl zu öffnen. Du kannst:

- Aus voreingestellten Farben wählen
- Einen Hex-Code eingeben (wie `#FF0000` für Rot)
- Die Farbauswahl nutzen, um jede beliebige Farbe zu wählen

### Labels

Das Label ist der Text, der auf oder über dem Block angezeigt wird. Halte Labels kurz und klar:

- Gut: „Gear", „Throttle", „Flaps"
- Weniger ideal: „Landing Gear Control Button", „Engine Throttle Slider"

## Block-Konfigurationen speichern

Block-Einstellungen werden als Teil des Panels gespeichert. Wenn du das Panel speicherst, werden alle Block-Konfigurationen beibehalten. Du musst einzelne Blöcke nicht separat speichern.

## Was kommt als Nächstes?

Jetzt, wenn du Blöcke verstehst, schauen wir uns die häufigsten im Detail an: [Buttons und Slider](06-buttons-and-sliders.md).
