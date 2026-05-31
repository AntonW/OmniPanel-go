# Kapitel 9: Panels organisieren

Wenn deine Panels komplexer werden, brauchst du Möglichkeiten, Steuerelemente in logische Gruppen zu organisieren. Dieses Kapitel deckt Paged Container und Section Groups ab — zwei Blöcke, die dir helfen, deine Panels übersichtlich und einfach nutzbar zu halten.

## Paged Container-Block

### Was er macht

Ein Paged Container lässt dich mehrseitige Panels mit Tabs erstellen. Statt alles auf einen Bildschirm zu quetschen, kannst du Steuerelemente über mehrere Seiten verteilen und mit Tabs zwischen ihnen wechseln.

Stell es dir wie Browser-Tabs vor — jede Seite hat ihre eigenen Blöcke, und du klickst auf einen Tab, um die Seiten zu wechseln.

### Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Pages** | Liste der Seiten-/Tab-Namen | „Flight", „Weapons", „Systems" |
| **Tab Position** | Wo die Tabs erscheinen | Top, Bottom, Left, Right |
| **Current Page** | Welche Seite aktiv ist (1-basierter Index) | 1, 2, 3... |
| **Grid Size** | Die internen Rasterabmessungen für jede Seite | 12x12 |
| **Hintergrundbild** | Optionales Bild hinter dem Seiteninhalt | Beliebige Bild-URL |
| **Hintergrundfarbe** | Seiten-Hintergrundfarbe | Beliebige Farbe |
| **Tab-Farben** | Farben für aktive und inaktive Tabs | Beliebige Farben |

### Seiten erstellen

1. Ziehe einen **Paged Container** auf deine Arbeitsfläche
2. Klicke auf das Zahnrad-Symbol
3. In der Einstellung **Pages** gib deine Seitennamen ein:
   - Klicke auf „Seite hinzufügen" oder tippe Namen, getrennt durch Kommas
   - Beispiel: „Flight, Weapons, Systems, Navigation"
4. Wähle die Tab-Position (Top ist Standard und am häufigsten)
5. Speichere die Einstellungen

### Seiten nutzen

Wenn du das Panel öffnest:

- Tabs erscheinen an der Position, die du gewählt hast
- Klicke auf einen Tab, um zu dieser Seite zu wechseln
- Jede Seite hat ihre eigenen Blöcke
- Blöcke auf einer Seite überlappen nicht mit Blöcken auf einer anderen Seite
- Die aktive Seite wird gemerkt — wenn du das Panel speicherst und neu lädst, öffnet es sich auf derselben Seite, die du zuletzt angesehen hast

> **Tipp:** Du kannst auch die **Current Page**-Nummer im Eigenschaften-Panel setzen, um zu einer bestimmten Seite zu springen, ohne auf Tabs zu klicken.

### Blöcke zu Seiten hinzufügen

1. Wähle im Editor den Paged Container aus
2. Klicke auf den Tab der Seite, die du bearbeiten möchtest
3. Ziehe Blöcke in diese Seite
4. Wechsle die Tabs, um Blöcke zu anderen Seiten hinzuzufügen

### Tab-Position-Optionen

| Position | Am besten für |
|----------|----------|
| **Top** | Am häufigsten, wie Browser-Tabs |
| **Bottom** | Einfacher Thumb-Zugriff auf Tablets |
| **Left** | Vertikale Tab-Leiste, gut für viele Seiten |
| **Right** | Vertikale Tab-Leiste, alternatives Layout |

### Hintergrundbilder

Du kannst ein Hintergrundbild für den Paged Container setzen:

1. Platziere dein Bild im `user/assets/`-Ordner
2. Gib in den Einstellungen den Bildpfad ein: `assets/dein-bild.png`
3. Das Bild erscheint hinter allen Blöcken auf der Seite

> **Tipp:** Nutze dezente, dunkle Bilder, damit deine Blöcke sichtbar und lesbar bleiben.
>
> **Durchsuche deine Assets:** Du kannst alle deine hochgeladenen Assets ansehen indem du `http://dein-server:3000/assets/` in deinem Browser öffnest. Dies zeigt eine Verzeichnisliste von allem in `user/assets/`, was es einfach macht den richtigen Dateinamen für deinen Bildpfad zu finden.

---

## Section Group-Block

### Was er macht

Eine Section Group ist ein beschrifteter Container, der zusammengehörige Blöcke gruppiert. Er zeichnet einen Rahmen um seinen Inhalt mit einem Titel-Label, sodass klar ist, welche Steuerelemente zusammengehören.

Stell es dir wie eine beschriftete Box vor — alles innerhalb der Box gehört zum Label.

### Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Title** | Das Label oben am Bereich | „Flight Controls", „Weapons" |
| **Rahmenfarbe** | Farbe des Bereichs-Rahmens | Beliebige Farbe |
| **Hintergrundfarbe** | Farbe innerhalb des Bereichs | Beliebige Farbe (oft leicht anders als der Panel-Hintergrund) |
| **Titel-Farbe** | Farbe des Titel-Textes | Beliebige Farbe |

### Eine Section Group erstellen

1. Ziehe eine **Section Group** auf deine Arbeitsfläche
2. Klicke auf das Zahnrad-Symbol
3. Setze den Titel (z. B. „Flight Controls")
4. Passe die Farben an
5. Speichere die Einstellungen

### Blöcke zu einer Section Group hinzufügen

1. Ziehe im Editor Blöcke **innerhalb** des Section Group-Rahmens
2. Die Blöcke werden Kinder des Bereichs
3. Wenn du die Section Group verschiebst, bewegen sich alle ihre Blöcke mit
4. Wenn du die Section Group skalierst, gibt das mehr oder weniger Raum für ihre Blöcke

### Wann Section Groups nutzen

- **Nach Funktion gruppieren:** „Flight Controls", „Weapons", „Navigation"
- **Nach System gruppieren:** „Engines", „Shields", „Life Support"
- **Nach Priorität gruppieren:** „Primary Controls", „Secondary Controls"
- **Visuelle Organisation:** Mach dein Panel strukturiert und professionell

---

## Paged Container und Section Groups kombinieren

Du kannst beides zusammen für maximale Organisation nutzen:

1. Erstelle einen Paged Container mit Seiten: „Flight", „Combat", „Systems"
2. Auf der „Flight"-Seite, füge Section Groups hinzu: „Movement", „Camera", „Communications"
3. Auf der „Combat"-Seite, füge Section Groups hinzu: „Weapons", „Defense", „Targeting"
4. Auf der „Systems"-Seite, füge Section Groups hinzu: „Power", „Diagnostics", „Settings"

Das erstellt eine saubere Hierarchie:
- **Seiten** trennen Hauptkategorien
- **Bereiche** gruppieren zusammengehörige Steuerelemente innerhalb jeder Kategorie

---

## Design-Tipps für organisierte Panels

### Vor dem Bauen planen

Bevor du anfängst, Blöcke zu platzieren, skizziere dein Panel-Layout:

1. Was sind die Hauptkategorien von Steuerelementen?
2. Wie viele Steuerelemente pro Kategorie?
3. Welche Steuerelemente nutzt du am häufigsten?
4. Sollten häufig genutzte Steuerelemente auf der ersten Seite sein?

### Halte es einfach

- Erstelle nicht zu viele Seiten — 3 bis 5 Seiten sind meist genug
- Überlade Bereiche nicht — lass Platz zwischen Blöcken
- Nutze konsistente Bereichsgrößen für einen sauberen Look

### Farbkonsistenz

- Nutze dieselbe Rahmenfarbe für alle Section Groups
- Nutze verschiedene Hintergrundfarben, um Bereiche zu unterscheiden
- Lass die Tab-Farben zu deinem gesamten Panel-Theme passen

### Namenskonventionen

Nutze klare, konsistente Namen für Seiten und Bereiche:

- Gut: „Flight", „Combat", „Systems"
- Weniger ideal: „Seite 1", „Seite 2", „Seite 3"
- Gut: „Engine Controls", „Weapon Systems"
- Weniger ideal: „Gruppe 1", „Gruppe 2"

### Dein Layout testen

Nach dem Organisieren deines Panels:

1. Öffne es auf deinem Tablet
2. Versuche, zwischen Seiten zu wechseln
3. Stelle sicher, dass alle Blöcke sichtbar und erreichbar sind
4. Prüfe, ob du Buttons antippen kannst, ohne versehentlich den falschen zu treffen
5. Passe Größen und Positionen nach Bedarf an

---

## Praktische Beispiele

### Beispiel 1: Flugsimulator-Panel

**Seiten:**
- „Flight" — Schub, Klappen, Fahrwerk, Trimmung
- „Navigation" — Autopilot, Wegpunkte, Kompass
- „Systems" — Treibstoff, Motorstatus, Elektrik

**Section Groups auf der „Flight"-Seite:**
- „Thrust" — Schubregler, Nachbrenner-Button
- „Aerodynamics" — Klappenregler, Spoiler-Buttons
- „Landing" — Fahrwerk-Schalter, Brems-Button

### Beispiel 2: Rennspiel-Panel

**Seiten:**
- „Driving" — Gangwahl, Boost, Bremsbalance
- „Setup" — Reifendruck, Federung, Flügelwinkel
- „Info" — Rundenzeiten, Geschwindigkeit, Position

**Section Groups auf der „Driving"-Seite:**
- „Transmission" — Gang hoch/runter Buttons, Kupplungsregler
- „Performance" — Boost-Button, DRS-Schalter
- „Braking" — Bremsbalance-Regler, Bremsbias-Buttons

### Beispiel 3: Weltraumspiel-Panel

**Seiten:**
- „Flight" — Navigation, Geschwindigkeit, Andocken
- „Combat" — Waffen, Schilde, Gegenmaßnahmen
- „Power" — Energieverteilung, Reaktorstatus

**Section Groups auf der „Combat"-Seite:**
- „Weapons" — Waffengruppen-Buttons, Feuer-Schalter
- „Defense" — Schild-Steuerungen, Gegenmaßnahmen-Werfer
- „Targeting" — Zielverfolgung, Scan, Lock-On-Anzeige

---

## Ebenenbaum

Der Ebenenbaum (meist auf der rechten Seite des Editors) zeigt die Hierarchie aller Blöcke auf deinem Panel:

- Blöcke der obersten Ebene erscheinen an der Wurzel
- Blöcke innerhalb von Section Groups sind unter dem Bereich eingerückt
- Seiten in einem Paged Container zeigen ihre Kind-Blöcke

Nutze den Ebenenbaum, um:

- **Blöcke neu anzuordnen** — ziehen, um die Stapelreihenfolge zu ändern
- **Blöcke zu duplizieren** — auf das Duplizieren-Symbol klicken
- **Blöcke auszuwählen** — klicken, um sie auf der Arbeitsfläche auszuwählen
- **Hierarchie zu sehen** — verstehen, welche Blöcke in welchen Containern sind

## Was kommt als Nächstes?

Du möchtest Live-Daten auf deinem Panel anzeigen? Lerne mehr über [Live-Datenanzeige](10-live-data-display.md) — CPU-Auslastung, eigene Metriken und mehr anzeigen.
