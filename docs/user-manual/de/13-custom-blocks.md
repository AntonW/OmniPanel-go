# Kapitel 13: Eigene Blöcke

Du magst den Look der eingebauten Blöcke nicht? Möchtest du ein Steuerelement erstellen, das es noch nicht gibt? Dieses Kapitel zeigt dir, wie du deine eigenen Blöcke erstellst.

## Was sind eigene Blöcke?

Eigene Blöcke sind HTML-Dateien, die du erstellst und die neue UI-Elemente für deine Panels definieren. Sie funktionieren genau wie die eingebauten Blöcke — du ziehst sie aus der Bibliothek, konfigurierst ihre Einstellungen und platzierst sie auf deinem Panel.

## Wo eigene Blöcke hin kommen

Eigene Blöcke gehören in den `user/blocks/`-Ordner:

```
OmniPanel-go/
├── user/
│   └── blocks/
│       ├── my-button.html
│       ├── my-slider.html
│       └── my-display.html
```

Du kannst sie in Unterordner organisieren:

```
OmniPanel-go/
├── user/
│   └── blocks/
│       ├── my-theme/
│       │   ├── button.html
│       │   └── slider.html
│       └── custom/
│           └── special-control.html
```

## Einen einfachen eigenen Block erstellen

### Schritt 1: Eine HTML-Datei erstellen

Erstelle eine neue Datei in `user/blocks/` mit der Endung `.html`. Nennen wir sie `my-button.html`.

### Schritt 2: Das HTML schreiben

Ein eigener Block ist einfach eine HTML-Datei mit speziellen Platzhaltern für Einstellungen:

```html
<div class="my-button" style="
  background-color: settings-backgroundcolor;
  color: settings-textcolor;
  font-size: settings-fontsizepx;
  border-radius: settings-borderradiuspx;
  padding: 10px 20px;
  text-align: center;
  cursor: pointer;
  user-select: none;
">
  settings-label
</div>
```

### Schritt 3: Einstellungen definieren

Füge am Anfang deiner HTML-Datei ein `<settings>`-Tag hinzu, das die konfigurierbaren Optionen auflistet:

```html
<settings>
  label:My Button:text
  backgroundcolor:#333333:color
  textcolor:#ffffff:color
  fontsize:16:number
  borderradius:8:number
</settings>

<div class="my-button" style="
  background-color: settings-backgroundcolor;
  color: settings-textcolor;
  font-size: settings-fontsizepx;
  border-radius: settings-borderradiuspx;
  padding: 10px 20px;
  text-align: center;
  cursor: pointer;
  user-select: none;
">
  settings-label
</div>
```

### Schritt 4: OmniPanel-go neu starten

Starte OmniPanel-go neu (oder aktualisiere den Editor) und dein eigener Block erscheint in der Block-Bibliothek.

---

## So funktionieren Einstellungen

### Das `<settings>`-Tag

Das `<settings>`-Tag definiert, welche Optionen im Zahnrad-Symbol-Einstellungsfenster erscheinen. Jede Zeile folgt diesem Format:

```
name:standardwert:typ
```

| Teil | Beschreibung | Beispiel |
|------|-------------|---------|
| `name` | Der Einstellungsname (in Platzhaltern verwendet) | `label`, `backgroundcolor` |
| `standardwert` | Der Startwert | `My Button`, `#333333` |
| `typ` | Der Eingabetyp | `text`, `color`, `number` |

### Einstellungstypen

| Typ | Was er zeigt | Beispiel |
|------|--------------|---------|
| `text` | Texteingabefeld | `label:My Button:text` |
| `color` | Farbauswahl | `backgroundcolor:#333333:color` |
| `number` | Zahleneingabe | `fontsize:16:number` |

### Einstellungen in HTML verwenden

Nutze `settings-name`-Platzhalter in deinem HTML. OmniPanel-go ersetzt diese durch die tatsächlichen Werte beim Rendern des Blocks:

```html
<!-- Dieser Platzhalter... -->
settings-label

<!-- ...wird daraus, wenn das Label „Fire" ist: -->
Fire

<!-- Dieser Platzhalter... -->
settings-backgroundcolor

<!-- ...wird daraus, wenn die Farbe „#FF0000" ist: -->
#FF0000
```

---

## JavaScript zu eigenen Blöcken hinzufügen

Eigene Blöcke können JavaScript enthalten, um Interaktionen wie Button-Drücke, Slider-Bewegungen und Datenaktualisierungen zu verarbeiten.

### Beispiel: Interaktiver Button

```html
<settings>
  label:Fire:text
  backgroundcolor:#cc0000:color
  textcolor:#ffffff:color
  fontsize:18:number
  joystick:0:number
  button:0:number
</settings>

<div class="my-button"
     data-joystick="settings-joystick"
     data-button="settings-button"
     style="
  background-color: settings-backgroundcolor;
  color: settings-textcolor;
  font-size: settings-fontsizepx;
  padding: 10px 20px;
  text-align: center;
  cursor: pointer;
  user-select: none;
  border-radius: 8px;
">
  settings-label
</div>

<script>
  const button = document.currentScript.previousElementSibling;

  button.addEventListener('mousedown', () => {
    const joystick = parseInt(button.dataset.joystick);
    const btn = parseInt(button.dataset.button);
    // Sende Button-Druck an OmniPanel-go
    window.parent.postMessage({
      type: 'joystick-button',
      joystick: joystick,
      button: btn,
      pressed: true
    }, '*');
  });

  button.addEventListener('mouseup', () => {
    const joystick = parseInt(button.dataset.joystick);
    const btn = parseInt(button.dataset.button);
    // Sende Button-Loslassen an OmniPanel-go
    window.parent.postMessage({
      type: 'joystick-button',
      joystick: joystick,
      button: btn,
      pressed: false
    }, '*');
  });

  // Touch-Unterstützung für mobile Geräte
  button.addEventListener('touchstart', (e) => {
    e.preventDefault();
    const joystick = parseInt(button.dataset.joystick);
    const btn = parseInt(button.dataset.button);
    window.parent.postMessage({
      type: 'joystick-button',
      joystick: joystick,
      button: btn,
      pressed: true
    }, '*');
  });

  button.addEventListener('touchend', (e) => {
    e.preventDefault();
    const joystick = parseInt(button.dataset.joystick);
    const btn = parseInt(button.dataset.button);
    window.parent.postMessage({
      type: 'joystick-button',
      joystick: joystick,
      button: btn,
      pressed: false
    }, '*');
  });
</script>
```

---

## CSS zu eigenen Blöcken hinzufügen

Du kannst CSS einbinden, um deine Blöcke zu gestalten. Nutze ein `<style>`-Tag:

```html
<settings>
  label:My Button:text
  backgroundcolor:#333333:color
  textcolor:#ffffff:color
</settings>

<style>
  .my-button {
    transition: all 0.2s ease;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
  }

  .my-button:hover {
    transform: scale(1.05);
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.4);
  }

  .my-button:active {
    transform: scale(0.95);
  }
</style>

<div class="my-button" style="
  background-color: settings-backgroundcolor;
  color: settings-textcolor;
  padding: 10px 20px;
  text-align: center;
  cursor: pointer;
  user-select: none;
  border-radius: 8px;
">
  settings-label
</div>
```

---

## Fortgeschritten: Daten abonnieren

Eigene Blöcke können Live-Daten anzeigen, indem sie den Datenbus abonnieren.

### Beispiel: Data Display-Block

```html
<settings>
  title:CPU:text
  datakey:cpu_usage:text
  unit:%:text
  decimals:1:number
  textcolor:#ffffff:color
  backgroundcolor:#1a1a1a:color
</settings>

<div class="data-display" style="
  background-color: settings-backgroundcolor;
  color: settings-textcolor;
  padding: 15px;
  border-radius: 8px;
  text-align: center;
">
  <div class="title">settings-title</div>
  <div class="value" id="value">--</div>
  <div class="unit">settings-unit</div>
</div>

<script>
  const display = document.currentScript.previousElementSibling;
  const valueEl = display.querySelector('#value');
  const dataKey = 'settings-datakey';
  const decimals = parseInt('settings-decimals');

  // Datenaktualisierungen abonnieren
  window.parent.postMessage({
    type: 'subscribe',
    key: dataKey
  }, '*');

  // Auf Datenaktualisierungen hören
  window.addEventListener('message', (e) => {
    if (e.data.type === 'data-update' && e.data.key === dataKey) {
      const value = parseFloat(e.data.value).toFixed(decimals);
      valueEl.textContent = value;
    }
  });
</script>
```

---

## Tipps für eigene Blöcke

### Einfach anfangen

Beginne mit einem einfachen statischen Block (nur HTML und CSS), bevor du JavaScript-Interaktionen hinzufügst.

### Eindeutige Klassennamen verwenden

Versieh deine CSS-Klassennamen mit einem Präfix, um Konflikte mit anderen Blöcken zu vermeiden:

```css
/* Gut */
.my-theme-button { }

/* Schlecht — könnte mit anderen Blöcken kollidieren */
.button { }
```

### Im Editor testen

Nach dem Erstellen eines eigenen Blocks:

1. Starte OmniPanel-go neu
2. Öffne den Editor
3. Ziehe deinen Block auf die Arbeitsfläche
4. Teste alle Einstellungen im Zahnrad-Symbol-Fenster
5. Öffne das Panel auf deinem Tablet, um Interaktionen zu testen

### Dateibenennung

Nutze beschreibende Namen für deine Block-Dateien:

- Gut: `sci-fi-button.html`, `volume-slider.html`
- Weniger ideal: `block1.html`, `test.html`

### Eigene Blöcke sichern

Deine eigenen Blöcke sind im `user/blocks/`-Ordner. Sichere diesen Ordner, wenn du deine eigenen Blöcke beim Aktualisieren von OmniPanel-go erhalten möchtest.

---

## Beispiel: Toggle Switch-Block

Hier ist ein vollständiger eigener Kippschalter mit der Standard-`.toggle-track`-Struktur:

```html
<settings>
  label:Autopilot:text
  joystick:0:number
  button:5:number
  toggle_color:#666666:color
  toggle_color_active:#00cc00:color
  label_color:#ffffff:color
  border_color:#00cc00:color
</settings>

<style>
  .toggle-switch {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5cqmin;
    cursor: pointer;
    user-select: none;
  }

  .toggle-track {
    height: var(--height);
    aspect-ratio: 2 / 1;
    border-radius: 999px;
    position: relative;
    transition: background-color 0.2s;
    background: var(--toggle-color);
    border: 1px solid var(--border-color);
  }

  .toggle-track::after {
    content: '';
    position: absolute;
    top: 50%;
    left: 3px;
    transform: translateY(-50%);
    width: 40%;
    height: 80%;
    border-radius: 50%;
    background: #ffffff;
    transition: left 0.2s;
  }

  .toggle-track.active {
    background-color: var(--toggle-color-active);
  }

  .toggle-track.active::after {
    left: calc(100% - 80% - 3px);
  }
</style>

<div class="toggle-switch" style="--toggle-color: settings-toggle_color; --toggle-color-active: settings-toggle_color_active; --border-color: settings-border_color;">
  <p style="color: settings-label_color;">settings-label</p>
  <div class="toggle-track" virtual-joystick="settings-joystick" emulate-button="settings-button"></div>
</div>
```

### Wichtige Punkte für Toggle-Blöcke

- Verwende ein `<div class="toggle-track">`-Element (kein `<button>`) mit `virtual-joystick`- und `emulate-button`-Attributen
- Die Spur muss `aspect-ratio: 2 / 1` haben, um ihre Pillenform beim Skalieren beizubehalten
- Der Knopf wird über das `::after`-Pseudo-Element mit `width: 40%` und `height: 80%` erstellt — diese Mathematik stellt einen perfekten Kreis sicher, weil 40% der 2x-breiten Spur gleich 80% der 1x-hohen Spur sind
- Zentriere den Knopf vertikal mit `top: 50%` und `transform: translateY(-50%)`
- Schiebe den Knopf im aktiven Zustand nach rechts mit `left: calc(100% - 80% - 3px)`
- Die `active`-Klasse wird vom Framework umgeschaltet, wenn sich der virtuelle Button-Zustand ändert

## Beispiel: Sequenz-Button-Block

Hier ist ein vollständiger Sequenz-Button, der WASD-Tasten sendet, während Strg gedrückt gehalten wird:

```html
<settings>
label:Stratagem:text
icon_url::text
layout:image-text:select:image-only,text-only,image-text
text_position:bottom:select:top,bottom,left,right
sequence_code:d,d,w,s,a:text
key_delay:100:number
keyboard_index:0:number
font_size:14:number
button_color:#1a1a1a:color
button_color_active:#ffcc00:color
label_color:#ffffff:color
border_color:#ffcc00:color
</settings>

<style>
  .sequence-button {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    user-select: none;
    background: var(--button-color);
    color: var(--label-color);
    font-size: var(--font-size);
    border: 2px solid var(--border-color);
    clip-path: polygon(10px 0, 100% 0, 100% calc(100% - 10px), calc(100% - 10px) 100%, 0 100%, 0 10px);
    transition: background-color 0.15s, transform 0.1s;
  }

  .sequence-button:active,
  .sequence-button.executing {
    background: var(--button-color-active);
    color: #000;
    transform: scale(0.97);
  }

  .sequence-button .icon {
    width: 32px;
    height: 32px;
    object-fit: contain;
  }

  .sequence-button.layout-image-text {
    flex-direction: var(--text-position, column);
  }

  .sequence-button.layout-image-text.text-position-top {
    flex-direction: column-reverse;
  }

  .sequence-button.layout-image-text.text-position-left {
    flex-direction: row-reverse;
  }

  .sequence-button.layout-image-text.text-position-right {
    flex-direction: row;
  }
</style>

<div class="sequence-button layout-settings-layout text-position-settings-text_position"
     style="--button-color: settings-button_color; --button-color-active: settings-button_color_active; --label-color: settings-label_color; --border-color: settings-border_color; --font-size: settings-font_sizepx;"
     data-icon-url="settings-icon_url"
     data-sequence-code="settings-sequence_code"
     data-key-delay="settings-key_delay"
     data-keyboard-index="settings-keyboard_index">
  <img class="icon" data-icon-url="settings-icon_url" alt="">
  <span class="label">settings-label</span>
</div>
```

### Wichtige Punkte für Sequenz-Button-Blöcke

- Verwende `data-sequence-code` zum Speichern der komma-getrennten WASD-Tasten (z.B. `d,d,w,s,a`)
- Verwende `data-key-delay` für die Pause zwischen Tastenanschlägen in Millisekunden
- Verwende `data-keyboard-index` zur Auswahl der virtuellen Tastatur
- Verwende `data-icon-url` statt `src`, um Browser-404s vor der JS-Initialisierung zu vermeiden
- Die `initSequenceButton`-Funktion in `client.js` verarbeitet Icon-Laden und Layout-Konfiguration
- Die `executeSequence`-Funktion hält Strg gedrückt, tippt jede Taste, lässt dann Strg los
- Füge die `executing`-Klasse während der Sequenzwiedergabe für visuelles Feedback hinzu

### Gemeinsame Button-Klassen für eigene Blöcke

Alle eingebauten Button-Blöcke (`button.html`, `command_block.html`, `sequence_button.html`)
verwenden gemeinsame CSS-Klassen für Icon-+ Text-Layouts. Du kannst diese auch in eigenen Blöcken nutzen:

```html
<button ...>
    <div class="btn-content layout-image-text text-bottom">
        <img class="btn-icon" data-icon-url="settings-icon_url" alt="" />
        <span class="btn-label">settings-label</span>
    </div>
</button>
```

Die `initButtonLayout()`-Funktion in `client.js` konfiguriert diese Klassen automatisch, wenn sie
`.btn-content` innerhalb eines Buttons erkennt. Unterstützte Layouts: `image-only`, `text-only`,
`image-text` mit Textpositionen: `top`, `bottom`, `left`, `right`.

## Was kommt als Nächstes?

Zum Abschluss: [Tipps und Fehlerbehebung](14-tips-and-troubleshooting.md) für Best Practices und häufige Lösungen.
