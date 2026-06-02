# Kapitel 12: Virtuelle Joysticks

Dieses Kapitel erklärt, wie OmniPanel-go virtuelle Gamecontroller erstellt, die deine Spiele erkennen und nutzen können. Das zu verstehen hilft dir, deine Panels korrekt zu konfigurieren und Eingabeprobleme zu beheben.

## Was sind virtuelle Joysticks?

Wenn du einen Button antippst oder einen Slider auf deinem Panel bewegst, sendet OmniPanel-go diese Eingabe nicht direkt an dein Spiel. Stattdessen erstellt es einen **virtuellen Joystick** — einen simulierten Gamecontroller, den dein PC als echtes Gerät sieht. Dein Spiel empfängt dann Eingaben von diesem virtuellen Joystick, genau wie von einem physischen Gamepad.

## Wie viele Eingaben pro Joystick?

Jeder virtuelle Joystick bietet:

- **8 analoge Achsen** — für Slider, Touchpads und kontinuierliche Steuerungen
- **16 Buttons** — für Buttons, Schalter und Ein/Aus-Steuerungen

### Die 8 Achsen

| Achse | Name | Typische Verwendung |
|------|------|-------------|
| 0 | X | Horizontale Bewegung (links-rechts) |
| 1 | Y | Vertikale Bewegung (hoch-runter) |
| 2 | Z | Tiefe oder dritte Dimension |
| 3 | RX | Rotation um die X-Achse (Roll) |
| 4 | RY | Rotation um die Y-Achse (Yaw) |
| 5 | RZ | Rotation um die Z-Achse (Pitch) |
| 6 | Throttle | Schubsteuerung |
| 7 | Rudder | Rudersteuerung |

Alle Achsen haben einen Wertebereich von **0 bis 255**:
- **0** = Minimum (ganz links, ganz oben, etc.)
- **127** = Zentrum/Neutral
- **255** = Maximum (ganz rechts, ganz unten, etc.)

### Die 16 Buttons

Buttons sind von 0 bis 15 nummeriert. Jeder Button kann sein:
- **Gedrückt** (aktiv)
- **Losgelassen** (inaktiv)

---

## Plattform-Unterschiede

### Linux

Unter Linux nutzt OmniPanel-go das eingebaute `uinput`-Kernelmodul. Das ist Teil des Linux-Kernels, also:

- **Keine zusätzliche Software nötig** — es funktioniert von Haus aus
- **Benötigt Root/Admin-Berechtigungen** — OmniPanel-go braucht Zugriff auf `/dev/uinput`

Wenn du Berechtigungsfehler bekommst, starte OmniPanel-go mit `sudo`:
```bash
sudo ./omnipanel-go
```

Oder füge deinen Benutzer zur `input`-Gruppe hinzu:
```bash
sudo usermod -aG input $USER
```
(Du musst dich ab- und wieder anmelden, damit das wirksam wird.)

### Windows

Unter Windows nutzt OmniPanel-go den **vJoy-Treiber**, um virtuelle Joysticks zu erstellen.

**Installation:**
1. Lade vJoy Version **2.2.2.0** von der [offiziellen vJoy Releases-Seite](https://github.com/BrunnerInnovation/vJoy/releases/tag/v2.2.2.0) herunter
2. Führe den Installer aus
3. Stelle während der Installation sicher, dass du die Anzahl der Geräte aktivierst, die du möchtest (mindestens 4 wird empfohlen)
4. Starte deinen Computer neu, wenn dazu aufgefordert

**Die vJoyInterface.dll bekommen:**
Nach der Installation von vJoy findet OmniPanel-go automatisch die `vJoyInterface.dll`-Datei, die es braucht. Es sucht an diesen Orten:
- Neben der OmniPanel-go-Anwendungsdatei (wenn mitgeliefert)
- Standardinstallationsverzeichnisse von vJoy: `C:\Program Files\vJoy\` oder `C:\Program Files (x86)\vJoy\`
- Dein System-PATH

Wenn du OmniPanel-go von der Quelle mit dem Script `scripts/build-with-vosk.ps1` gebaut hast, wird die DLL automatisch neben deiner ausführbaren Datei kopiert. Andernfalls findet OmniPanel-go die DLL beim ersten Start im vJoy-Installationsverzeichnis.

OmniPanel-go lädt diese DLL einmal und behält sie, solange die App läuft. Dadurch wird eine wiederholte vJoy-Neuinitialisierung beim Start vermieden und die Stabilität auf manchen Windows-Systemen verbessert.

**Hinweis:** vJoy wird nur für virtuelle Joysticks benötigt. Virtuelle Maus- und Tastatureingaben nutzen die eingebaute `SendInput`-Funktion von Windows und brauchen keine zusätzlichen Treiber.

### macOS

macOS wird **nicht unterstützt** für virtuelle Joystick- oder Mauseingaben. Du kannst den OmniPanel-go-Webserver trotzdem auf einem Mac laufen lassen und Panels entwerfen, aber die Eingabesimulation funktioniert nicht.

---

## Die Anzahl der Joysticks konfigurieren

### In config.json

Öffne `config.json` und setze `numJoysticks`:

```json
{
  "numJoysticks": 5
}
```

Das erstellt 5 virtuelle Joysticks (Indizes 0, 1, 2, 3, 4).

### Standardwert

Der Standard ist **5 Joysticks**, was dir gibt:
- 32 Buttons pro Joystick × 5 = 80 Buttons insgesamt
- 8 Achsen pro Joystick × 5 = 40 Achsen insgesamt

Das reicht für die meisten Panels. Wenn du mehr Eingaben brauchst, erhöhe die Anzahl.

### Zur Laufzeit ändern

Du kannst die Joystick-Anzahl auch von der Startseite aus ändern, ohne OmniPanel-go neu zu starten. Suche nach der Joystick-Anzahl-Steuerung und passe sie dort an.

---

## Virtuelle Mauseingabe

Zusätzlich zu virtuellen Joysticks kann OmniPanel-go Mauseingaben simulieren.

### Fähigkeiten

- **Mausbewegung** — den Cursor relativ zu seiner aktuellen Position bewegen
- **Linksklick** — einen linken Maustasten-Druck simulieren
- **Rechtsklick** — einen rechten Maustasten-Druck simulieren
- **Mittelklick** — einen mittleren Maustasten-Druck simulieren (Scrollrad-Klick)
- **Scrollrad** — Hoch- oder Runterscrollen simulieren

### Plattform-Unterstützung

- **Linux:** Nutzt `uinput` (dasselbe wie virtuelle Joysticks)
- **Windows:** Nutzt eingebautes `SendInput` (kein zusätzlicher Treiber nötig)
- **macOS:** Nicht unterstützt

### Mauseingabe nutzen

Mauseingabe wird hauptsächlich über **Mousepad-Blöcke** auf deinem Panel genutzt. Konfiguriere die Empfindlichkeit des Mousepads, um zu steuern, wie schnell sich der Cursor bewegt.

---

## Virtuelle Tastatureingabe

OmniPanel-go kann auch Tastatureingaben simulieren, sodass du Tastendrücke und Kombinationen (wie Strg+A) an jede Anwendung senden kannst.

### Fähigkeiten

- **Einzelner Tastendruck** — jeder Buchstabe, Zahl, Funktionstaste oder Sondertaste
- **Tastenkombinationen** — Modifikator + Taste (z. B. Strg+A, Strg+Shift+Esc, Alt+F4)
- **Unterstützte Modifikatoren:** Strg, Shift, Alt, Meta (Windows/Command-Taste)

### Plattform-Unterstützung

- **Linux:** Nutzt `uinput` (dasselbe wie virtuelle Joysticks und Maus)
- **Windows:** Nutzt eingebautes `SendInput` (kein zusätzlicher Treiber nötig)
- **macOS:** Nicht unterstützt

### Tastatureingabe nutzen

Tastatureingabe wird über die **Keyboard Key**-Einstellung an jedem Button- oder Slider-Block konfiguriert:

1. Klicke auf das Zahnrad-Symbol eines Buttons oder Sliders
2. Setze **Keyboard Key** auf die gewünschte Taste oder Kombination (z. B. „a", „ctrl+a")
3. Setze **Keyboard Index**, um zu wählen, welche virtuelle Tastatur (meist 0)

Wenn du den Button auf deinem Panel klickst, wird der Tastendruck an das Host-System gesendet. Das funktioniert mit jeder Anwendung, die Tastatureingaben akzeptiert.

### Eine Bildschirmtastatur erstellen

Du kannst eine vollständige Bildschirmtastatur erstellen, indem du Button-Blöcke für jede Taste platzierst und ihre Keyboard Key-Werte setzt. Das mitgelieferte **Keyboard**-Panel-Template demonstriert dieses Layout.

### Verfügbare Tastennamen

Nutze diese Namen in der Keyboard Key-Einstellung:

| Kategorie | Beispiele |
|-----------|-----------|
| Buchstaben | `a` bis `z` |
| Zahlen | `0` bis `9` |
| Funktionstasten | `f1` bis `f12` |
| Modifikatoren | `ctrl`, `shift`, `alt`, `meta`, `win`, `super` |
| Sondertasten | `space`, `enter`, `return`, `escape`, `esc`, `tab`, `backspace`, `capslock` |
| Satzzeichen | `minus`, `equal`, `comma`, `dot`, `slash`, `semicolon`, `apostrophe`, `grave`, `backslash`, `leftbracket`, `rightbracket` |
| Navigation | `insert` / `ins`, `delete` / `del`, `home`, `end`, `pageup` / `pgup`, `pagedown` / `pgdn` |
| Pfeiltasten | `up`, `down`, `left`, `right` |
| Numpad | `numpad0`–`numpad9`, `numpaddot`, `numpadenter`, `numpadplus`, `numpadminus`, `numpadmultiply`, `numpaddivide` |
| Medien | `volumeup`, `volumedown`, `mute`, `playpause`, `stop`, `nextsong`, `previoussong` |
| System | `printscreen`, `scrolllock`, `numlock` |

Kombiniere Modifikatoren mit `+`: `ctrl+a`, `ctrl+shift+a`, `alt+f4`, `meta+e`.

---

## Panel-Steuerungen Joystick-Eingaben zuordnen

### Button-Blöcke

Jeder Button-Block braucht:
- **Joystick Index** — welcher virtueller Controller (0-9)
- **Button ID** — welcher Button auf diesem Controller (0-15)

**Beispiel:** Ein Button mit Joystick Index 0 und Button ID 3 steuert den 4. Button des 1. virtuellen Joysticks.

### Slider-Blöcke

Jeder Slider-Block braucht:
- **Joystick Index** — welcher virtueller Controller (0-9)
- **Slider ID** — welche Achse auf diesem Controller (0-7)

**Beispiel:** Ein Slider mit Joystick Index 0 und Slider ID 0 steuert die X-Achse (erste Achse) des 1. virtuellen Joysticks.

### Touch Pad-Blöcke

Jeder Touchpad-Block braucht:
- **Joystick Index** — welcher virtueller Controller (0-9)
- **Axis ID** — welches Achsenpaar genutzt werden soll (0-3)

Axis ID-Zuordnung:
- **Axis ID 0** — nutzt Achse 0 (X) und Achse 1 (Y)
- **Axis ID 1** — nutzt Achse 2 (Z) und Achse 3 (RX)
- **Axis ID 2** — nutzt Achse 4 (RY) und Achse 5 (RZ)
- **Axis ID 3** — nutzt Achse 6 (Throttle) und Achse 7 (Rudder)

---

## Spiele einrichten, um virtuelle Joysticks zu nutzen

### Schritt 1: OmniPanel-go starten

Stelle sicher, dass OmniPanel-go läuft, bevor du dein Spiel startest. Die virtuellen Joysticks werden erstellt, wenn OmniPanel-go startet.

### Schritt 2: Dein Spiel starten

Starte dein Spiel wie gewohnt.

### Schritt 3: Spiel-Steuerungen konfigurieren

In den Steuerelement-Einstellungen deines Spiels:

1. Gehe zur Eingabe-/Controller-Konfiguration
2. Suche nach Joystick- oder Gamepad-Einstellungen
3. Wähle den virtuellen Joystick (er könnte als „vJoy Device" unter Windows oder „OmniPanel-go Joystick" unter Linux erscheinen)
4. Ordne die Buttons und Achsen Spielaktionen zu

### Schritt 4: Eingaben testen

1. Öffne dein Panel auf deinem Tablet
2. Tippe auf einen Button oder bewege einen Slider
3. Prüfe, ob das Spiel reagiert
4. Wenn nicht, überprüfe, ob Joystick Index und Button/Achsen-IDs mit der Konfiguration deines Spiels übereinstimmen

---

## Fehlerbehebung für virtuelle Joysticks

### Spiel erkennt den virtuellen Joystick nicht

**Windows:**
- Stelle sicher, dass vJoy installiert und konfiguriert ist
- Öffne das vJoy-Konfigurationstool und überprüfe, ob Geräte aktiviert sind
- Stelle sicher, dass `vJoyInterface.dll` sich in einer dieser Speicherorte befindet:
  - Neben `omnipanel-go.exe`
  - In deinem vJoy-Installationsverzeichnis
  - In deinem System-PATH
- Starte OmniPanel-go nach der Installation von vJoy neu
- Prüfe, dass die DLL nicht von Antivirus-Software unter Quarantäne gestellt wurde

Wenn beim Start Popups von `vJoyInterface DLL` erscheinen (zum Beispiel `RegisterClassEx failed` oder `Creation of dummy window failed`), beende OmniPanel-go vollständig und starte es erneut, nachdem du die vJoy-Installation geprüft hast. Aktuelle OmniPanel-go-Versionen laden die DLL pro Lauf nur einmal, um dieses Problem zu reduzieren.

**Linux:**
- Prüfe, ob du Berechtigung für den Zugriff auf `/dev/uinput` hast
- Versuche, OmniPanel-go mit `sudo` zu starten
- Überprüfe, ob das `uinput`-Kernelmodul geladen ist: `lsmod | grep uinput`

### Eingaben funktionieren nicht

- Überprüfe, ob der **Joystick Index** in deinen Block-Einstellungen mit dem Joystick übereinstimmt, auf den dein Spiel hört
- Prüfe, ob die **Button ID** oder **Slider ID** mit der richtigen Eingabe übereinstimmt
- Nutze die **Eingabe-Zuordnungsansicht** im Editor, um zu sehen, welche Eingaben bereits verwendet werden
- Stelle sicher, dass nicht zwei Blöcke dieselbe Eingabe nutzen (es sei denn, du möchtest, dass sie dasselbe tun)

### Zu wenige Eingaben

Wenn dir Buttons oder Achsen ausgehen:

1. Erhöhe `numJoysticks` in `config.json`
2. Starte OmniPanel-go neu
3. Nutze höhere Joystick Index-Werte in deinen Blöcken (1, 2, 3, etc.)

---

## Tipps für virtuelle Joysticks

### Deine Eingaben organisieren

Führe eine Aufzeichnung, welche Eingaben du zugeordnet hast:

```
Joystick 0:
  Button 0: Landing Gear
  Button 1: Flaps Up
  Button 2: Flaps Down
  Achse 0: Throttle (Slider)
  Achse 1: Camera X/Y (Touch Pad)

Joystick 1:
  Button 0: Weapon Group 1
  Button 1: Weapon Group 2
  ...
```

### Konsistente Nummerierung

Nutze ein konsistentes Nummerierungsschema über deine Panels hinweg:

- Joystick 0: Primäre Flugsteuerungen
- Joystick 1: Waffen und Kampf
- Joystick 2: Systeme und Hilfsmittel
- Joystick 3: Navigation und Kommunikation

Das macht es einfacher, sich zu merken, welche Eingabe was macht.

### Testen ohne Spiel

Du kannst virtuelle Joystick-Eingaben testen, ohne ein Spiel zu starten:

1. Unter Windows nutze das Tool „USB-Gamecontroller einrichten" in der Systemsteuerung
2. Unter Linux nutze `jstest` oder `evtest`, um Joystick-Ereignisse zu überwachen
3. Tippe auf Buttons und bewege Slider auf deinem Panel — du solltest sehen, dass die Eingaben registriert werden

## Was kommt als Nächstes?

Du möchtest deine eigenen Block-Designs erstellen? Lerne mehr über [Eigene Blöcke](13-custom-blocks.md).
