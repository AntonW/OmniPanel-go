# Kapitel 16: RSS-Feeds

Möchtest du die neuesten Nachrichten, Blogbeiträge oder Podcast-Folgen auf deinem Panel sehen? Dieses Kapitel behandelt den RSS-Feed-Block und wie du einen oder mehrere Feeds zu deinem Panel hinzufügst.

## Was ist ein RSS-Feed?

RSS (Really Simple Syndication) ist eine Möglichkeit für Websites, ihre neuesten Inhalte in einem Standardformat zu teilen. Viele Nachrichtenseiten, Blogs, YouTube-Kanäle und Podcasts bieten RSS-Feeds an. Anstatt jede Website einzeln zu besuchen, um nach neuen Inhalten zu suchen, sammelt ein RSS-Reader alle Updates an einem Ort.

Der RSS-Feed-Block in OmniPanel-go funktioniert als RSS-Reader — er lädt deine gewählten Feeds herunter und zeigt die neuesten Einträge direkt auf deinem Panel an.

### Was er anzeigt

- **Eintragstitel** — die Überschrift oder der Name des Artikels/Videos/Podcasts
- **Veröffentlichungsdatum** — wann der Eintrag veröffentlicht wurde
- **Feed-Label** — ein benutzerdefinierter Name für jeden Feed (z. B. "Tech-News", "Mein Blog")
- **Beschreibung** — eine kurze Zusammenfassung des Eintrags (optional)
- **Klicken zum Öffnen** — tippe auf einen Eintrag, um ihn im Browser deines Host-PCs zu öffnen

---

## Einen RSS-Feed-Block hinzufügen

### Schritt 1: Feed-URLs finden

Zuerst brauchst du die RSS/Atom-Feed-URLs, die du anzeigen möchtest. So findest du sie am häufigsten:

- **Nachrichtenseiten**: Suche nach einem RSS-Symbol (📡) oder einem "Abonnieren"-Link auf der Website
- **YouTube**: `https://www.youtube.com/feeds/videos.xml?channel_id=KANAL_ID`
- **Blogs**: Oft unter `/feed`, `/rss` oder `/atom.xml`
- **Podcasts**: Normalerweise auf der Website des Podcasts verlinkt

### Schritt 2: Block hinzufügen

1. Öffne den Editor
2. Finde die Kategorie **Inhalt** in der Blockbibliothek (📰-Symbol)
3. Ziehe den **RSS-Feed**-Block auf deine Arbeitsfläche
4. Klicke auf das Zahnrad-Symbol, um die Einstellungen zu öffnen

### Schritt 3: Deine Feeds konfigurieren

Im Textbereich **Feed-URLs** gibst du eine Feed-URL pro Zeile ein. Optional kannst du ein Anzeige-Label nach einem `|`-Zeichen hinzufügen:

```
https://beispiel.de/news/rss
https://beispiel.de/tech/rss|Tech-News
https://www.youtube.com/feeds/videos.xml?channel_id=ABC123|Mein YouTube-Kanal
```

- Zeilen ohne `|` verwenden den eigenen Titel des Feeds als Label
- Zeilen mit `|Label` zeigen dein benutzerdefiniertes Label an

### Schritt 4: Einstellungen anpassen

| Einstellung | Was sie bewirkt | Standardwert |
|-------------|----------------|--------------|
| **Aktualisierungsintervall** | Wie oft nach neuen Feeds gesucht wird (Sekunden, min. 10) | 60 |
| **Max. Einträge** | Maximale Anzahl angezeigter Einträge (1-100) | 20 |
| **Datum anzeigen** | Veröffentlichungsdatum ein- oder ausblenden | An |
| **Feed-Label anzeigen** | Anzeige-Label des Feeds ein- oder ausblenden | An |
| **Beschreibung anzeigen** | Eintragszusammenfassung ein- oder ausblenden | An |
| **Max. Beschreibungslänge** | Maximale Zeichen für die Beschreibung (20-500) | 150 |
| **Farbe für neue Einträge** | Hervorhebungsfarbe für neue Einträge | Cyan (#1ccad8ff) |
| **Feed-Label-Farbe** | Farbe für den Feed-Label-Text | Blau (#3daee9ff) |

### Schritt 5: Speichern und testen

1. Speichere das Panel
2. Öffne dein Panel in einem Browser
3. Warte ein paar Sekunden — die neuesten Einträge sollten erscheinen
4. Neue Einträge werden in der konfigurierten Farbe hervorgehoben

---

## So funktioniert die Hervorhebung neuer Einträge

Jedes Gerät, das dein Panel öffnet, verfolgt unabhängig, welche Einträge es bereits gesehen hat. Das bedeutet:

- **Dein Tablet** könnte 5 neue hervorgehobene Einträge sehen
- **Dein Handy** könnte 8 neue hervorgehobene Einträge sehen (wenn es das Panel später geöffnet hat)
- Wenn neuere Einträge eintreffen, verblassen die alten Hervorhebungen — nur die neuesten ungesehenen Einträge bleiben hervorgehoben

Dies ist eine gerätespezifische Verfolgung, sodass du auf jedem Bildschirm siehst, was neu ist.

---

## Einträge anklicken, um URLs zu öffnen

Wenn du auf deinem Panel auf einen RSS-Eintrag tippst oder klickst, sendet OmniPanel-go einen Befehl an deinen Host-PC, um die URL in deinem Standardbrowser zu öffnen. Das bedeutet:

- Du musst dein Panel nicht verlassen
- Der Artikel öffnet sich auf deinem Gaming-PC, nicht im Browser deines Tablets
- Funktioniert mit jedem Browser, der als Systemstandard eingestellt ist

---

## Mehrere Feeds verwenden

Du kannst beliebig viele Feeds hinzufügen. Alle Einträge aus allen Feeds werden zusammengeführt und nach Datum sortiert (neueste zuerst). So erhältst du einen einheitlichen News-Stream aus allen deinen Quellen.

### Beispiel: News-Dashboard

```
https://feeds.bbci.co.uk/news/rss.xml|BBC News
https://rss.nytimes.com/services/xml/rss/nyt/HomePage.xml|NY Times
https://www.theverge.com/rss/index.xml|The Verge
https://hnrss.org/frontpage|Hacker News
```

Dies würde die neuesten Geschichten aus allen vier Quellen in einem Block anzeigen, sortiert nach Veröffentlichungszeitpunkt.

---

## Fehlerbehebung

### Meldung "Keine Feeds konfiguriert"

- Stelle sicher, dass du mindestens eine gültige URL im Textbereich Feed-URLs eingegeben hast
- Jede URL sollte in einer eigenen Zeile stehen
- Überprüfe, ob die URLs gültige RSS/Atom-Feeds sind (keine normalen Webseiten)
- Wenn die Meldung beim ersten Laden des Panels kurz erscheint, warte einen Moment — der Feed verbindet sich mit dem Server und die Einträge erscheinen in Kürze

### Keine Einträge sichtbar

- Der Feed könnte vorübergehend nicht verfügbar sein — überprüfe die URL in deinem Browser
- Einige Feeds erfordern eine Authentifizierung oder sind nicht öffentlich zugänglich
- Überprüfe die Server-Logs auf Abruffehler

### Einträge aktualisieren sich nicht

- Die Einstellung **Aktualisierungsintervall** steuert, wie oft Feeds überprüft werden (mindestens 10 Sekunden)
- Bei einem Wert von 300 (5 Minuten) musst du bis zu 5 Minuten auf neue Einträge warten
- Verringere das Aktualisierungsintervall für schnellere Updates

### Feed-Label zeigt den falschen Namen

- Wenn du kein `|Label` zur URL hinzugefügt hast, verwendet der Block den eigenen Titel des Feeds
- Einige Feeds haben generische Titel wie "RSS Feed" oder "Neueste Beiträge"
- Füge ein benutzerdefiniertes Label im Format `URL|Label` hinzu, um es zu überschreiben

---

## Tipps

- **Verwende benutzerdefinierte Labels**, um ähnliche Feeds zu unterscheiden (z. B. "BBC Tech" vs. "BBC Sport")
- **Stelle ein niedrigeres Aktualisierungsintervall** ein (z. B. 30 Sekunden) für zeitkritische Feeds wie Nachrichten
- **Stelle ein höheres Aktualisierungsintervall** ein (z. B. 300 Sekunden) für Feeds, die sich selten aktualisieren
- **Begrenze die max. Einträge**, um zu verhindern, dass der Block auf kleinen Bildschirmen zu lang wird
- **Schalte Beschreibungen aus**, wenn du nur Überschriften für eine kompakte Ansicht möchtest
