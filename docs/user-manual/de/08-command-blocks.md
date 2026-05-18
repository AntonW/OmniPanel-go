# Kapitel 8: Command Blocks

Command Blocks lassen dich Programme auf deinem PC starten oder Webanfragen senden, indem du einen Button auf deinem Panel antippst. Dieses Kapitel deckt alles ab, was du mit Command Blocks machen kannst.

## Was ist ein Command Block?

Ein Command Block ist ein Button, der beim Drücken einen Befehl ausführt, statt eine Joystick-Eingabe zu senden. Du kannst ihn nutzen, um:

- Spiele oder Anwendungen zu starten
- Systemeinstellungen zu steuern (Lautstärke, Helligkeit)
- Daten an andere Programme zu senden
- Webdienste auszulösen
- Skripte auszuführen

## Zwei Modi

Command Blocks funktionieren in zwei Modi:

### Shell-Modus

Führt einen Befehl direkt auf deinem PC aus, genau als würdest du ihn in ein Terminal oder die Eingabeaufforderung tippen.

**Beispiele:**
- `notepad` — öffnet Notepad unter Windows
- `echo Hallo` — gibt „Hallo" in der Konsole aus
- `ls -la` — listet Dateien unter Linux
- `volume set 50` — setzt die Systemlautstärke (wenn du ein Lautstärke-Tool hast)

### HTTP-Modus

Sendet eine Webanfrage an einen Server oder eine API.

**Beispiele:**
- Eine GET-Anfrage senden, um einen Server-Status zu prüfen
- Eine POST-Anfrage senden, um ein Smart-Home-Gerät zu steuern
- Daten an eine Webanwendung senden

## Einen Command Block erstellen

1. Ziehe einen **Command Block** aus der Bibliothek auf deine Arbeitsfläche
2. Klicke auf das Zahnrad-Symbol, um die Einstellungen zu öffnen
3. Wähle den Befehlstyp: **Shell** oder **HTTP**

---

## Shell-Modus Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Label** | Text auf dem Button | „Spiel starten", „Screenshot" |
| **Icon-URL** | Pfad zu einem Icon-Bild (optional) | `helldivers2/Reinforce_Stratagem_Icon.svg` |
| **Layout** | Wie Icon und Text angezeigt werden | `image-text`, `image-only`, `text-only` |
| **Textposition** | Wo die Beschriftung relativ zum Icon erscheint | `bottom`, `top`, `left`, `right` |
| **Command Type** | Auf „Shell" setzen | Shell |
| **Command** | Der auszuführende Befehl | `notepad`, `echo Hallo`, `calc` |
| **Button-Farbe** | Button-Füllfarbe | Beliebige Farbe |
| **Button-Farbe Aktiv** | Farbe bei Button-Druck | Beliebige Farbe |
| **Beschriftungsfarbe** | Label-Textfarbe | Beliebige Farbe |
| **Rahmenfarbe** | Button-Umrissfarbe | Beliebige Farbe |
| **Hold Repeat** | Halten-zum-Wiederholen aktivieren (siehe unten) | An oder Aus |
| **Wiederholintervall** | Wie oft wiederholt wird (in Millisekunden) | 500, 1000, 2000 |

### Befehlsbeispiele

**Windows:**
```
notepad          - Öffnet Notepad
calc             - Öffnet den Rechner
mspaint          - Öffnet Paint
explorer .       - Öffnet den Datei-Explorer im aktuellen Ordner
```

**Linux:**
```
gedit            - Öffnet den Gedit-Texteditor
gnome-calculator - Öffnet den Rechner
xdg-open .       - Öffnet den Dateimanager im aktuellen Ordner
```

---

## HTTP-Modus Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Label** | Text auf dem Button | „Licht umschalten", „Daten senden" |
| **Icon-URL** | Pfad zu einem Icon-Bild (optional) | `helldivers2/Reinforce_Stratagem_Icon.svg` |
| **Layout** | Wie Icon und Text angezeigt werden | `image-text`, `image-only`, `text-only` |
| **Textposition** | Wo die Beschriftung relativ zum Icon erscheint | `bottom`, `top`, `left`, `right` |
| **Command Type** | Auf „HTTP" setzen | HTTP |
| **Method** | HTTP-Anfragetyp | GET, POST, PUT, DELETE, PATCH |
| **URL** | Die Webadresse, an die die Anfrage gesendet wird | `http://192.168.1.50/api/light` |
| **Body** | Daten, die mit der Anfrage gesendet werden (für POST/PUT) | `{"on": true}` |
| **Button-Farbe** | Button-Füllfarbe | Beliebige Farbe |
| **Button-Farbe Aktiv** | Farbe bei Button-Druck | Beliebige Farbe |
| **Beschriftungsfarbe** | Label-Textfarbe | Beliebige Farbe |
| **Rahmenfarbe** | Button-Umrissfarbe | Beliebige Farbe |
| **Hold Repeat** | Halten-zum-Wiederholen aktivieren | An oder Aus |
| **Wiederholintervall** | Wie oft wiederholt wird (in Millisekunden) | 500, 1000, 2000 |

### HTTP-Methodentypen

| Methode | Wann nutzen | Beispiel |
|--------|-------------|---------|
| **GET** | Daten von einem Server anfordern | Server-Status abrufen |
| **POST** | Neue Daten an einen Server senden | Einen neuen Eintrag erstellen |
| **PUT** | Bestehende Daten aktualisieren | Eine Einstellung ändern |
| **DELETE** | Daten löschen | Eine Datei löschen |
| **PATCH** | Daten teilweise aktualisieren | Ein Feld ändern |

### HTTP Body

Für POST-, PUT- und PATCH-Anfragen kannst du einen Body mit der Anfrage mitsenden. Das sind typischerweise JSON-Daten:

```json
{"action": "turn_on", "brightness": 75}
```

---

## Parameter: Befehle anpassbar machen

Parameter lassen dich einstellbare Eingaben über deinem Command Block-Button hinzufügen. Statt Werte fest in deinen Befehl zu kodieren, kannst du Parameter nutzen, um sie spontan zu ändern.

### Parameter hinzufügen

1. Finde in den Command Block-Einstellungen den Bereich **Parameter**
2. Klicke auf **Parameter hinzufügen**
3. Konfiguriere den Parameter:

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Name** | Der Parameter-Name (im Befehl verwendet) | `volume`, `brightness` |
| **Type** | Eingabetyp | Number, Text, Toggle |
| **Default Value** | Startwert | 50, „hallo", false |
| **Min/Max** | Für Number-Typ: erlaubter Bereich | 0 bis 100 |
| **Step** | Für Number-Typ: Schrittgröße | 1, 5, 10 |

### Parameter in Befehlen verwenden

In deinem Befehl nutze `{name}` dort, wo der Parameterwert erscheinen soll.

**Shell-Beispiel:**
```
echo Lautstärke ist gesetzt auf {volume}
```

Wenn der Volume-Parameter auf 75 gesetzt ist, wird der Befehl zu:
```
echo Lautstärke ist gesetzt auf 75
```

**HTTP-Beispiel:**
```json
{"brightness": {brightness}, "color": "{color}"}
```

Wenn brightness 80 und color „red" ist, wird der Body zu:
```json
{"brightness": 80, "color": "red"}
```

### Parameter-Typen

**Number:**
- Zeigt eine Zahleneingabe mit Hoch/Runter-Pfeilen
- Setze Min-, Max- und Schritt-Werte
- Gut für: Lautstärkepegel, Helligkeit, Anzahl

**Text:**
- Zeigt ein Texteingabefeld
- Gut für: Namen, Nachrichten, Dateipfade

**Toggle:**
- Zeigt einen Ein/Aus-Schalter
- Sendet `true` oder `false`
- Gut für: Aktivieren/Deaktivieren-Flags

---

## Halten-zum-Wiederholen

Halten-zum-Wiederholen lässt dich den Command Block-Button gedrückt halten, um den Befehl wiederholt in festgelegten Intervallen auszuführen.

### Wann Halten-zum-Wiederholen nutzen

- **Lautstärkeregler** — gedrückt halten, um die Lautstärke kontinuierlich zu erhöhen/verringern
- **Scrollen** — gedrückt halten, um weiter zu scrollen
- **Schnelle Befehle** — jeder Befehl, den du wiederholt auslösen möchtest

### Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Hold Repeat** | Wiederholung aktivieren oder deaktivieren | An oder Aus |
| **Wiederholintervall** | Zeit zwischen Wiederholungen (Millisekunden) | 100, 500, 1000 |

### Wiederholintervall-Leitfaden

- **100ms** — sehr schnelle Wiederholung (10 Mal pro Sekunde)
- **500ms** — mittlere Wiederholung (2 Mal pro Sekunde)
- **1000ms** — langsame Wiederholung (einmal pro Sekunde)
- **2000ms** — sehr langsame Wiederholung (einmal alle 2 Sekunden)

---

## Visuelles Feedback

Command Blocks geben visuelles Feedback, damit du weißt, was passiert:

### Während der Ausführung

Der Button zeigt eine **leuchtende Animation**, während der Befehl läuft.

### Bei Erfolg

Der Button-Rahmen blinkt kurz **grün** auf.

### Bei Fehler

Der Button-Rahmen blinkt kurz **rot** auf.

### Toast-Benachrichtigung

Eine Nachricht erscheint, die die Befehlsausgabe zeigt. Diese Nachricht:

- Zeigt, was der Befehl zurückgegeben hat
- Verschwindet automatisch nach 5 Sekunden
- Erscheint oben auf dem Panel

---

## Praktische Beispiele

### Beispiel 1: Lautstärkeregler

**Ziel:** Buttons erstellen, um die Systemlautstärke zu erhöhen und zu verringern.

**Lautstärke erhöhen Button:**
- Label: „Lauter"
- Command Type: Shell
- Command: `pactl set-sink-volume @DEFAULT_SINK@ +5%` (Linux) oder ein Windows-Lautstärke-Tool-Befehl
- Hold Repeat: An
- Wiederholintervall: 200

**Lautstärke verringern Button:**
- Label: „Leiser"
- Command Type: Shell
- Command: `pactl set-sink-volume @DEFAULT_SINK@ -5%` (Linux)
- Hold Repeat: An
- Wiederholintervall: 200

### Beispiel 2: Spiel starten

**Ziel:** Ein Button, um dein Lieblingspiel zu starten.

**Button-Einstellungen:**
- Label: „Star Citizen starten"
- Command Type: Shell
- Command: `C:\Program Files\Roberts Space Industries\StarCitizen\LIVE\StarCitizen.exe` (Windows)

### Beispiel 3: Smart-Home-Steuerung

**Ziel:** Ein Smart-Home-Licht von deinem Panel aus einschalten.

**Button-Einstellungen:**
- Label: „Wohnzimmer Licht"
- Command Type: HTTP
- Method: POST
- URL: `http://192.168.1.100/api/lights/1`
- Body: `{"on": true}`

### Beispiel 4: Screenshot-Button

**Ziel:** Einen Screenshot mit einem Tippen machen.

**Button-Einstellungen:**
- Label: „Screenshot"
- Command Type: Shell
- Command: `gnome-screenshot` (Linux) oder ein Windows-Screenshot-Tool

---

## Tipps für Command Blocks

### Sicherheitshinweis

Shell-Befehle laufen mit denselben Berechtigungen wie OmniPanel-go. Sei vorsichtig mit Befehlen, die:

- Dateien löschen
- Systemeinstellungen ändern
- Mit Administrator-/Root-Rechten laufen

### Befehle testen

Bevor du einen Befehl zu deinem Panel hinzufügst, teste ihn zuerst in einem Terminal oder der Eingabeaufforderung. So stellst du sicher, dass der Befehl funktioniert, und siehst, welche Ausgabe zu erwarten ist.

### Fehlerbehandlung

Wenn ein Befehl fehlschlägt:

1. Prüfe das rote Aufblinken und die Fehler-Toast-Benachrichtigung
2. Lies die Fehlermeldung
3. Teste den Befehl in einem Terminal, um die vollständige Fehlermeldung zu sehen
4. Korrigiere den Befehl und versuche es erneut

### Pfad-Probleme

Unter Windows, wenn dein Befehl Leerzeichen im Pfad enthält, setze ihn in Anführungszeichen:

```
"C:\Program Files\My App\app.exe"
```

## Was kommt als Nächstes?

Lerne, wie du komplexe Panels organisierst mit [Seiten, Tabs und Bereichen](09-organizing-panels.md).
