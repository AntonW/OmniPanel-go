# Kapitel 10: Live-Datenanzeige

Du möchtest die CPU-Auslastung, RAM-Nutzung oder eigene Daten direkt auf deinem Panel sehen? Dieses Kapitel deckt ab, wie du Live-Daten, die sich in Echtzeit aktualisieren, auf deinen Bedienpanels anzeigst.

## Was sind Live-Daten?

Live-Daten sind Informationen, die sich in Echtzeit auf deinem Panel aktualisieren. Statt statischer Buttons und Slider kannst du Anzeigen haben, die Folgendes zeigen:

- CPU-Auslastung in Prozent
- Arbeitsspeicher (RAM)-Nutzung
- Festplattennutzung
- Netzwerk-Download/Upload-Geschwindigkeit
- Beliebige eigene Daten, die du einspeist

## Systemmetriken (Eingebaut)

OmniPanel-go sammelt automatisch alle 500 Millisekunden (zweimal pro Sekunde) Systemmetriken. Diese Metriken sind immer verfügbar — du musst nichts konfigurieren, um sie zu nutzen.

### Verfügbare Systemmetriken

| Data Key | Was er zeigt | Beispielwert |
|----------|--------------|---------------|
| `cpu_usage` | CPU-Auslastung in Prozent | 45,2 |
| `memory_usage` | RAM-Auslastung in Prozent | 62,8 |
| `disk_usage` | Festplattennutzung in Prozent | 71,3 |
| `network_rx` | Netzwerk-Download-Geschwindigkeit (KB/s) | 1250,5 |
| `network_tx` | Netzwerk-Upload-Geschwindigkeit (KB/s) | 342,1 |

> **Hinweis:** Unter Windows ist nur `disk_usage` für Systemmetriken verfügbar. Volle Systemmetriken (CPU, RAM, Netzwerk) sind unter Linux verfügbar.

---

## Data Display-Block

### Was er macht

Der Data Display-Block zeigt einen Live-Datenwert auf deinem Panel an. Er aktualisiert sich automatisch, wann immer sich die Daten ändern.

### Einstellungen

| Einstellung | Was sie macht | Beispielwerte |
|---------|-------------|----------------|
| **Data Key** | Der Name der anzuzeigenden Daten | `cpu_usage`, `memory_usage` |
| **Title** | Label über dem Wert | „CPU", „RAM", „Network Down" |
| **Unit** | Text nach dem Wert | „%", „KB/s", „°C" |
| **Dezimalstellen** | Wie viele Stellen nach dem Komma | 0, 1, 2 |
| **Textfarbe** | Farbe des Wert-Textes | Beliebige Farbe |
| **Hintergrundfarbe** | Block-Hintergrundfarbe | Beliebige Farbe |
| **Größe** | Wie groß die Anzeige ist | Small, Medium, Large |

### Einen Data Display-Block hinzufügen

1. Ziehe einen **Data Display**-Block auf deine Arbeitsfläche
2. Klicke auf das Zahnrad-Symbol
3. Setze den **Data Key** auf eine der Systemmetriken (z. B. `cpu_usage`)
4. Setze den **Title** (z. B. „CPU")
5. Setze die **Unit** (z. B. „%")
6. Setze **Dezimalstellen** auf 1 für eine Stelle nach dem Komma
7. Passe Farben und Größe an
8. Speichere die Einstellungen

### Beispiel-Konfigurationen

**CPU-Auslastungsanzeige:**
- Data Key: `cpu_usage`
- Title: `CPU`
- Unit: `%`
- Dezimalstellen: `1`
- Ergebnis: Zeigt „CPU: 45,2%"

**Netzwerk-Download-Geschwindigkeit:**
- Data Key: `network_rx`
- Title: `Download`
- Unit: `KB/s`
- Dezimalstellen: `0`
- Ergebnis: Zeigt „Download: 1250 KB/s"

**Arbeitsspeicher-Nutzung:**
- Data Key: `memory_usage`
- Title: `RAM`
- Unit: `%`
- Dezimalstellen: `1`
- Ergebnis: Zeigt „RAM: 62,8%"

### Visuelles Feedback

Wenn sich ein Datenwert ändert, **blinkt die Anzeige kurz** auf, um deine Aufmerksamkeit zu erregen. Das hilft dir, Änderungen auf einen Blick zu bemerken, ohne ständig auf die Zahlen zu starren.

---

## Eigene Daten einspeisen

Du kannst deine eigenen Daten von externen Programmen, Skripten oder Webanwendungen an OmniPanel-go senden. Diese Daten erscheinen dann auf deinem Panel genau wie Systemmetriken.

### Methode 1: HTTP-API

Sende eine POST-Anfrage an den Daten-Endpunkt von OmniPanel-go:

```
POST http://DEINE-PC-IP:3000/api/data/push
Content-Type: application/json

{
  "key": "server_temp",
  "value": 42.5,
  "unit": "C",
  "source": "sensors"
}
```

| Feld | Was es macht | Erforderlich |
|-------|-------------|----------|
| `key` | Der Datenname (in Data Display-Blöcken verwendet) | Ja |
| `value` | Der Datenwert (Zahl oder Text) | Ja |
| `unit` | Einheiten-Label (optional, wird auf der Anzeige gezeigt) | Nein |
| `source` | Woher die Daten kommen (zur Organisation) | Nein |

### Beispiel: curl nutzen (Linux/Mac)

```bash
curl -X POST http://192.168.1.100:3000/api/data/push \
  -H "Content-Type: application/json" \
  -d '{"key": "gpu_temp", "value": 65, "unit": "°C"}'
```

### Beispiel: PowerShell nutzen (Windows)

```powershell
Invoke-RestMethod -Uri "http://192.168.1.100:3000/api/data/push" -Method POST -ContentType "application/json" -Body '{"key": "gpu_temp", "value": 65, "unit": "°C"}'
```

### Beispiel: Python nutzen

```python
import requests

requests.post("http://192.168.1.100:3000/api/data/push", json={
    "key": "gpu_temp",
    "value": 65,
    "unit": "°C"
})
```

### Eigene Daten anzeigen

Sobald du Daten mit einem Key (wie `gpu_temp`) eingespeist hast, kannst du sie auf deinem Panel anzeigen:

1. Füge einen Data Display-Block hinzu
2. Setze den Data Key auf `gpu_temp`
3. Setze den Title auf „GPU Temp"
4. Setze die Unit auf „°C"
5. Der Block zeigt jetzt deine eigenen Daten

### Methode 2: WebSocket

Für Anwendungen, die kontinuierlich Daten einspeisen müssen, kannst du WebSocket-Verbindungen nutzen. Das ist fortgeschrittener und wird typischerweise von Entwicklern genutzt, die Integrationen bauen.

---

## Star Citizen Themen-Data Display

### Was er macht

Dieselbe Funktionalität wie das Standard-Data Display, aber mit einem Sci-Fi-Visuallstil — leuchtender Text und eckiges Design.

### Einstellungen

Dieselben wie der Standard-Data Display-Block, mit zusätzlicher visueller Anpassung für die Sci-Fi-Ästhetik.

---

## Praktische Beispiele

### Beispiel 1: Systemmonitor-Panel

Erstelle ein Panel, das den Gesundheitszustand deines PCs zeigt:

- **CPU-Auslastung** — Data Key: `cpu_usage`, Unit: `%`
- **RAM-Nutzung** — Data Key: `memory_usage`, Unit: `%`
- **Festplattennutzung** — Data Key: `disk_usage`, Unit: `%`
- **Download-Geschwindigkeit** — Data Key: `network_rx`, Unit: `KB/s`
- **Upload-Geschwindigkeit** — Data Key: `network_tx`, Unit: `KB/s`

### Beispiel 2: Game-Server-Monitor

Wenn du einen Game-Server betreibst, speise Serverdaten in dein Panel ein:

```bash
# Spieleranzahl einspeisen
curl -X POST http://192.168.1.100:3000/api/data/push \
  -H "Content-Type: application/json" \
  -d '{"key": "players", "value": 24, "unit": "players"}'

# Server-FPS einspeisen
curl -X POST http://192.168.1.100:3000/api/data/push \
  -H "Content-Type: application/json" \
  -d '{"key": "server_fps", "value": 60, "unit": "FPS"}'
```

Zeige sie dann auf deinem Panel mit Data Display-Blöcken an.

### Beispiel 3: Smart-Home-Dashboard

Speise Sensordaten von Smart-Home-Geräten ein:

```bash
# Temperatur
curl -X POST http://192.168.1.100:3000/api/data/push \
  -H "Content-Type: application/json" \
  -d '{"key": "living_room_temp", "value": 22.5, "unit": "°C"}'

# Luftfeuchtigkeit
curl -X POST http://192.168.1.100:3000/api/data/push \
  -H "Content-Type: application/json" \
  -d '{"key": "living_room_humidity", "value": 45, "unit": "%"}'
```

---

## Tipps für Datenanzeigen

### Aktualisierungsfrequenz

- Systemmetriken aktualisieren sich alle **500ms** (zweimal pro Sekunde)
- Eigene Daten aktualisieren sich **sofort**, wenn du sie einspeist
- Dein Panel spiegelt Änderungen wider, sobald sie eintreffen

### Dezimalstellen

- Nutze **0 Dezimalstellen** für ganze Zahlen (Spieleranzahl, FPS)
- Nutze **1 Dezimalstelle** für Prozentwerte und Temperaturen
- Nutze **2 Dezimalstellen** für präzise Messungen

### Farbkodierung

Nutze Farben, um Status anzuzeigen:

- **Grün** — normale Werte (CPU unter 50%)
- **Gelb** — Vorsichtswerte (CPU 50-80%)
- **Rot** — Warnwerte (CPU über 80%)

> **Hinweis:** Der Data Display-Block ändert nicht automatisch die Farben basierend auf Werten. Du müsstest mehrere Anzeigen erstellen oder eigene Blöcke für dynamische Färbung nutzen.

### Größe

- **Small** — gut für kompakte Panels mit vielen Anzeigen
- **Medium** — ausgewogen für die meisten Anwendungen
- **Large** — gut für wichtige Metriken, die du auf einen Blick sehen möchtest

## Was kommt als Nächstes?

Du möchtest dein Panel mit deiner Stimme steuern? Lerne mehr über [Sprachbefehle](11-speech-commands.md).
