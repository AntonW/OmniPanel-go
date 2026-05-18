# Kapitel 11: Sprachbefehle

Steuere dein Panel mit deiner Stimme. Dieses Kapitel deckt ab, wie du Spracherkennung einrichtest und nutzt, um Befehle auszulösen, Buttons zu drücken und Slider anzupassen — einfach durch Sprechen.

## Was sind Sprachbefehle?

Sprachbefehle lassen dich dein Panel durch Sprechen statt durch Tippen steuern. Du kannst:

- Shell-Befehle ausführen, indem du einen Satz sagst
- HTTP-Anfragen mit deiner Stimme senden
- Buttons auf deinem Panel drücken, indem du sprichst
- Slider auf bestimmte Werte einstellen, indem du eine Zahl sagst

## Zwei Aufnahmeorte

OmniPanel-go kann Audio von zwei verschiedenen Orten aufnehmen. Dies wird durch die `recording_location`-Einstellung in `config.json` gesteuert.

### Client-Aufnahme (Standard)

Das Mikrofon deines Tablets oder Handys nimmt das Audio auf. Das Audio wird zur Verarbeitung über das Netzwerk an deinen PC gesendet.

**So funktioniert es:**
1. Du hältst den Mikrofon-Button auf deinem Panel gedrückt
2. Der Browser deines Tablets fragt nach Mikrofon-Berechtigung (nur beim ersten Mal)
3. Dein Tablet nimmt deine Stimme auf
4. Das Audio wird an deinen PC über das Netzwerk gesendet
5. Dein PC verarbeitet es und führt den Befehl aus

**Am besten geeignet für:**
- Push-to-Talk-Modus
- Wenn dein Tablet näher an dir ist als dein PC
- Wenn dein PC kein Mikrofon hat

**Anforderungen:**
- Browser-Mikrofon-Berechtigung (du wirst beim ersten Mal gefragt)
- Netzwerkverbindung zwischen Tablet und PC

### Host-Aufnahme (Server-seitig)

Das Mikrofon deines Gaming-PCs nimmt das Audio direkt auf. Dein Tablet sendet nur Start/Stop-Signale — kein Audio verlässt dein Tablet.

**So funktioniert es:**
1. Du hältst den Mikrofon-Button auf deinem Panel gedrückt
2. Dein Tablet sendet ein „Aufnahme starten"-Signal an deinen PC
3. Dein PC beginnt, von seinem eigenen Mikrofon aufzunehmen
4. Du sprichst deinen Befehl
5. Du lässt den Button los — dein Tablet sendet ein „Aufnahme stoppen"-Signal
6. Dein PC verarbeitet das gerade aufgenommene Audio und führt den Befehl aus

**Am besten geeignet für:**
- Wake-Word-Modus (PC hört ständig zu)
- Wenn dein PC ein gutes Mikrofon hat (Headset, Webcam, etc.)
- Wenn du dem Browser keine Mikrofon-Berechtigung geben möchtest
- Wenn dein Tablet weit von dir entfernt ist, aber dein PC-Mikrofon nah ist

**Anforderungen:**
- Ein funktionierendes Mikrofon, das mit deinem PC verbunden ist
- Unter Linux: PulseAudio oder ALSA (normalerweise vorinstalliert)
- Unter Windows: Jedes Mikrofon, das mit Windows funktioniert

> **Tipp:** Host-Aufnahme bedeutet, dass dein Tablet niemals Mikrofon-Berechtigung benötigt. Der Browser sendet nur winzige Steuersignale. Die gesamte Audioverarbeitung erfolgt auf deinem PC.

---

## Zwei Aktivierungsmodi

### Push-to-Talk

Halte einen Mikrofon-Button auf deinem Panel gedrückt, sprich deinen Befehl, dann lass den Button los. Das ist der Standardmodus und funktioniert in allen Browsern.

**So funktioniert es:**
1. Ein schwebender Mikrofon-Button erscheint auf deinem Panel
2. Halte den Button gedrückt
3. Sprich deinen Befehl
4. Lass den Button los
5. OmniPanel-go verarbeitet, was du gesagt hast, und führt den passenden Befehl aus

### Wake Word

OmniPanel-go hört kontinuierlich nach einem bestimmten Wort (dem „Wake Word"). Wenn du es sagst, beginnt das System, nach deinem Befehl zu hören.

**So funktioniert es:**
1. OmniPanel-go hört ständig zu (aber nimmt nicht auf)
2. Sage das Wake Word (Standard: „omnipanel-go")
3. Nach dem Wake Word sprich deinen Befehl
4. OmniPanel-go verarbeitet und führt den passenden Befehl aus

> **Hinweis:** Der Wake-Word-Modus funktioniert nur in Chrome- und Edge-Browsern. Safari und Firefox unterstützen ihn nicht.

---

## Zwei Spracherkennungs-Engines

### Vosk (Offline)

Vosk läuft komplett auf deinem PC — keine Internetverbindung nötig. Es nutzt **grammatik-beschränkte Erkennung**, was bedeutet, dass es nur versucht, die Sätze zu erkennen, die du als Sprachbefehle definiert hast. Das verbessert die Genauigkeit dramatisch im Vergleich zum Versuch, beliebige englische Wörter zu erkennen.

**So funktioniert grammatik-beschränkte Erkennung:**
- OmniPanel-go sammelt alle deine Sprachbefehl-Sätze und Aliase
- Es erstellt eine „Grammatik" — eine Liste erlaubter Sätze — und übergibt sie an Vosk
- Vosk versucht nur, deine definierten Sätze zu erkennen, ignoriert alles andere
- Wenn du neue Sprach-Trigger über den Editor hinzufügst, wird die Grammatik automatisch aktualisiert

**Vorteile:**
- Funktioniert ohne Internet
- Schnelle Reaktionszeit
- Privat (Audio verlässt nie deinen PC)
- **Hohe Genauigkeit** für definierte Befehle (Grammatik-beschränkter Modus)

**Nachteile:**
- Erfordert das Herunterladen eines Modells (~50MB)
- Erkennt nur Sätze, die du definiert hast (versteht keine freie Sprache)
- Begrenzte Sprachunterstützung

**Einrichtung:**
1. Setze in `config.json` `"stt_engine": "vosk"`
2. Lass `vosk_model_path` leer, um das Modell automatisch herunterzuladen
3. Oder setze `vosk_model_path` auf ein Modell, das du bereits heruntergeladen hast

### llama-cpp-server (AI-Server)

Nutzt eine bestehende llama-cpp-server-Instanz für Spracherkennung. Das unterstützt Whisper-kompatible Modelle oder AI-Chat-Modi.

**Vorteile:**
- Genauer als Vosk für freie Sprache
- Unterstützt viele Sprachen
- Kann natürliche Sprachbefehle verstehen

**Nachteile:**
- Erfordert einen separaten Server
- Braucht leistungsfähigere Hardware
- Etwas langsamere Reaktionszeit

**Einrichtung:**
1. Setze in `config.json` `"stt_engine": "llama-cpp"`
2. Setze `llama_cpp_url` auf deine Server-Adresse (Standard: `http://localhost:8080`)
3. Setze `llama_cpp_api_mode` auf `"transcriptions"` (Whisper-Stil) oder `"chat"` (AI-Chat)
4. Wenn du den Chat-Modus nutzt, setze `llama_cpp_model` und `llama_cpp_prompt`

---

## Sprache konfigurieren

### Schritt 1: Sprache in config.json aktivieren

Öffne `config.json` und finde den `speech`-Bereich:

```json
{
  "speech": {
    "enabled": true,
    "recording_location": "client",
    "trigger_mode": "push-to-talk",
    "wake_word": "omnipanel-go",
    "wake_word_listen_sec": 8,
    "stt_engine": "vosk",
    "tts_enabled": true,
    "speech_allowlist": []
  }
}
```

| Einstellung | Optionen | Beschreibung |
|---------|---------|-------------|
| `enabled` | `true` oder `false` | Spracherkennung ein- oder ausschalten |
| `recording_location` | `"client"` oder `"host"` | Wo Audio aufgenommen wird: `"client"` nutzt das Mikrofon deines Tablets, `"host"` nutzt das Mikrofon deines PCs |
| `trigger_mode` | `"push-to-talk"` oder `"wake-word"` | Wie die Spracherkennung aktiviert wird |
| `wake_word` | Beliebiges Wort | Das Wort, das den Wake-Word-Modus aktiviert (Standard: „omnipanel-go") |
| `wake_word_listen_sec` | Zahl | Sekunden, die nach Wake-Word-Erkennung zugehört wird (nur Host-Modus, Standard: 8) |
| `stt_engine` | `"vosk"` oder `"llama-cpp"` | Welche Spracherkennungs-Engine genutzt werden soll |
| `vosk_model_path` | Dateipfad | Pfad zum Vosk-Modell (leer lassen für automatisches Herunterladen) |
| `llama_cpp_url` | URL | Adresse deines llama-cpp-Servers |
| `llama_cpp_api_key` | Text | API-Key für llama-cpp-server (falls erforderlich) |
| `llama_cpp_api_mode` | `"transcriptions"` oder `"chat"` | API-Modus für llama-cpp-server |
| `llama_cpp_model` | Text | Modellname für den Chat-Modus (z. B. „gemma-3-4b") |
| `llama_cpp_prompt` | Text | Anweisungen für die KI im Chat-Modus |
| `tts_enabled` | `true` oder `false` | Sprachbestätigungen aktivieren (Text-zu-Sprache) |
| `speech_allowlist` | Liste von Mustern | Erlaubte Sprachbefehle (leer = alle erlaubt) |

### Schritt 2: Sprachbefehle definieren

Du kannst Sprachbefehle auf zwei Arten definieren:

**Methode 1: Im Editor (pro Block)**

1. Öffne den Editor
2. Klicke auf das Zahnrad-Symbol eines beliebigen Blocks
3. Finde die **Speech Trigger**-Einstellungen
4. Setze:
   - **Speech Trigger** — der Satz, der diesen Block aktiviert (z. B. „gear up")
   - **Aliases** — alternative Sätze (z. B. „landing gear up", „gear up please")
   - **Trigger Type** — was der Befehl macht (Button-Druck, Slider-Änderung, Shell-Befehl, HTTP-Anfrage)

**Methode 2: In speech_commands.json**

Bearbeite die Datei `user/speech_commands.json`:

```json
[
  {
    "phrase": "gear up",
    "aliases": ["landing gear up", "gear up please"],
    "type": "button",
    "joystick": 0,
    "button": 3
  },
  {
    "phrase": "set throttle to *",
    "aliases": ["throttle to *"],
    "type": "slider",
    "joystick": 0,
    "slider": 0
  },
  {
    "phrase": "launch notepad",
    "aliases": ["open notepad"],
    "type": "shell",
    "command": "notepad"
  }
]
```

---

## Befehlstypen

### Button-Druck

Löst einen Button-Block auf deinem Panel aus. Sprachbefehle vom Typ „button"
werden direkt auf dem Server ausgeführt — du brauchst kein geöffnetes Panel
im Browser. Die Button-ID wird automatisch aus den Panel-Dateien übernommen.

**Einstellungen:**
- `type`: `"button"`
- `joystick`: Welcher virtueller Controller (0-9)
- `button`: Welcher Button (0-15)

**Beispiel:**
```json
{
  "phrase": "gear up",
  "type": "button",
  "joystick": 0,
  "button": 3
}
```

### Slider-Änderung

Setzt einen Slider auf einen gesprochenen Wert. Auch Slider-Befehle werden
direkt auf dem Server ausgeführt — kein Browser-Panel nötig.

**Einstellungen:**
- `type`: `"slider"`
- `joystick`: Welcher virtueller Controller (0-9)
- `slider`: Welche Achse (0-7)

**Beispiel:**
```json
{
  "phrase": "set throttle to *",
  "type": "slider",
  "joystick": 0,
  "slider": 0
}
```

Sage „set throttle to 75" und der Slider bewegt sich auf 75 (von 255).

### Shell-Befehl

Führt einen Befehl auf deinem PC aus.

**Einstellungen:**
- `type`: `"shell"`
- `command`: Der auszuführende Befehl

**Beispiel:**
```json
{
  "phrase": "launch notepad",
  "type": "shell",
  "command": "notepad"
}
```

### HTTP-Anfrage

Sendet eine Webanfrage.

**Einstellungen:**
- `type`: `"http"`
- `method`: GET, POST, PUT, DELETE oder PATCH
- `url`: Die URL, an die die Anfrage gesendet wird
- `body`: Optionaler Anfrage-Body

**Beispiel:**
```json
{
  "phrase": "turn on living room light",
  "type": "http",
  "method": "POST",
  "url": "http://192.168.1.100/api/lights/1",
  "body": "{\"on\": true}"
}
```

---

## Parameter-Extraktion

Sprachbefehle können Werte aus deinen gesprochenen Sätzen mit Wildcards (`*`) extrahieren.

### So funktioniert es

Definiere einen Satz mit `*` dort, wo der Wert sein soll:

```json
{
  "phrase": "set throttle to *",
  "type": "slider",
  "joystick": 0,
  "slider": 0
}
```

Wenn du „set throttle to 75" sagst, extrahiert OmniPanel-go `75` und setzt den Slider auf diesen Wert.

### Mehrere Parameter

Du kannst mehrere Wildcards nutzen:

```json
{
  "phrase": "set * to *",
  "type": "slider",
  "joystick": 0,
  "slider": 0
}
```

Das passt auf Sätze wie „set throttle to 75" oder „set volume to 50".

---

## Aliase

Aliase lassen dich denselben Befehl mit verschiedenen Sätzen auslösen.

```json
{
  "phrase": "gear up",
  "aliases": ["landing gear up", "gear up please", "deploy gear"],
  "type": "button",
  "joystick": 0,
  "button": 3
}
```

All diese Sätze lösen denselben Button-Druck aus:
- „gear up"
- „landing gear up"
- „gear up please"
- „deploy gear"

---

## Fuzzy Matching

OmniPanel-go nutzt Fuzzy Matching, um dich zu verstehen, auch wenn die Spracherkennung nicht perfekt ist. Es vergleicht, was es gehört hat, mit deinen definierten Befehlen mit einer **80%-Ähnlichkeitsschwelle**.

**Beispiel:**
- Du hast definiert: „gear up"
- Du hast gesagt: „gear upp" (leicht falsch erkannt)
- Ergebnis: Passt trotzdem, weil es nah genug dran ist

---

## Sicherheits-Allowlist

Die Speech-Allowlist schränkt ein, welche Sätze Befehle ausführen dürfen. Das verhindert versehentliche oder bösartige Sprachbefehle.

### So funktioniert es

Setze in `config.json` die `speech_allowlist` auf eine Liste erlaubter Muster:

```json
{
  "speech_allowlist": [
    "gear up",
    "gear down",
    "set * to *",
    "launch *"
  ]
}
```

- **Exakte Übereinstimmung:** `"gear up"` — nur „gear up" ist erlaubt
- **Wildcard:** `"set * to *"` — jeder Satz, der auf dieses Muster passt, ist erlaubt
- **Leere Liste:** Alle Befehle sind erlaubt (keine Einschränkungen)

Wenn ein gesprochener Befehl auf kein Muster in der Allowlist passt, wird er ignoriert.

---

## Text-zu-Sprache-Bestätigungen

Wenn `tts_enabled` auf `true` gesetzt ist, spricht OmniPanel-go eine Bestätigung, nachdem es einen Befehl ausgeführt hat.

**Beispiel:**
- Du sagst: „gear up"
- OmniPanel-go führt den Befehl aus
- OmniPanel-go sagt: „Gear up confirmed"

Das gibt dir Audio-Feedback, dass dein Befehl verstanden und ausgeführt wurde.

---

## Visuelles Feedback

### Mikrofon-Button

Der schwebende Mikrofon-Button zeigt verschiedene Zustände:

- **Normal** — bereit zum Aufnehmen
- **Pulsierend rot** — nimmt deine Stimme auf
- **Pulsierend gelb** — verarbeitet, was du gesagt hast

### Toast-Benachrichtigung

Eine Nachricht erscheint, die zeigt, was OmniPanel-go gehört hat:

- Zeigt den erkannten Text
- Zeigt, welcher Befehl gepasst hat
- Verschwindet automatisch nach ein paar Sekunden

### Leuchten des passenden Blocks

Wenn ein Sprachbefehl auf einen Block auf deinem Panel passt, **leuchtet dieser Block kurz cyan**, damit du sehen kannst, welches Steuerelement aktiviert wurde.

---

## Praktische Beispiele

### Beispiel 1: Sprachgesteuertes Flug-Panel

**Befehle:**
```json
[
  {"phrase": "gear up", "aliases": ["landing gear up"], "type": "button", "joystick": 0, "button": 0},
  {"phrase": "gear down", "aliases": ["landing gear down"], "type": "button", "joystick": 0, "button": 1},
  {"phrase": "set throttle to *", "aliases": ["throttle to *"], "type": "slider", "joystick": 0, "slider": 0},
  {"phrase": "flaps up", "type": "button", "joystick": 0, "button": 2},
  {"phrase": "flaps down", "type": "button", "joystick": 0, "button": 3}
]
```

### Beispiel 2: Sprachgesteuertes Smart Home

**Befehle:**
```json
[
  {"phrase": "turn on living room light", "aliases": ["lights on"], "type": "http", "method": "POST", "url": "http://192.168.1.100/api/lights/1", "body": "{\"on\": true}"},
  {"phrase": "turn off living room light", "aliases": ["lights off"], "type": "http", "method": "POST", "url": "http://192.168.1.100/api/lights/1", "body": "{\"on\": false}"},
  {"phrase": "set temperature to *", "type": "http", "method": "POST", "url": "http://192.168.1.100/api/thermostat", "body": "{\"temp\": *}"},
  {"phrase": "launch spotify", "type": "shell", "command": "spotify"}
]
```

---

## Tipps für Sprachbefehle

### Server-seitige Ausführung

Button- und Slider-Sprachbefehle werden direkt auf deinem PC über das virtuelle
Joystick-System ausgeführt. Das bedeutet:
- **Kein Browser-Panel nötig** — Befehle funktionieren auch ohne geöffnetes Panel
- **Keine Doppel-Ausführung** — der Server führt den Befehl aus, der Browser zeigt nur einen visuellen Blitz
- **Automatische ID-Zuordnung** — Button-/Slider-IDs werden beim Start aus deinen Panel-Dateien gelesen

### Klar sprechen

- Sprich in normalem Tempo — hetze nicht
- Artikuliere klar, besonders bei wichtigen Befehlen
- Reduziere Hintergrundgeräusche für bessere Erkennung

## Auslösemodi

Diese Modi sind gegenseitig exklusiv — setze einen in `config.json`.

### Push-to-Talk (`"push-to-talk"`)

Halte den schwebenden Mikrofon-Button auf dem Panel (oder einen dedizierten Push-to-Talk-Block), sprich deinen Befehl und lasse los, um ihn zu verarbeiten.

- Visuelles Feedback: roter Puls während der Aufnahme, gelber Puls während der Verarbeitung
- Funktioniert sowohl mit Client- als auch Host-Aufnahme

### Wake Word (`"wake-word"`)

Kontinuierliches freihändiges Zuhören. Das Verhalten hängt von `recording_location` ab:

**Host-Modus** (`"recording_location": "host"`):
- Der Server hört kontinuierlich nach dem Wake Word über das PC-Mikrofon
- Wenn das Wake Word erkannt wird, nimmt der Server `wake_word_listen_sec` Sekunden Folgesprache auf und verarbeitet sie
- Funktioniert in allen Browsern — keine browserspezifischen Anforderungen
- Der schwebende Mikrofon-Button wird ausgeblendet, da der Server alles verarbeitet

**Client-Modus** (`"recording_location": "client"`):
- Der Browser hört nach dem Wake Word über die Web Speech API (nur Chrome/Edge)
- Bei Erkennung nimmt das Tablet-Mikrofon Folgesprache auf
- Visuelles Feedback: cyaner Puls während des Zuhörens, schnellerer Puls bei Wake-Word-Erkennung

### Ein Wake Word wählen

- Wähle ein Wort, das leicht zu sagen ist und im Gespräch unwahrscheinlich vorkommt
- Vermeide häufige Wörter wie „hey" oder „ok"
- Zweisilbige Wörter funktionieren am besten (z. B. „omnipanel-go", „computer")

### Spracherkennung testen

1. Beginne mit einfachen, einzigartigen Sätzen
2. Teste jeden Befehl einzeln
3. Prüfe die Toast-Benachrichtigung, um zu sehen, was OmniPanel-go gehört hat
4. Passe deine Sätze oder Aliase an, wenn die Erkennung schlecht ist

### Browser-Berechtigungen

**Client-Aufnahme** erfordert Mikrofon-Berechtigung von deinem Browser. Wenn du Spracherkennung zum ersten Mal nutzt, fragt dein Browser nach Mikrofon-Berechtigung. Klicke auf **Zulassen**.

Falls du die Berechtigung versehentlich verweigert hast:
- **Chrome/Edge:** Klicke auf das Schloss-Symbol in der Adressleiste, dann aktiviere Mikrofon
- **Firefox:** Klicke auf das Kamera-/Mikrofon-Symbol in der Adressleiste, dann aktiviere

> **Hinweis:** Host-Aufnahme erfordert KEINE Browser-Mikrofon-Berechtigung. Das Mikrofon deines PCs wird direkt genutzt, der Browser braucht also keinen Zugriff auf ein Mikrofon.

### Aufnahmeort

- **Client-Aufnahme** (`"recording_location": "client"`) — nutzt das Mikrofon deines Tablets. Der Browser nimmt Audio auf und sendet es an deinen PC über das Netzwerk. Empfohlen für Push-to-Talk, wenn dein Tablet nah an dir ist.
- **Host-Aufnahme** (`"recording_location": "host"`) — nutzt das Mikrofon deines PCs. Dein PC nimmt Audio direkt von seinem eigenen Mikrofon auf. Dein Tablet sendet nur Start/Stop-Signale. Empfohlen für Wake-Word-Modus oder wenn dein PC ein besseres Mikrofon hat.

**So wechselst du:**
1. Öffne `config.json` auf deinem PC
2. Ändere `"recording_location"` zu `"client"` oder `"host"`
3. Starte OmniPanel-go neu

## Was kommt als Nächstes?

Verstehe, wie Spiele deine Panel-Eingaben sehen in [Virtuelle Joysticks](12-virtual-joysticks.md).
