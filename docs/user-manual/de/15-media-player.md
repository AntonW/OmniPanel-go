# Kapitel 15: Medienwiedergabe

Möchtest du sehen, welche Musik gerade läuft, und sie von deinem Panel aus steuern? Dieses Kapitel behandelt den Medienwiedergabe-Block und wie du ihn mit deinen Musik-Apps unter Linux und Windows verbindest.

## Was ist der Medienwiedergabe-Block?

Der Medienwiedergabe-Block zeigt Informationen über das aktuell abgespielte Medium (Musik, Video, Podcast) und ermöglicht dir die Steuerung der Wiedergabe mit Schaltflächen. Im Automatikmodus funktioniert er unter Linux über MPRIS und unter Windows über SMTC (System Media Transport Controls) — einschließlich Spotify, VLC, Browser-Mediensitzungen und vielen weiteren.

### Was wird angezeigt

- **Cover-Bild** — Album-Artwork des aktuellen Titels
- **Titel** — Song- oder Videoname
- **Künstler** — Interpret oder Kanalname
- **Fortschrittsbalken** — wie weit du im Track bist
- **Lautstärke** — aktuelle Player-Lautstärke (wird in den Lautstärke-Tasten angezeigt)
- **Steuerungstasten** — vorheriger, Play/Pause, nächster, Lautstärke runter, Lautstärke hoch

---

## Steuerungsmodi

Der Medienwiedergabe-Block hat zwei Steuerungsmodi. Wähle den, der zu deinem Setup passt:

### MPRIS-Modus (Automatischer Medienmodus)

Das ist der automatische Modus. OmniPanel-go verbindet sich automatisch mit dem Medien-System der Plattform:
- **Linux:** MPRIS über D-Bus
- **Windows:** SMTC (System Media Transport Controls)

**Was du bekommst:**
- Echtzeit-Updates wenn sich der Song ändert
- Cover-Bild vom Medienplayer
- Play/Pause/Vor/Zurück-Tasten die die tatsächliche App steuern
- Fortschrittsbalken der sich während der Wiedergabe bewegt

**Voraussetzungen:**
- Linux-Desktop (KDE Plasma, GNOME, etc.) mit MPRIS-kompatiblem Player oder Windows 10/11 mit App, die Mediensteuerung bereitstellt
- Medien-Watcher in `config.json` aktivieren (Abschnitt `mpris`, siehe unten)

### Tastatur-Modus (Alle Plattformen)

Dieser Modus sendet Tastatur-Medientasten (wie die Play/Pause-Taste auf deiner Tastatur) zur Steuerung von Medien. Er funktioniert auf jedem Betriebssystem, erfordert aber dass die Medien-App auf Tastaturkürzel reagiert.

**Was du bekommst:**
- Tasten die Tastatur-Medientasten simulieren
- Funktioniert unter Windows, macOS und Linux
- Keine automatischen Daten-Updates — du stellst Titel/Künstler/Cover manuell ein

---

## Automatischen Medienmodus einrichten

### Schritt 1: MPRIS in der Config aktivieren

Öffne `config.json` und füge den `mpris`-Abschnitt hinzu oder aktualisiere ihn:

```json
{
  "mpris": {
    "enabled": true,
    "poll_interval": 1000
  }
}
```

- `enabled`: Auf `true` setzen um MPRIS-Überwachung zu aktivieren
- `poll_interval`: Wie oft nach Updates gesucht wird, in Millisekunden (1000 = 1 Sekunde). Minimum ist 500ms.

Starte OmniPanel-go nach dieser Änderung neu.

### Schritt 2: Medien-App starten

Öffne eine Medien-App, die System-Mediensteuerung unterstützt:
- **Spotify** (Desktop-App)
- **VLC** (jede Plattform)
- **Firefox**, **Chrome** oder **Edge** (mit laufender Medienwiedergabe)
- **Rhythmbox**, **Audacious**, **Clementine** und viele weitere

### Schritt 3: Medienwiedergabe-Block hinzufügen

1. Öffne den Editor
2. Finde die **Medien**-Kategorie in der Block-Bibliothek (🎵 Symbol)
3. Ziehe den **Medienwiedergabe**-Block auf deine Arbeitsfläche
4. Klicke auf das Zahnrad-Symbol um die Einstellungen zu öffnen
5. Setze **Steuerungsmodus** auf `mpris`
6. Aktiviere die Features die du möchtest (Cover-Bild, Titel, Künstler, Fortschrittsbalken)
7. Speichere das Panel

### Schritt 4: Testen

Öffne dein Panel in einem Browser. Wenn du Musik in deinem Medienplayer abspielst, sollte der Block automatisch Titel, Künstler und Cover-Bild anzeigen. Die Steuerungstasten sollten funktionieren um abzuspielen, zu pausieren, zu überspringen und die Lautstärke anzupassen.

> **Hinweis für verteilte Setups (serve + connect mode):** Der automatische Medienmodus funktioniert genauso — der Medienwiedergabe-Block kommuniziert über den Relay-Server mit dem Host-Agent. Aktiviere `mpris` in der `config.json` des Host-Agent (nicht in der Server-Config). Der Host-Agent muss mit dem Relay-Server verbunden sein.

---

## Zwischen Medienplayern wechseln

Wenn du mehrere Medienplayer gleichzeitig laufen hast (zum Beispiel Spotify und VLC), zeigt OmniPanel-go kleine Tabs über dem Cover-Bild-Bereich an. Diese Tabs lassen dich auswählen, welcher Player angezeigt und gesteuert wird.

**So funktioniert es:**
- Wenn nur ein Player läuft, werden keine Tabs angezeigt — der Block zeigt einfach diesen Player
- Wenn zwei oder mehr Player laufen, erscheinen Tabs oben auf dem Cover-Bild
- Jeder Tab zeigt den Namen des Players (z.B. "Spotify", "VLC media player")
- Klicke auf einen Tab um zu diesem Player zu wechseln — Cover-Bild, Titel, Künstler und Fortschrittsbalken aktualisieren sich sofort
- Der aktive Tab ist hervorgehoben damit du siehst welcher Player gerade ausgewählt ist
- Wenn du einen Player schließt der ausgewählt war, wechselt OmniPanel-go automatisch zu einem anderen verfügbaren Player

**Beispiel:** Du hast Spotify mit Musik und VLC mit einem Video. Die Tabs zeigen "Spotify" und "VLC media player". Klicke auf "VLC media player" um das Video anzuzeigen und zu steuern. Die Steuerungstasten (Play, Pause, Weiter, etc.) beeinflussen jetzt VLC statt Spotify.

---

## Einstellungen des Medienwiedergabe-Blocks

| Einstellung | Was sie bewirkt | Beispielwerte |
|-------------|-----------------|---------------|
| **Steuerungsmodus** | Wie der Block Daten erhält und Befehle sendet | `mpris` (automatisch), `keyboard` (manuell) |
| **Cover anzeigen** | Cover-Bild-Bereich ein- oder ausblenden | An / Aus |
| **Cover-Bild** | Manuelle Bild-URL (nur Tastatur-Modus) | `https://...` oder leer lassen |
| **Titel anzeigen** | Track-Titel ein- oder ausblenden | An / Aus |
| **Titel** | Manueller Titel-Text (nur Tastatur-Modus) | "Mein Song" |
| **Künstler anzeigen** | Künstlername ein- oder ausblenden | An / Aus |
| **Künstler** | Manueller Künstler-Text (nur Tastatur-Modus) | "Künstlername" |
| **Fortschritt anzeigen** | Fortschrittsbalken ein- oder ausblenden | An / Aus |
| **Fortschrittswert** | Manueller Fortschritt 0-100% (nur Tastatur-Modus) | 0 bis 100 |
| **Schriftgröße** | Textgröße | 8 bis 24 |
| **Hintergrundfarbe** | Block-Hintergrund | Jede Farbe |
| **Tastenfarbe** | Farbe der Steuerungstasten | Jede Farbe |
| **Tastenfarbe aktiv** | Farbe bei Hover/Aktiv | Jede Farbe |
| **Textfarbe** | Farbe von Titel und Künstler | Jede Farbe |
| **Randfarbe** | Farbe des Block-Rands | Jede Farbe |
| **Rand-Radius** | Ecken-Rundung (0-100) | 0 (eckig) bis 100 (rund) |

---

## Fehlerbehebung

### Kein Cover-Bild sichtbar

Manche Browser (Chrome, Firefox) speichern Cover-Bilder in temporären Dateien die der Block über einen Proxy erreichen kann. Wenn kein Cover-Bild angezeigt wird:
1. Stelle sicher dass MPRIS in `config.json` aktiviert ist
2. Prüfe ob dein Medienplayer tatsächlich etwas abspielt
3. Probiere einen anderen Medienplayer — manche expose kein Cover-Bild über MPRIS
4. **Im serve/connect mode:** Stelle sicher dass der Host-Agent mit dem Relay-Server verbunden ist. Cover-Bilder werden über den Relay-Server gesendet, also bedeutet ein getrennter Host kein Cover-Bild.
5. **Mit aktivierter Authentifizierung:** Der Cover-Bild-Endpunkt benötigt das Auth-Token. Wenn du einen 401-Fehler siehst, prüfe ob dein Token gültig ist und der Browser es gespeichert hat (achte auf die Weiterleitung zur Login-Seite).

### Tasten steuern meine Musik nicht

Im automatischen Medienmodus (`Steuerungsmodus = mpris`):
1. Stelle sicher dass dein Medienplayer läuft und abspielt
2. Prüfe die Server-Logs auf MPRIS-Fehler
3. Manche Player müssen "aktiv" sein (ein offenes Fenster haben) um Befehle anzunehmen
4. **Im serve/connect mode:** Ein 500-Fehler bei Tastenklicks bedeutet meist dass der Host-Agent getrennt ist oder MPRIS auf dem Host nicht aktiviert ist. Prüfe die Host-Agent-Logs auf "MPRIS watcher failed to start" Meldungen.

Im Tastatur-Modus:
1. Stelle sicher dass die Medien-App auf Tastatur-Medientasten reagiert
2. Prüfe ob die Tastatur-Tasten korrekt in den Block-Einstellungen zugewiesen sind

### "Keine Medienplayer verbunden"

Das bedeutet, OmniPanel-go kann keine aktiven Mediensitzungen auf deinem System finden:
1. Stelle sicher dass `mpris.enabled` auf `true` in `config.json` steht
2. Starte OmniPanel-go nach dem Aktivieren von MPRIS neu
3. Starte einen Medienplayer (Spotify, VLC, etc.)
4. Unter KDE Plasma stelle sicher dass das "Medienwiedergabe"-Widget deinen Player sehen kann
5. **Im serve/connect mode:** Dieser Fehler tritt auf wenn der Relay-Server den Host-Agent nicht erreichen kann. Überprüfe ob der Host-Agent läuft und verbunden ist (achte auf den Host-IP-Indikator auf der Startseite).

---

## Unterstützte Medienplayer

Jede Anwendung mit System-Mediensteuerung funktioniert. Unter Linux sind das MPRIS2-Apps, unter Windows SMTC-Apps. Übliche Player:

| Player | Cover-Bild | Steuerung | Hinweise |
|--------|------------|-----------|----------|
| Spotify (Desktop) | Ja | Vollständig | Beste Erfahrung |
| VLC | Ja | Vollständig | Funktioniert super |
| Firefox | Ja | Vollständig | Über Plasma Browser Integration |
| Chrome/Chromium | Manchmal | Vollständig | Cover-Bild hängt von der Seite ab |
| Rhythmbox | Ja | Vollständig | GNOME Standard |
| Audacious | Ja | Vollständig | Leichtgewichtiger Player |
| Clementine | Ja | Vollständig | Funktionsreicher Player |

---

[← Zurück: Tipps und Fehlerbehebung](14-tips-and-troubleshooting.md) · [Weiter: RSS-Feeds →](16-rss-feeds.md)
