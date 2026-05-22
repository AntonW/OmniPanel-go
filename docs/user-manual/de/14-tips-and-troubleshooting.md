# Kapitel 14: Tipps und Fehlerbehebung

Dieses Kapitel deckt Best Practices für das Design großartiger Panels ab, häufige Probleme, die auftreten können, und wie du sie behebst.

## Best Practices für das Design

### Plane dein Layout zuerst

Bevor du den Editor öffnest, skizziere dein Panel auf Papier:

1. Welche Steuerelemente brauchst du?
2. Welche nutzt du am häufigsten?
3. Wie sollten sie gruppiert werden?
4. Welche Größe sollte jedes Steuerelement haben?

Eine schnelle Skizze spart Zeit und führt zu einem besseren Layout.

### Häufig genutzte Steuerelemente priorisieren

- Platziere **am häufigsten genutzte Steuerelemente** in leicht erreichbaren Bereichen (Mitte oder unten auf dem Bildschirm)
- Mache sie **größer** für einfacheres Antippen
- Nutze **auffällige Farben**, damit du sie schnell findest

### Konsistente Farben verwenden

Erstelle ein Farbschema und halte dich daran:

| Farbe | Empfohlene Verwendung |
|-------|--------------|
| Grün | Navigation, Bewegung, positive Aktionen |
| Rot | Waffen, Kampf, Notfall-Aktionen |
| Blau | Systeme, Hilfsmittel, Informationen |
| Gelb | Warnungen, Alarme, Vorsicht |
| Grau | Deaktivierte oder inaktive Steuerelemente |

### Platz zwischen Steuerelementen lassen

Packe Steuerelemente nicht zu eng zusammen. Lass mindestens eine Rastereinheit zwischen Blöcken, damit du während intensiven Gameplays nicht versehentlich den falschen antippst.

### Seiten klug nutzen

- Halte zusammengehörige Steuerelemente auf derselben Seite
- Erstelle nicht zu viele Seiten (3-5 sind ideal)
- Benenne Seiten klar („Flight", „Combat", nicht „Seite 1", „Seite 2")

### Auf deinem tatsächlichen Gerät testen

Entwirf auf deinem PC, aber teste immer auf dem Tablet oder Handy, das du tatsächlich beim Spielen nutzt:

- Prüfe, ob Buttons groß genug zum Antippen sind
- Stelle sicher, dass Farben in deiner Gaming-Umgebung lesbar sind
- Mach dich vergewissern, dass das Layout sich natürlich in deinen Händen anfühlt

---

## Häufige Probleme und Lösungen

### OmniPanel-go startet nicht

**Problem:** Du doppelklickst auf `omnipanel-go.exe` (oder startest `./omnipanel-go`) und nichts passiert.

**Lösungen:**
- **Windows:** Öffne eine Eingabeaufforderung, navigiere zum OmniPanel-go-Ordner und starte `omnipanel-go.exe` von dort aus. Du siehst Fehlermeldungen, wenn etwas nicht stimmt.
- **Linux:** Starte `./omnipanel-go` in einem Terminal, um Fehlermeldungen zu sehen.
- **Port bereits belegt:** Wenn Port 3000 bereits von einem anderen Programm genutzt wird, ändere den Port in `config.json`:
  ```json
  {
    "port": 3001
  }
  ```

### Keine Verbindung vom Tablet aus

**Problem:** Dein Tablet zeigt „Diese Website kann nicht erreicht werden."

**Lösungen:**
1. Überprüfe, ob beide Geräte im **selben WiFi-Netzwerk** sind
2. Prüfe, ob OmniPanel-go noch auf deinem PC läuft
3. Überprüfe, ob die IP-Adresse korrekt ist (sie könnte sich geändert haben)
4. Prüfe die Firewall deines PCs — OmniPanel-go könnte blockiert sein
5. Versuche zuerst, vom PC-Browser aus mit `http://localhost:3000` zuzugreifen

### Panel reagiert nicht auf Berührungen

**Problem:** Du tippst auf Buttons auf deinem Panel, aber im Spiel passiert nichts.

**Lösungen:**
1. Prüfe das Verbindungsprotokoll auf der Startseite — ist dein Gerät verbunden?
2. Überprüfe, ob **Joystick Index** und **Button/Slider ID** korrekt sind
3. Stelle sicher, dass dein Spiel so konfiguriert ist, dass es auf den richtigen virtuellen Joystick hört
4. Teste Eingaben mit dem Joystick-Testtool deines Betriebssystems (siehe Kapitel 12)

### Virtueller Joystick wird vom Spiel nicht erkannt

**Problem:** Dein Spiel sieht den virtuellen Joystick nicht.

**Lösungen:**
- **Windows:**
  - Stelle sicher, dass vJoy installiert ist (Version 2.2.2.0)
  - Öffne die vJoy-Konfiguration und überprüfe, ob Geräte aktiviert sind
  - Starte OmniPanel-go nach der Installation von vJoy neu
  - Starte dein Spiel neu, nachdem du OmniPanel-go gestartet hast

- **Linux:**
  - Prüfe Berechtigungen: `ls -l /dev/uinput`
  - Starte OmniPanel-go bei Bedarf mit `sudo`
  - Überprüfe, ob das `uinput`-Modul geladen ist: `lsmod | grep uinput`

### Spracherkennung funktioniert nicht

**Problem:** Du sprichst, aber nichts passiert.

**Lösungen:**
1. Prüfe, ob Sprache in `config.json` aktiviert ist (`"enabled": true`)
2. Überprüfe, ob dein Browser Mikrofon-Berechtigung hat
3. Prüfe die Browser-Konsole auf Fehler (drücke F12)
4. Wenn du Vosk nutzt, überprüfe, ob das Modell korrekt heruntergeladen wurde
5. Wenn du llama-cpp nutzt, überprüfe, ob der Server läuft und die URL korrekt ist
6. Versuche zuerst den Push-to-Talk-Modus, um das Problem einzugrenzen

### Befehle werden nicht ausgeführt

**Problem:** Du tippst auf einen Command Block, aber nichts passiert.

**Lösungen:**
1. Prüfe die Toast-Benachrichtigung auf Fehlermeldungen
2. Teste den Befehl zuerst in einem Terminal
3. Überprüfe, ob der Befehlspfad korrekt ist (achte auf Tippfehler)
4. Unter Windows setze Pfade mit Leerzeichen in Anführungszeichen: `"C:\Program Files\App\app.exe"`
5. Prüfe, ob OmniPanel-go die Berechtigung hat, den Befehl auszuführen

### Panel sieht auf dem Tablet anders aus als auf dem PC

**Problem:** Dein Panel-Layout sieht auf deinem PC gut aus, aber auf deinem Tablet ist es durcheinander.

**Lösungen:**
1. Prüfe die Seitenverhältnis-Einstellung im Editor — passe sie an dein Tablet an
2. Nutze prozentbasierte Größen (das Rastersystem handhabt das automatisch)
3. Teste während des Designs auf deinem tatsächlichen Tablet, nicht nur in der PC-Vorschau
4. Vermeide feste Pixelgrößen — nutze stattdessen Rastereinheiten

---

## FAQ

### Kann ich OmniPanel-go über das Internet nutzen?

Nein. OmniPanel-go ist nur für die Nutzung im lokalen Netzwerk gedacht. Sowohl dein PC als auch dein Tablet müssen im selben WiFi-Netzwerk sein. Das ist absichtlich so für Privatsphäre und Sicherheit.

### Können mehrere Personen OmniPanel-go gleichzeitig nutzen?

Ja. Mehrere Geräte können sich gleichzeitig mit OmniPanel-go verbinden, und jedes kann verschiedene Panels öffnen.

### Muss ich etwas auf meinem Tablet installieren?

Nein. OmniPanel-go läuft komplett im Webbrowser deines Tablets. Keine Apps zum Installieren.

### Funktioniert OmniPanel-go mit jedem Spiel?

OmniPanel-go funktioniert mit Spielen, die Joystick- oder Mauseingaben unterstützen. Die meisten PC-Spiele unterstützen diese Eingabemethoden. Spiele, die nur Tastatureingaben unterstützen, funktionieren nicht direkt (obwohl du Command Blocks nutzen könntest, um Tastatursimulations-Tools zu starten).

### Wie sichere ich meine Panels?

Deine Panels werden als JSON-Dateien im `user/panels/`-Ordner gespeichert. Kopiere diesen Ordner, um alle deine Panels zu sichern.

### Wie teile ich meine Panels mit jemand anderem?

Kopiere die Panel-JSON-Datei aus `user/panels/` und lege sie im selben Ordner einer anderen OmniPanel-go-Installation ab.

### Kann ich OmniPanel-go auf einem Mac nutzen?

Du kannst den OmniPanel-go-Server auf einem Mac laufen lassen und Panels entwerfen, aber virtuelle Joystick- und Mauseingaben funktionieren nicht. macOS wird für Spieleingabe-Simulation nicht unterstützt.

### Funktioniert OmniPanel-go mit Controllern wie Xbox- oder PlayStation-Pads?

OmniPanel-go erstellt virtuelle Joysticks, die Spiele als generische Controller sehen. Es interagiert nicht mit physischen Controllern. Du kannst einen physischen Controller und OmniPanel-go-Panels gleichzeitig nutzen — sie erscheinen dem Spiel als separate Geräte.

### Wie aktualisiere ich OmniPanel-go?

Lade die neueste Version herunter und ersetze deine bestehende `omnipanel-go`-Binärdatei. Dein `user/`-Ordner (Panels, Blöcke, Konfiguration) wird beibehalten, sodass du keine deiner Arbeiten verlierst.

### Kann ich die Port-Nummer ändern?

Ja. Bearbeite `config.json`:
```json
{
  "port": 8080
}
```
Oder nutze die Umgebungsvariable `OMNIPANEL_PORT=8080`.

### Wie erhöhe ich die Anzahl der virtuellen Joysticks?

Bearbeite `config.json`:
```json
{
  "numJoysticks": 8
}
```
Oder nutze die Umgebungsvariable `OMNIPANEL_NUMJOYSTICKS=8`.

---

## Hilfe bekommen

Wenn du feststeckst:

1. **Prüfe dieses Handbuch** — dein Problem könnte in einem früheren Kapitel behandelt werden
2. **Prüfe das Verbindungsprotokoll** — es zeigt Verbindungs- und Trennungsereignisse
3. **Teste isoliert** — versuche eine Funktion nach der anderen, um das Problem zu identifizieren
4. **Prüfe die Konfigurationsdatei** — überprüfe deine Einstellungen in `config.json`

---

## Schnellreferenz

### Standard-Port
```
3000
```

### Zugriffs-URLs
```
Startseite:  http://DEINE-PC-IP:3000/
Editor:      http://DEINE-PC-IP:3000/editor
Panel:       http://DEINE-PC-IP:3000/panel?name=PanelName
```

### Dateispeicherorte
```
Konfiguration:     config.json
Panels:            user/panels/
Eigene Blöcke:     user/blocks/
Sprachbefehle:     user/speech_commands.json
Assets:            user/assets/
```

### Joystick-Limits
```
Joysticks:  10 (Indizes 0-9)
Buttons:    16 pro Joystick (IDs 0-15)
Achsen:     8 pro Joystick (IDs 0-7)
```

### Achsen-Namen
```
0: X         1: Y         2: Z         3: RX
4: RY        5: RZ        6: Throttle  7: Rudder
```

### Systemmetriken Data Keys
```
cpu_usage      memory_usage    disk_usage
network_rx     network_tx
```

---

## Herzlichen Glückwunsch!

Du hast das OmniPanel-go-Benutzerhandbuch abgeschlossen. Du weißt jetzt, wie du:

- OmniPanel-go einrichtest und startest
- Bedienpanels erstellst und anpasst
- Alle Block-Typen nutzt (Buttons, Slider, Touchpads, Command Blocks, Datenanzeigen)
- Panels mit Seiten und Bereichen organisierst
- Live-Daten anzeigst
- Dein Panel mit Sprachbefehlen steuerst
- Musikwiedergabe mit dem Medienwiedergabe-Block steuerst
- Live-Nachrichten und Blogs mit RSS-Feeds anzeigst
- Eigene Blöcke erstellst
- Häufige Probleme behebst

Jetzt baue ein paar großartige Bedienpanels und genieße deine Spiele mit einem individuellen Cockpit-Dashboard!

Weiter zu [Medienwiedergabe](15-media-player.md) zum Anzeigen und Steuern von Musik/Videos, oder schau dir [RSS-Feeds](16-rss-feeds.md) für Live-Nachrichten und Blogs an.
