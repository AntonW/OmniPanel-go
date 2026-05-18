# Kapitel 4: Dein erstes Panel erstellen

Dieses Kapitel führt dich Schritt für Schritt durch die Erstellung deines ersten Bedienpanels von Grund auf. Am Ende wirst du ein funktionierendes Panel mit Buttons und Schiebereglern haben, das du auf deinem Tablet öffnen kannst.

## Den Editor öffnen

Klicke auf der Startseite auf **Panel Editor** oder **Neues Panel**. Du siehst die Editor-Oberfläche mit:

- **Obere Menüleiste** — Neu, Laden, Speichern, Duplizieren, Löschen, Rückgängig, Wiederherstellen, Export, Import
- **Linke Seitenleiste** — die Block-Bibliothek, nach Kategorien sortiert, mit Suche
- **Mittlere Arbeitsfläche** — wo du Blöcke anordnest (mit Zoom/Verschieben-Steuerung)
- **Rechte Seitenleiste** — Tabs für Eigenschaften, Ebenen, Bindungen und Sprache

## Die Arbeitsfläche

Die Arbeitsfläche ist eine Leinwand, auf der du Blöcke platzierst und anordnest. Sie nutzt ein **12x12-Rastersystem** — stell dir die Arbeitsfläche in 12 Spalten und 12 Reihen unterteilt vor. Das hilft dir, Blöcke sauber auszurichten.

### Raster-Einrastung

Wenn du einen Block ziehst, rastet er automatisch am Raster ein. So bleibt alles ausgerichtet und dein Panel sieht professionell aus.

### Zoom und Verschieben

- **Zoom**: Halte `Strg` gedrückt und drehe das Mausrad, oder verwende die +/−-Buttons in der Werkzeugleiste
- **Verschieben**: Mittelklick und Ziehen, oder verwende die „Fit"-Taste, um die Ansicht zurückzusetzen
- **Auto-Fit**: Das Panel wird beim Öffnen des Editors automatisch zentriert und passend skaliert
- **Zoom-Stufe** wird in der Werkzeugleiste angezeigt (z. B. 100%)

### Seitenverhältnis

Oben auf der Arbeitsfläche kannst du eine Seitenverhältnis-Voreinstellung wählen:

- **16:9** — Breitbild (die meisten Tablets und Handys)
- **4:3** — ältere Tablets, einige iPads
- **Benutzerdefiniert** — eigenes Verhältnis einstellen

Diese Vorschau hilft dir zu sehen, wie dein Panel auf verschiedenen Geräten aussehen wird.

## Schritt 1: Deinen ersten Button hinzufügen

1. Finde in der linken Seitenleiste den **Button**-Block
2. Klicke und ziehe ihn auf die Arbeitsfläche
3. Lass ihn dort los, wo der Button erscheinen soll
4. Der Block rastet am Raster ein

## Schritt 2: Den Button konfigurieren

Jeder Block hat Einstellungen, die du anpassen kannst:

1. Klicke auf den Block, um ihn auszuwählen — der **Eigenschaften**-Tab in der rechten Seitenleiste öffnet sich automatisch mit den Block-Einstellungen
2. Die Einstellungen sind nach Kategorien gruppiert:
   - **Identität** — Label-Text
   - **Erscheinungsbild** — Farben, Schriftgröße, Border Radius
   - **Eingabe-Zuordnung** — Joystick-Index, Button-ID, Tastaturkürzel
   - **Befehl** — Befehlstyp, Shell-Befehl, HTTP-Methode/URL/Body (für Befehlsblöcke)
   - **Sprache** — Sprachauslöser und Aliase (falls zutreffend)
3. Ändere das Label zu „Gear"
4. Lass die Joystick-Einstellungen vorerst so, wie sie sind
5. Klicke außerhalb des Blocks, um die Auswahl aufzuheben (das Eigenschaften-Panel wird geleert)

## Schritt 3: Einen Slider hinzufügen

1. Ziehe einen **Slider**-Block aus der **Sliders**-Kategorie auf die Arbeitsfläche
2. Platziere ihn unter deinem Button
3. Wähle ihn aus und konfiguriere im Eigenschaften-Tab:
   - **Label** — ändere zu „Throttle"
   - **Joystick Index** — lass auf 0
   - **Slider ID** — welche Achse gesteuert werden soll (0 bis 7)
   - **Farben** — passe das Slider-Design an

## Schritt 4: Blöcke skalieren und positionieren

### Blöcke verschieben

- Greife das **Kreuz-Symbol** (Verschiebe-Griff) auf einem beliebigen Block
- Ziehe ihn an eine neue Position
- Lass los, um ihn am Raster einrasten zu lassen

### Blöcke skalieren

- Greife die **untere rechte Ecke** eines beliebigen Blocks
- Ziehe, um ihn größer oder kleiner zu machen
- Der Block rastet in Rastergrößen ein

### Tipps für das Layout

- Lass etwas Platz zwischen Blöcken, damit du nicht versehentlich den falschen antippst
- Gruppiere zusammengehörige Steuerelemente (alle Flugsteuerungen in einem Bereich, Waffen in einem anderen)
- Mache häufig genutzte Buttons größer, damit sie leichter anzutippen sind

## Schritt 5: Dein Panel speichern

1. Klicke auf den **Speichern**-Button in der oberen Werkzeugleiste
2. Ein Dialog fragt nach einem Panel-Namen
3. Gib einen Namen wie „Flight Controls" ein
4. Klicke auf Speichern

Dein Panel ist jetzt gespeichert und einsatzbereit.

## Schritt 6: Dein Panel öffnen

### Auf deinem PC

1. Gehe zurück zur Startseite
2. Du siehst deine „Flight Controls"-Panel-Karte
3. Klicke darauf, um das Panel zu öffnen

### Auf deinem Tablet

1. Öffne die Startseite auf deinem Tablet
2. Tippe auf die „Flight Controls"-Panel-Karte
3. Dein Panel öffnet sich mit dem Button und dem Slider

## Dein Panel testen

- **Tippe auf den Button** — er sollte kurz aufleuchten, um zu bestätigen, dass er eine Eingabe gesendet hat
- **Ziehe den Slider** — bewege ihn hoch und runter, um die Reaktion zu sehen
- **Prüfe dein Spiel** — wenn ein Spiel läuft und auf Joystick-Eingaben wartet, sollte es diese Eingaben erhalten

> **Hinweis:** Wenn kein Spiel läuft, werden die Eingaben trotzdem gesendet — es gibt nur nichts, das sie empfängt. Du kannst überprüfen, ob die Eingaben funktionieren, in [Kapitel 12: Virtuelle Joysticks](12-virtual-joysticks.md).

## Ein gespeichertes Panel laden

Um ein bereits erstelltes Panel zu bearbeiten:

1. Öffne den Editor
2. Klicke auf den **Laden**-Button in der oberen Menüleiste
3. Ein Dialog zeigt alle deine gespeicherten Panels
4. Klicke auf das Panel, das du bearbeiten möchtest
5. Das Panel wird in die Arbeitsfläche geladen

## Rückgängig und Wiederherstellen

Der Editor speichert bis zu 100 Änderungen:

- **Rückgängig**: `Strg+Z` oder der Rückgängig-Button in der Menüleiste
- **Wiederherstellen**: `Strg+Shift+Z` oder `Strg+Y`, oder der Wiederherstellen-Button
- Funktioniert für: Blöcke hinzufügen, verschieben, skalieren, Einstellungen ändern, duplizieren, löschen

## Blöcke löschen

- Wähle einen Block aus und drücke die **Entf**-Taste auf deiner Tastatur
- Oder klicke auf das **Mülleimer-Symbol** 🗑 in der oberen rechten Ecke des Blocks
- Du kannst auch auf das **Zahnrad-Symbol** ⚙ in der oberen linken Ecke klicken, um das Eigenschaften-Panel zu öffnen

## Blöcke duplizieren

- Im **Ebenen**-Tab (rechte Seitenleiste) klicke auf das **Duplizieren-Symbol** (⧉) neben einem Block
- Eine Kopie erscheint leicht versetzt vom Original

## Was kommt als Nächstes?

Jetzt, wenn du die Grundlagen kennst, tauchen wir tiefer ein in [Blöcke verstehen](05-understanding-blocks.md) — all die verschiedenen Steuerelement-Typen, die du zu deinen Panels hinzufügen kannst.
