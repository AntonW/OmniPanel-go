# Kapitel 2: Erste Schritte

Dieses Kapitel führt dich durch den ersten Start von OmniPanel-go und den Zugriff von deinem Tablet oder Handy aus.

## Betriebsmodi

OmniPanel-go kann auf drei verschiedene Arten laufen, je nach deinem Setup:

| Modus | Wann verwenden | Befehl |
|-------|----------------|--------|
| **Standard** | Alles auf einem PC | `./omnipanel-go` |
| **Server** | WebUI auf zentralem PC, Host hinter Firewall | `./omnipanel-go serve` |
| **Host-Agent** | Verbindung zu einem zentralen Server | `./omnipanel-go connect <IP>:<PORT>` |

Die meisten Nutzer sollten den **Standard**-Modus verwenden — er ist am einfachsten und funktioniert super im lokalen Netzwerk.

Die Modi **Server** und **Host-Agent** sind für fortgeschrittene Setups, bei denen du die WebUI auf einem Rechner (immer eingeschalteter Server) und die eigentliche Eingabesimulation auf einem anderen Rechner (möglicherweise hinter einer Firewall) betreiben möchtest. Siehe [Kapitel 14: Tipps und Fehlerbehebung](14-tips-and-troubleshooting.md) für weitere Details.

### Authentifizierung für Server- und Host-Agent-Modi

Wenn dein Server-Administrator ein geheimes Token eingerichtet hat, musst du es beim Zugriff auf den Server angeben. Das schützt die WebUI und Host-Verbindungen vor unbefugtem Zugriff.

**Zugriff auf die WebUI mit Token:**
```
http://server-ip:3000/?token=dein-geheimes-token
```

**Verbinden des Host-Agenten mit Token:**
Setze das Token in `config.json` auf dem Host-Rechner:
```json
{
  "server_address": "10.0.0.1:3000",
  "auth_token": "dein-geheimes-token"
}
```
Oder nutze die Umgebungsvariable:
```bash
export OMNIPANEL_AUTH_TOKEN=dein-geheimes-token
./omnipanel-go connect 10.0.0.1:3000
```

> **Hinweis:** Wenn kein Token konfiguriert ist, funktioniert alles wie zuvor — Authentifizierung ist optional.

### Docker-Container (Server-Modus)

Für Nutzer, die mit Docker vertraut sind, kann der Server-Modus auch als Container laufen:

```bash
docker run --rm -p 3000:3000 \
  -v $(pwd)/user:/var/run/ko/user \
  -w /var/run/ko \
  omnipanel-go:latest serve
```

Der Container kopiert beim ersten Start automatisch Standard-Panels, Blöcke und Themes in deinen `user/`-Ordner. Deine Anpassungen bleiben über Container-Neustarts hinweg erhalten. Spracherkennung über Vosk ist im Container nicht verfügbar — verwende stattdessen llama-cpp-server (HTTP-API).

## Schritt 1: OmniPanel-go auf deinem PC starten

### Unter Windows (Standard-Modus)

1. Finde die Datei `omnipanel-go.exe` in deinem OmniPanel-go-Ordner
2. Doppelklicke darauf, um sie zu starten
3. Ein Befehlsfenster öffnet sich — **lasse es geöffnet**, solange du OmniPanel-go nutzt
4. Du solltest eine Meldung sehen, dass der Server auf Port 3000 läuft

### Unter Linux (Standard-Modus)

1. Öffne ein Terminal
2. Navigiere zu deinem OmniPanel-go-Ordner
3. Führe den Befehl aus:
   ```
   ./omnipanel-go
   ```
4. Du solltest eine Meldung sehen, dass der Server auf Port 3000 läuft

> **Wichtig:** Lass dieses Fenster geöffnet. Wenn du es schließt, stoppt OmniPanel-go und deine Panels funktionieren nicht mehr.

### Log-Ausgabe

OmniPanel-go zeigt standardmäßig farbige Log-Nachrichten an. Du kannst das Log-Format bei Bedarf ändern:

| Format | Verwendung | Am besten für |
|--------|-----------|---------------|
| **Farbig** (Standard) | `./omnipanel-go` | Normale Nutzung — leicht lesbar |
| **Plain Text** | `./omnipanel-go --log-format=text` | Speichern in einer Log-Datei |
| **JSON** | `./omnipanel-go --log-format=json` | Fortgeschrittene Nutzer mit Log-Tools |

Du kannst auch die `LOG_FORMAT`-Umgebungsvariable setzen, damit du den Parameter nicht jedes Mal eingeben musst:
```
export LOG_FORMAT=json
./omnipanel-go
```

## Schritt 2: Die IP-Adresse deines PCs finden

Dein Tablet muss wissen, wo es OmniPanel-go in deinem Netzwerk findet. Diese Adresse ist die IP-Adresse deines PCs.

### Unter Windows

1. Drücke `Windows-Taste + R`
2. Tippe `cmd` ein und drücke Enter
3. Tippe im schwarzen Fenster:
   ```
   ipconfig
   ```
4. Suche nach **IPv4-Adresse** unter deinem WiFi- oder Ethernet-Adapter
5. Sie sieht ungefähr so aus: `192.168.1.100`

### Unter Linux

1. Öffne ein Terminal
2. Tippe:
   ```
   ip addr show
   ```
3. Suche nach `inet` unter deinem `wlan0` (WiFi) oder `eth0` (Ethernet) Interface
4. Es sieht ungefähr so aus: `192.168.1.100`

## Schritt 3: OmniPanel-go auf deinem PC öffnen

1. Öffne deinen Webbrowser (Chrome, Firefox, Edge, etc.)
2. Tippe in die Adressleiste:
   ```
   http://localhost:3000
   ```
3. Drücke Enter
4. Du solltest die **Startseite** mit deiner Panel-Liste sehen

> **Hinweis:** `localhost` bedeutet „dieser Computer". Das funktioniert nur, wenn du direkt am PC bist, auf dem OmniPanel-go läuft.

## Schritt 4: OmniPanel-go auf deinem Tablet oder Handy öffnen

1. Öffne den Webbrowser auf deinem Tablet oder Handy
2. Tippe in die Adressleiste:
   ```
   http://DEINE-PC-IP:3000
   ```
   Ersetze `DEINE-PC-IP` durch die IP-Adresse, die du in Schritt 2 gefunden hast. Zum Beispiel:
   ```
   http://192.168.1.100:3000
   ```
3. Drücke Los/Enter
4. Du solltest dieselbe **Startseite** wie auf deinem PC sehen

> **Tipp:** Speichere diese Seite als Lesezeichen auf deinem Tablet, damit du sie nicht jedes Mal eintippen musst.

## Schritt 5: Überprüfen, ob es funktioniert

Auf der Startseite solltest du sehen:

- Eine Liste gespeicherter Panels (kann leer sein, wenn du OmniPanel-go zum ersten Mal nutzt)
- Einen Bereich, der deine Verbindungs-URL anzeigt
- Ein Verbindungsprotokoll am unteren Rand

Wenn du all das siehst, herzlichen Glückwunsch — OmniPanel-go läuft und dein Tablet ist verbunden!

## Häufige Probleme

### „Diese Website kann nicht erreicht werden" auf deinem Tablet

- Stelle sicher, dass sowohl dein PC als auch dein Tablet im **selben WiFi-Netzwerk** sind
- Prüfe, ob OmniPanel-go noch auf deinem PC läuft (das Befehlsfenster sollte geöffnet sein)
- Überprüfe die IP-Adresse noch einmal — sie könnte sich geändert haben, wenn dein Router neu gestartet wurde

### Firewall blockiert die Verbindung (Windows)

Wenn du OmniPanel-go zum ersten Mal startest, fragt Windows möglicherweise, ob du es durch die Firewall lassen möchtest. Klicke auf **Zulassen**.

Falls du diese Meldung verpasst hast:

1. Öffne Windows-Sicherheit
2. Gehe zu Firewall- und Netzwerkschutz
3. Klicke auf „App durch Firewall zulassen"
4. Finde `omnipanel-go.exe` und stelle sicher, dass sowohl Privat als auch Öffentlich aktiviert sind

### Falsche IP-Adresse

Die IP-Adresse deines PCs kann sich ändern, wenn dein Router neu startet. Wenn die Verbindung plötzlich nicht mehr funktioniert, überprüfe deine IP-Adresse noch einmal mit den Schritten aus Schritt 2.

## Was kommt als Nächstes?

Jetzt, wenn OmniPanel-go läuft, erkunde die [Startseite](03-the-start-page.md) — dein Dashboard zum Verwalten von Panels und Verbindungen.
