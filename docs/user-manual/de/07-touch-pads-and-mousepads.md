# Kapitel 7: Touchpads und Mousepads

Touchpads und Mousepads lassen dich analoge Eingaben steuern, indem du deinen Finger über eine Fläche ziehst — genau wie bei einem Joystick oder Laptop-Trackpad. Dieses Kapitel deckt beide Block-Typen ab und wie du sie konfigurierst.

## Touch Pad-Block

### Was er macht

Ein Touch Pad ist ein virtueller Joystick. Du ziehst deinen Finger in einem kreisförmigen Bereich herum, und OmniPanel-go sendet Achsenwerte an dein Spiel, basierend darauf, wo dein Finger ist. Stell es dir vor wie den Analog-Stick auf einem Gamecontroller.

### Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Label** | Text über dem Touchpad | „Camera", „Flight Stick", „Turret" |
| **Joystick Index** | Welcher virtueller Controller (0-9) | 0 für ersten Controller |
| **Axis ID** | Welches Achsenpaar genutzt werden soll (0-3) | 0 nutzt X- und Y-Achse |
| **Farbe** | Farbe des Touchpad-Rings und der Anzeige | Beliebige Farbe |
| **Travel Distance** | Wie weit sich die Anzeige vom Zentrum bewegt (in Pixeln) | 50 bis 150 |
| **Safe Zone** | Totzone in der Mitte, wo kleine Bewegungen ignoriert werden | 0 bis 30 |

### So funktioniert es

1. Berühre das Pad und ziehe deinen Finger
2. Ein Punkt bewegt sich in die Richtung, in die du ziehst
3. Der Abstand vom Zentrum bestimmt den Achsenwert
4. Lass deinen Finger los und der Punkt schnappt zurück ins Zentrum

### Achsen-Zuordnung

Jedes Touch Pad steuert **zwei Achsen** gleichzeitig:

- **Axis ID 0** — nutzt Achse 0 (horizontal) und Achse 1 (vertikal)
- **Axis ID 1** — nutzt Achse 2 (horizontal) und Achse 3 (vertikal)
- **Axis ID 2** — nutzt Achse 4 (horizontal) und Achse 5 (vertikal)
- **Axis ID 3** — nutzt Achse 6 (horizontal) und Achse 7 (vertikal)

### Wertebereich

- **Zentrum-Position** — beide Achsen bei 127 (neutral)
- **Ganz links** — horizontale Achse bei 0
- **Ganz rechts** — horizontale Achse bei 255
- **Ganz oben** — vertikale Achse bei 0
- **Ganz unten** — vertikale Achse bei 255

### Travel Distance

Diese Einstellung steuert, wie weit der Anzeigepunkt sich vom Zentrum bewegen kann:

- **Kleinere Werte (50-80)** — der Punkt bewegt sich nicht weit, gut für präzise kleine Anpassungen
- **Mittlere Werte (80-120)** — ausgewogen für allgemeine Nutzung
- **Größere Werte (120-150)** — der Punkt bewegt sich stark, gut für Vollbereichs-Steuerung

### Safe Zone (Totzone)

Die Safe Zone ist ein kleiner Bereich in der Mitte, in dem winzige Fingerbewegungen ignoriert werden. Das verhindert versehentliche Eingaben durch zittrige Hände.

- **0** — keine Totzone, jede Bewegung wird registriert
- **10-20** — kleine Totzone, empfohlen für die meisten Anwendungen
- **20-30** — große Totzone, nur absichtliche Bewegungen werden registriert

### Wann ein Touch Pad nutzen

- Kamerasteuerung in Flugsimulationen
- Geschützturm-Zielen in Fahrzeug-Kampfspielen
- Charakterbewegung in Third-Person-Spielen
- Jede Situation, in der du sanfte analoge Steuerung in zwei Richtungen brauchst

---

## Star Citizen Themen-Touch Pad

### Was er macht

Dieselbe Funktionalität wie das Standard-Touch Pad, aber mit einem Sci-Fi-Visuallstil — leuchtende Ringe und eckige Designelemente.

### Einstellungen

Dieselben wie das Standard-Touch Pad, mit zusätzlichen visuellen Anpassungsoptionen für die Sci-Fi-Ästhetik.

---

## Mousepad-Block

### Was er macht

Ein Mousepad simuliert ein Laptop-Trackpad. Ziehe deinen Finger, um den Mauszeiger auf deinem PC zu bewegen. Das ist anders als ein Touch Pad — statt Joystick-Achsenwerte zu senden, sendet es echte Mausbewegungen.

### Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Label** | Text über dem Mousepad | „Mouse", „Cursor-Steuerung" |
| **Mousepad Index** | Welche virtuelle Maus genutzt werden soll (meist 0) | 0 |
| **Empfindlichkeit** | Wie schnell sich der Cursor im Verhältnis zu deinem Finger bewegt | 1 bis 20 |
| **Farbe** | Farbe der Mousepad-Oberfläche | Beliebige Farbe |

### So funktioniert es

1. Berühre das Mousepad und ziehe deinen Finger
2. Der Mauszeiger auf deinem PC bewegt sich in dieselbe Richtung
3. Die Geschwindigkeit hängt von der Empfindlichkeits-Einstellung ab
4. Lass deinen Finger los und der Cursor bleibt, wo er ist (schnappt nicht zurück)

### Empfindlichkeit

- **Niedrige Werte (1-5)** — Cursor bewegt sich langsam, gut für präzises Zielen
- **Mittlere Werte (5-10)** — ausgewogen für allgemeine Nutzung
- **Hohe Werte (10-20)** — Cursor bewegt sich schnell, gut für schnelle Navigation

### Maus-Buttons

Der Mousepad-Block selbst verarbeitet nur die Bewegung. Für Mausklicks brauchst du Command Blocks oder Button-Blocks, die für Mauseingaben konfiguriert sind.

### Wann ein Mousepad nutzen

- Spiele, die Maus-Zielen erfordern (FPS-Spiele)
- Desktop-Steuerung von deinem Tablet aus
- Menü-Navigation in Spielen, die keine Controller unterstützen
- Jede Situation, in der du den echten Mauszeiger bewegen musst

---

## Touch Pad vs. Mousepad: Welches?

| Merkmal | Touch Pad | Mousepad |
|---------|-----------|----------|
| **Eingabetyp** | Joystick-Achsenwerte | Mausbewegung |
| **Verhalten** | Schnappt ins Zentrum zurück | Cursor bleibt, wo du ihn lässt |
| **Am besten für** | Analoge Steuerung (Schub, Kamera) | Zeigen und Klicken |
| **Spiel-Unterstützung** | Spiele mit Joystick-Unterstützung | Jedes Spiel, das eine Maus nutzt |
| **Präzision** | Sanfter analoger Bereich | Pixelgenaue Cursor-Steuerung |

### Schnelle Entscheidungshilfe

- Du musst **sanft steuern oder zielen**? Nutze ein **Touch Pad**
- Du musst **den Mauszeiger bewegen**? Nutze ein **Mousepad**
- Du spielst eine **Flugsimulation**? Nutze ein **Touch Pad** für die Kamerasteuerung
- Du spielst ein **FPS-Spiel**? Nutze ein **Mousepad** zum Zielen

---

## 3-Finger-Wischgeste: Zwischen Panels wechseln

### Was sie macht

Du kannst zwischen Panels wechseln, ohne zur Startseite zurückzugehen, indem du mit drei Fingern über deinen Touchscreen wischt.

### So benutzt du sie

1. Lege **drei Finger** gleichzeitig auf den Bildschirm
2. Wische nach **links** (von rechts nach links), um zum **nächsten Panel** zu wechseln
3. Wische nach **rechts** (von links nach rechts), um zum **vorherigen Panel** zu wechseln
4. Hebe deine Finger an — das Panel wechselt automatisch

### So funktioniert es

- **Panel-Reihenfolge**: Panels sind alphabetisch nach Name sortiert
- **Übergang**: Wenn du beim letzten Panel nach links wischt, kommst du zum ersten Panel, und umgekehrt
- **Visuelles Feedback**: Ein halbtransparentes Overlay erscheint und zeigt den Panel-Namen während du wischt
- **Minimale Wischdistanz**: Du musst mindestens 100 Pixel wischen (etwa 2-3 cm auf den meisten Tablets)
- **Abklingzeit**: Es gibt eine kurze Pause (eine halbe Sekunde) zwischen Wechseln, um versehentliches schnelles Wechseln zu verhindern

### Tipps

- Stelle sicher, dass du **genau drei Finger** benutzt — ein oder zwei Finger funktionieren nicht
- Wische **horizontal** — vertikale Wischgesten lösen keinen Panel-Wechsel aus
- Die Geste funktioniert überall auf dem Bildschirm **außer** auf Touch Pad-, Mousepad- oder Push-to-Talk-Blöcken
- Wenn das Panel nicht wechselt, versuche etwas schneller oder mit mehr Distanz zu wischen
- Pinch-to-Zoom ist auf dem Client deaktiviert, also musst du nicht befürchten, versehentlich zu zoomen

### Wann du die 3-Finger-Wischgeste nutzen solltest

- Du hast mehrere Panels für verschiedene Spiele oder Szenarien
- Du möchtest schnell während des Gameplays Panels wechseln
- Du möchtest dein Spiel nicht minimieren, um auf die Startseite zuzugreifen
- Du benutzt ein Touch-Gerät (Tablet, Touchscreen-Laptop oder Handy)

---

## Tipps für Touchpads und Mousepads

### Größenempfehlungen

- Mache Touch Pads und Mousepads **mindestens 3x3 Rastereinheiten** groß für komfortable Nutzung
- Größere Pads (4x4 oder 5x5) geben dir mehr Raum für präzise Steuerung
- Mache sie nicht zu groß — dir geht der Platz auf dem Panel aus

### Platzierungs-Tipps

- Platziere Touch Pads und Mousepads in Bereichen, wo dein Thumb oder Finger natürlich ruht
- Halte sie von den Panel-Rändern entfernt, damit du nicht versehentlich über den Rand ziehst
- Gruppiere zusammengehörige Steuerelemente in der Nähe (z. B. Feuer-Button neben einem Kamera-Touch Pad)

### Empfindlichkeit einstellen

Beginne mit mittleren Einstellungen und passe basierend auf dem Gefühl an:

1. Setze die Empfindlichkeit auf einen mittleren Wert
2. Teste in deinem Spiel
3. Wenn es zu langsam ist, erhöhe die Empfindlichkeit
4. Wenn es zu sprunghaft ist, verringere die Empfindlichkeit
5. Wiederhole, bis es sich richtig anfühlt

### Mehrere Touch Pads nutzen

Du kannst mehrere Touch Pads auf demselben Panel haben, die jeweils verschiedene Achsen steuern:

- Touch Pad 1: Kamerasteuerung (Axis ID 0)
- Touch Pad 2: Schub und Ruder (Axis ID 1)
- Touch Pad 3: Geschützturm-Zielen (Axis ID 2)

Stelle nur sicher, dass jedes Touch Pad eine andere Axis ID nutzt, um Konflikte zu vermeiden.

## Was kommt als Nächstes?

Du möchtest Programme starten oder Webanfragen von deinem Panel aus senden? Lerne mehr über [Command Blocks](08-command-blocks.md).
