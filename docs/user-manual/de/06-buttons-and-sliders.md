# Kapitel 6: Buttons und Slider

Buttons und Slider sind die häufigsten Steuerelemente auf jedem Panel. Dieses Kapitel deckt jeden Typ von Button- und Slider-Block ab, alle ihre Optionen und wie du sie nutzt.

## Button-Block

### Was er macht

Ein Button sendet einen einzelnen Joystick-Button-Druck, wenn du ihn antippst. Stell es dir wie eine Tastatur-Taste oder Gamepad-Taste vor — drücke sie, und das Spiel registriert einen Button-Druck. Lass sie los, und das Spiel registriert ein Loslassen.

### Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Label** | Text auf dem Button | „Gear", „Fire", „Boost" |
| **Icon-URL** | Pfad zu einem Icon-Bild (optional) | `helldivers2/Reinforce_Stratagem_Icon.svg` |
| **Layout** | Wie Icon und Text angezeigt werden | `image-text`, `image-only`, `text-only` |
| **Textposition** | Wo die Beschriftung relativ zum Icon erscheint | `bottom`, `top`, `left`, `right` |
| **Joystick Index** | Welcher virtuelle Controller (0-9) | 0 für ersten Controller |
| **Button ID** | Welcher Button auf diesem Controller (0-15) | 0 bis 15 |
| **Keyboard Index** | Welche virtuelle Tastatur (0-9) für Tastatur-Shortcuts | 0 für erste Tastatur |
| **Keyboard Key** | Taste oder Kombi, die beim Klick gesendet wird. Wenn gesetzt, sendet der Klick Tastatur-Ereignisse statt Joystick-Button-Ereignisse. | „a", „ctrl+a", „ctrl+shift+escape" |
| **Button-Farbe** | Button-Füllfarbe | Beliebige Farbe |
| **Button-Farbe Aktiv** | Farbe bei Button-Druck | Beliebige Farbe |
| **Beschriftungsfarbe** | Label-Textfarbe | Beliebige Farbe |
| **Rahmenfarbe** | Button-Umrissfarbe | Beliebige Farbe |
| **Schriftgröße** | Größe des Label-Textes | 1 bis 48 |
| **Border Radius** | Wie stark abgerundet die Ecken sind | 0 (quadratisch) bis 100 (Kreis) |

### So nutzt du ihn

1. Ziehe einen Button-Block auf dein Panel
2. Klicke auf das Zahnrad-Symbol
3. Setze das Label (z. B. „Landing Gear")
4. Setze den Joystick Index (meist 0)
5. Setze die Button ID (wähle eine ungenutzte Nummer, 0-15)
6. Passe die Farben an dein Panel-Theme an
7. Speichere das Panel

### In deinem Spiel

In den Steuerelement-Einstellungen deines Spiels ordne den entsprechenden Joystick-Button der gewünschten Aktion zu. Wenn dein Button Joystick 0, Button 3 nutzt, ordne „Joystick 0 Button 3" der Aktion „Landing Gear umschalten" im Spiel zu.

---

## Slider-Block (Horizontal)

### Was er macht

Ein horizontaler Slider steuert einen Joystick-Achsenwert von 0 bis 255 (oder einem eigenen Maximalwert). Ziehe den Slider-Knopf nach links oder rechts, um den Wert zu ändern. Spiele sehen das als analoge Eingabe — wie ein Gashebel oder Lenkrad.

Slider können auch direkt die Systemeinstellungen deines Computers steuern, wie Lautstärke oder Monitor-Helligkeit, indem sie Befehle auf deinem Computer ausführen, wenn du den Slider bewegst.

### Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Label** | Text über dem Slider | „Throttle", „Lautstärke", „Helligkeit" |
| **Joystick Index** | Welcher virtueller Controller (0-9) | 0 für ersten Controller |
| **Slider ID** | Welche Achse auf diesem Controller (0-7) | 0 bis 7 |
| **Knopffarbe** | Farbe des ziehbaren Knopfs | Beliebige Farbe |
| **Hintergrundfarbe** | Farbe der Slider-Spur | Beliebige Farbe |
| **Fortschrittsfarbe** | Farbe des gefüllten Bereichs | Beliebige Farbe |
| **Schriftgröße** | Größe des Label-Textes | 12 bis 48 |
| **Max Value** | Maximalwert, den der Slider erreichen kann | 255 (Standard), 100 für Prozentwerte |
| **System Control** | Name für den Systemsteuerungs-Modus (optional) | „volume", „brightness" |
| **System Control Command** | Befehl, der auf deinem Computer ausgeführt wird, wenn der Slider bewegt wird (optional) | Siehe Beispiele unten |

### So nutzt du ihn (Joystick-Modus)

1. Ziehe einen Slider-Block auf dein Panel
2. Klicke auf das Zahnrad-Symbol
3. Setze das Label (z. B. „Throttle")
4. Setze den Joystick Index (meist 0)
5. Setze die Slider ID (wähle eine ungenutzte Nummer, 0-7)
6. Passe die Farben an
7. Speichere das Panel

### So nutzt du ihn (Systemsteuerungs-Modus)

Um einen Slider zur Steuerung von Lautstärke, Helligkeit oder anderen Systemeinstellungen zu nutzen:

1. Ziehe einen Slider-Block auf dein Panel
2. Klicke auf das Zahnrad-Symbol
3. Setze das Label (z. B. „Lautstärke")
4. Setze **System Control** auf einen Namen wie „volume"
5. Setze **System Control Command** auf einen Befehl mit `{value}` an der Stelle, wo die Slider-Zahl hin soll
6. Setze **Max Value** passend zum erwarteten Bereich deines Befehls (z. B. 100 für Prozentwerte)
7. Speichere das Panel

### Beispiele für Systemsteuerungs-Befehle

| Was du steuern möchtest | Befehl | Max Value |
|--------------------------|---------|-----------|
| Systemlautstärke (PulseAudio/PipeWire) | `pactl set-sink-volume @DEFAULT_SINK@ {value}%` | 100 |
| Systemlautstärke (ALSA) | `amixer set Master {value}%` | 100 |
| Monitor-Helligkeit (ddcutil, externe Monitore) | `ddcutil -d 1 setvcp 10 {value}` | 100 |
| Laptop-Bildschirmhelligkeit | `brightnessctl set {value}%` | 100 |

> **Hinweis:** Der `{value}`-Platzhalter wird durch die aktuelle Slider-Position ersetzt. Wenn dein Slider von 0 bis 100 geht und du ihn auf 75 stellst, wird der Befehl mit `75` anstelle von `{value}` ausgeführt.

### Wertebereich

- **Linke Position** = 0
- **Rechte Position** = Max Value (Standard: 255)
- **Mittlere Position** = Max Value ÷ 2

### In deinem Spiel

Ordne die entsprechende Joystick-Achse in deinem Spiel zu. Zum Beispiel könnte „Joystick 0 Achse 0" der „Schubsteuerung" zugeordnet werden.

---

## Slider Vertical-Block

### Was er macht

Genau dasselbe wie der horizontale Slider, aber vertikal ausgerichtet. Ziehe den Slider-Knopf hoch oder runter, um den Wert zu ändern.

### Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Label** | Text über dem Slider | „Throttle", „Höhenruder-Trimmung" |
| **Joystick Index** | Welcher virtueller Controller (0-9) | 0 für ersten Controller |
| **Slider ID** | Welche Achse auf diesem Controller (0-7) | 0 bis 7 |
| **Knopffarbe** | Farbe des ziehbaren Knopfs | Beliebige Farbe |
| **Hintergrundfarbe** | Farbe der Slider-Spur | Beliebige Farbe |
| **Fortschrittsfarbe** | Farbe des gefüllten Bereichs | Beliebige Farbe |
| **Schriftgröße** | Größe des Label-Textes | 12 bis 48 |

### Wertebereich

- **Untere Position** = 0
- **Obere Position** = 255
- **Mittlere Position** = 127

### Wann vertikal vs. horizontal nutzen

- **Vertikale Slider** — super für Schubregler (nach vorne für mehr Leistung, nach hinten für weniger)
- **Horizontale Slider** — super für Balance-Steuerungen (Links-Rechts-Anpassungen wie Bremsbalance)

---

## Star Citizen Themen-Button

### Was er macht

Dasselbe wie der Standard-Button, aber mit einer Sci-Fi-Ästhetik — abgeschrägte Ecken und ein futuristischerer Look.

### Zusätzliche Einstellungen

| Einstellung | Was sie macht |
|---------|-------------|
| **Rahmenbreite** | Dicke des Button-Rahmens |
| **Aktive Farbe** | Farbe, wenn der Button gedrückt ist |

---

## Toggle-Block

### Was er macht

Ein Kippschalter, der in der Ein- oder Aus-Position bleibt. Anders als ein normaler Button (der nur aktiv ist, während du ihn drückst), merkt sich ein Toggle seinen Zustand. Die Spur behält immer ein 2:1-Seitenverhältnis und der Knopf bleibt perfekt kreisförmig, egal wie du den Block skalierst.

### Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Label** | Text auf dem Toggle | „Autopilot", „Lichter" |
| **Joystick Index** | Welcher virtueller Controller (0-9) | 0 |
| **Button ID** | Welcher Button auf diesem Controller (0-15) | 0 bis 15 |
| **Toggle-Modus** | Wie das Button-Signal funktioniert | „momentary" (Standard) oder „toggle" |
| **Toggle-Farbe** | Spurfarbe, wenn Toggle AUS ist | Beliebige Farbe |
| **Toggle-Farbe Aktiv** | Spurfarbe, wenn Toggle AN ist | Grün, Blau, etc. |
| **Label-Farbe** | Label-Textfarbe | Beliebige Farbe |
| **Rahmenfarbe** | Spur-Umrissfarbe | Beliebige Farbe |
| **Schriftgröße** | Größe des Label-Textes | 12 bis 48 |

### Toggle-Modus erklärt

Beide Modi behalten den visuellen Toggle-Zustand bei — der Knopf bleibt dort, wo du ihn hingesetzt hast. Der Unterschied liegt im Signal, das an das Spiel gesendet wird:

- **Momentary** (Standard): Jeder Klick sendet einen kurzen Button-Druck gefolgt von einem Loslassen (100ms Abstand). Der visuelle Zustand wechselt und bleibt. Nutze dies, wenn das Spiel einen kurzen Button-Tipp erwartet, um etwas umzuschalten.
- **Toggle**: Sendet Button-Zustand 1 (gedrückt), wenn du ihn EINSchaltest, und Zustand 0 (loslassen), wenn du ihn AUSSchaltest. Das Signal entspricht der visuellen Position. Nutze dies, wenn das Spiel den Button-Zustand direkt ausliest.

### So funktioniert es

- Einmal tippen, um EINzuschalten
- Nochmal tippen, um AUSzuschalten
- Der visuelle Zustand zeigt, ob er gerade an oder aus ist
- Die Spur behält immer eine 2:1-Pillenform; der Knopf bleibt kreisförmig

### In deinem Spiel

In den Steuerelement-Einstellungen deines Spiels ordne den entsprechenden Joystick-Button der gewünschten Aktion zu. Wenn dein Toggle Joystick 0, Button 5 nutzt, ordne „Joystick 0 Button 5" der Aktion „Autopilot umschalten" im Spiel zu.

---

## Star Citizen Power Slider

### Was er macht

Ein vertikaler Slider mit einem Fortschritts-Füllbalken, der den aktuellen Wert visuell anzeigt. Super für Energielevel, Schildstärke oder Energiemanagement-Anzeigen.

### Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Label** | Text über dem Slider | „Shields", „Engines", „Weapons" |
| **Joystick Index** | Welcher virtueller Controller (0-9) | 0 |
| **Slider ID** | Welche Achse auf diesem Controller (0-7) | 0 bis 7 |
| **Spurfarbe** | Farbe der leeren Spur | Dunkelgrau |
| **Fortschrittsfarbe** | Farbe des gefüllten Bereichs | Blau, Grün, etc. |

---

## Sequenz-Button-Block

### Was er macht

Ein Sequenz-Button sendet eine Reihe von Tastaturanschlägen, während die Strg-Taste gedrückt gehalten wird. Wenn du den Button antippst, hält OmniPanel-go Strg gedrückt, tippt jede Taste in deiner Sequenz nacheinander und lässt Strg dann los. Dies ist für Spiele gedacht, die **Strg+WASD**-Richtungseingaben anstelle von Pfeiltasten verwenden.

### Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|-------------|--------------|----------------|
| **Beschriftung** | Text auf dem Button | "Vorwärts", "Links", "Rechts strafen" |
| **Icon-URL** | Pfad zu einem Icon-Bild (optional) | `helldivers2/Reinforce_Stratagem_Icon.svg` |
| **Layout** | Wie Icon und Text angezeigt werden | `image-text`, `image-only`, `text-only` |
| **Textposition** | Wo die Beschriftung relativ zum Icon erscheint | `bottom`, `top`, `left`, `right` |
| **Sequenzcode** | Komma-getrennte Tasten (w, a, s, d verwenden) | `d,d,w,s,a` für rechts-rechts-oben-unten-links |
| **Tastenverzögerung** | Pause zwischen Tastenanschlägen in Millisekunden | 50 bis 1000 (Standard: 100) |
| **Tastatur-Index** | Welche virtuelle Tastatur (0-9) | 0 für erste Tastatur |
| **Schriftgröße** | Größe des Beschriftungstextes | 1 bis 48 |
| **Button-Farbe** | Button-Füllfarbe | Beliebige Farbe |
| **Button-Farbe Aktiv** | Farbe bei Ausführung | Beliebige Farbe |
| **Beschriftungsfarbe** | Textfarbe der Beschriftung | Beliebige Farbe |
| **Randfarbe** | Button-Umrissfarbe | Beliebige Farbe |

### So funktionieren Sequenzcodes

Der Sequenzcode verwendet einfache Buchstaben für WASD-Tasten:

| Buchstabe | Richtung |
|-----------|----------|
| `w` | Oben / Vorwärts |
| `a` | Links |
| `s` | Unten / Zurück |
| `d` | Rechts |

**Beispiel:** `d,d,w,s,a` bedeutet: Strg gedrückt halten, D tippen, D tippen, W tippen, S tippen, A tippen, Strg loslassen.

### So verwendest du ihn

1. Ziehe einen Sequenz-Button-Block auf dein Panel
2. Klicke auf das Zahnrad-Symbol
3. Setze die Beschriftung (z.B. "Verstärkung rufen")
4. Setze den Sequenzcode (z.B. `d,d,w,s,a`)
5. Optional: Setze eine Icon-URL für eine visuelle Anzeige
6. Passe die Tastenverzögerung an, wenn dein Spiel langsamere oder schnellere Eingaben braucht (Standard 100ms funktioniert für die meisten Spiele)
7. Speichere das Panel

### Wann du ihn verwendest

- Spiele, die Strg+WASD für Richtungsbefehle verwenden (z.B. Helldivers 2 Stratagem-Eingaben)
- Jede Situation, in der du eine bestimmte Tastensequenz mit einem Tippen senden musst
- Schnellzugriff-Panels für komplexe Tastatur-Makros

### Tipps

- **Tastenverzögerung:** Wenn das Spiel Eingaben verpasst, erhöhe die Verzögerung (versuche 150-200ms). Wenn es sich zu langsam anfühlt, verringere sie (versuche 50-80ms).
- **Sequenzlänge:** Es gibt kein festes Limit, aber sehr lange Sequenzen (10+ Tasten) können sich träge anfühlen.
- **Icon + Text:** Verwende das `image-text`-Layout mit einem Icon, um Sequenz-Buttons visuell von normalen Buttons zu unterscheiden.

---

## Star Citizen Emergency Button

### Was er macht

Ein dramatischer, rot pulsierender Button für Notfall-Aktionen. Er hat eine leuchtende Animation, um Aufmerksamkeit zu erregen.

### Einstellungen

| Einstellung | Was sie macht |
|---------|-------------|
| **Label** | Text auf dem Button | „EJECT", „ABORT", „EMERGENCY" |
| **Joystick Index** | Welcher virtueller Controller (0-9) |
| **Button ID** | Welcher Button auf diesem Controller (0-15) |
| **Rote Farbvarianten** | Verschiedene Rottöne für den Pulseffekt |

### Wann nutzen

Für Aktionen, die du selten nutzt, aber im Notfall schnell erreichen musst:
- Schleudersitz
- Notbremse
- Selbstzerstörung (falls dein Spiel das hat)
- Panik-Button

---

## Tipps für Buttons und Slider

### Tastatur-Shortcuts

Jeder Button oder Slider kann durch eine Tastatur-Taste oder -Kombination ausgelöst werden. Setze die **Keyboard Key**-Einstellung im Zahnrad-Menü des Blocks:

- **Einzelne Taste:** „a", „b", „f1", „space", „enter"
- **Mit Modifikatoren:** „ctrl+a", „ctrl+shift+a", „alt+f4"
- **Unterstützte Modifikatoren:** `ctrl`, `shift`, `alt`, `meta` (Windows/Command-Taste)

Wenn ein Button eine Keyboard Key-Einstellung hat, sendet ein Klick darauf das Tastatur-Ereignis an den Host-System statt eines Joystick-Button-Drucks. Das ermöglicht dir, Bildschirmtastaturen oder Schnellzugriff-Panels zu bauen, die in jede Anwendung hineintippen.

**Physische Tastatur-Auslösung:** Drücken der zugewiesenen Tasten-Kombination auf deiner physischen Tastatur löst ebenfalls den Button/Slider aus — der Panel-Button wird visuell aktiviert und das Tastatur-Ereignis wird an den Host gesendet.

### Farbkodierung

Nutze Farben, um zusammengehörige Steuerelemente zu gruppieren:
- **Grün** — Navigation und Bewegung
- **Rot** — Waffen und Kampf
- **Blau** — Systeme und Hilfsmittel
- **Gelb** — Warnungen und Alarme

### Größe ist wichtig

- Mache häufig genutzte Buttons **größer**, damit sie leichter anzutippen sind
- Mache selten genutzte Buttons **kleiner**, um Platz zu sparen
- Mache Slider **breiter/höher** für präzisere Steuerung

### Best Practices für Labels

- Halte Labels **kurz** — maximal 1-2 Wörter
- Nutze **Abkürzungen**, wenn nötig (z. B. „NAV" statt „Navigation")
- Nutze **Icons oder Symbole** für universelle Aktionen (Pfeile für Richtung, etc.)

### Eingabekonflikte vermeiden

Jeder Button und Slider braucht eine eindeutige Joystick-Eingabe. Der Editor zeigt, welche Eingaben bereits verwendet werden. Bevor du ein neues Steuerelement hinzufügst:

1. Prüfe die Eingabe-Zuordnungsansicht
2. Wähle einen Joystick Index und eine Button/Slider ID, die noch nicht verwendet wird
3. Wenn alle Eingaben auf einem Joystick belegt sind, erhöhe den Joystick Index, um einen anderen virtuellen Controller zu nutzen

## Was kommt als Nächstes?

Bereit für fortgeschrittenere Steuerelemente? Lerne mehr über [Touchpads und Mousepads](07-touch-pads-and-mousepads.md) für analogen Joystick- und Mauseingaben.
