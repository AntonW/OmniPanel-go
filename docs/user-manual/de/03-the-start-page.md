# Kapitel 3: Die Startseite

Die Startseite ist dein Dashboard — das Erste, was du siehst, wenn du OmniPanel-go öffnest. Sie ist dein Kontrollzentrum zum Verwalten von Panels, Überwachen von Verbindungen und Anpassen von Einstellungen.

## So greifst du auf die Startseite zu

Öffne deinen Browser und gehe zu:
```
http://DEINE-PC-IP:3000
```

## Was du siehst

### Panel-Liste

Der Hauptbereich zeigt Karten für jedes gespeicherte Panel. Jede Karte zeigt:

- **Panel-Name** — der Name, den du beim Speichern vergeben hast
- **Klicke auf die Karte** — öffnet dieses Panel auf deinem Gerät

Wenn du noch keine Panels erstellt hast, ist dieser Bereich leer.

### Schnelllinks

Am oberen Rand der Seite findest du Verknüpfungen:

- **Panel Editor** — öffnet den visuellen Editor, in dem du Panels entwirfst
- **Neues Panel** — öffnet den Editor mit einer leeren Arbeitsfläche

### Verbindungs-URL

Dieser Bereich zeigt die Webadresse, die du auf deinem Tablet oder Handy eingeben solltest, um auf OmniPanel-go zuzugreifen. Sie sieht so aus:

```
http://192.168.1.100:3000
```

Du kannst diese Adresse kopieren und mit jedem in deinem Netzwerk teilen, der sich verbinden möchte.

### Vollbild-Steuerung

Buttons zum Umschalten des Vollbildmodus auf allen verbundenen Geräten:

- **Vollbild aktivieren** — macht alle verbundenen Tablets vollbild (versteckt Browser-Leisten)
- **Vollbild verlassen** — bringt alle Geräte zurück zur normalen Browser-Ansicht

> **Tipp:** Der Vollbildmodus ist super für Immersion — dein Panel sieht dann aus wie eine dedizierte App statt einer Webseite.

### Verbindungsprotokoll

Am unteren Rand der Seite siehst du ein Live-Protokoll, das Folgendes anzeigt:

- **Wann Geräte sich verbinden** — mit Zeitstempel und IP-Adresse
- **Wann Geräte sich trennen** — mit Zeitstempel und IP-Adresse
- **Wann der Host-Agent sich verbindet oder trennt** (im verteilten/Relay-Modus) — zeigt die IP-Adresse des Hosts

Das ist nützlich für:

- Prüfen, ob dein Tablet erfolgreich verbunden ist
- Sehen, welche Host-Maschine verbunden ist (im Relay-Modus)
- Sehen, wie viele Geräte gerade verbunden sind
- Fehlerbehebung bei Verbindungsproblemen

Beispiel-Protokolleintrag:
```
[14:32:15] Gerät verbunden von 192.168.1.50
[14:33:00] Host verbunden: 10.0.0.25
[14:35:42] Gerät getrennt von 192.168.1.50
```

### Host-IP-Anzeige (Panel-Ansicht)

Wenn du ein Panel auf deinem Tablet oder Handy im verteilten (Relay-) Modus öffnest, erscheint eine kleine Anzeige in der **oberen linken Ecke**, die die IP-Adresse des verbundenen Hosts zeigt:

```
Host: 10.0.0.25
```

So kannst du schnell überprüfen, mit welcher Maschine dein Panel kommuniziert. Die Anzeige verschwindet, wenn der Host sich trennt, und erscheint wieder, wenn ein neuer Host sich verbindet.

### Theme-Umschalter

In der oberen rechten Ecke findest du einen Button zum Wechseln zwischen:

- **Dunkles Theme** — dunklere Farben, angenehmer für die Augen in abgedunkelten Räumen
- **Helles Theme** — hellere Farben, besser für gut beleuchtete Umgebungen

Deine Einstellung wird automatisch gespeichert, sodass OmniPanel-go sich deine Wahl beim nächsten Öffnen merkt.

## Die Startseite nutzen

### Ein Panel auf deinem Tablet öffnen

1. Öffne die Startseite auf deinem Tablet
2. Tippe auf die Panel-Karte, die du nutzen möchtest
3. Das Panel öffnet sich mit all deinen Buttons, Schiebereglern und Steuerelementen

### Ein Panel auf deinem PC öffnen

Du kannst Panels auch im PC-Browser zum Testen öffnen:

1. Klicke auf der Startseite auf eine Panel-Karte
2. Das Panel öffnet sich in deinem Browser
3. Du kannst Buttons mit der Maus anklicken, um sie zu testen

### Mehrere Geräte verwalten

Du kannst gleichzeitig verschiedene Panels auf verschiedenen Geräten geöffnet haben:

- Tablet 1: Flugsteuerungs-Panel
- Tablet 2: Navigations- und Systempanel
- Handy: Schnellzugriffs-Panel für Notfunktionen

Jedes Gerät öffnet sein Panel unabhängig voneinander.

## Was kommt als Nächstes?

Bereit, dein erstes Panel zu bauen? Weiter zu [Kapitel 4: Dein erstes Panel erstellen](04-creating-your-first-panel.md), um zu lernen, wie du den visuellen Editor nutzt.
