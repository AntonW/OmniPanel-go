# Chapter 16: RSS Feeds

Want to see the latest news, blog posts, or podcast episodes on your panel? This chapter covers the RSS Feed block and how to add one or more feeds to your panel.

## What Is an RSS Feed?

RSS (Really Simple Syndication) is a way for websites to share their latest content in a standard format. Many news sites, blogs, YouTube channels, and podcasts offer RSS feeds. Instead of visiting each website to check for new content, an RSS reader collects all the updates in one place.

The RSS Feed block in OmniPanel-go works as an RSS reader — it fetches your chosen feeds and displays the latest entries directly on your panel.

### What It Shows

- **Entry title** — the headline or name of the article/video/podcast
- **Publication date** — when the entry was published
- **Feed label** — a custom name for each feed (e.g., "Tech News", "My Blog")
- **Description** — a short summary of the entry (optional)
- **Click to open** — tap an entry to open it on your host PC or panel device

---

## Adding an RSS Feed Block

### Step 1: Find Feed URLs

First, you need the RSS/Atom feed URLs you want to display. Here are some common ways to find them:

- **News sites**: Look for an RSS icon (📡) or "Subscribe" link on the website
- **YouTube**: `https://www.youtube.com/feeds/videos.xml?channel_id=CHANNEL_ID`
- **Blogs**: Often at `/feed`, `/rss`, or `/atom.xml`
- **Podcasts**: Usually linked on the podcast's website

### Step 2: Add the Block

1. Open the Editor
2. Find the **Content** category in the block library (📰 icon)
3. Drag the **RSS Feed** block onto your workspace
4. Click the gear icon to open settings

### Step 3: Configure Your Feeds

In the **Feed URLs** text area, enter one feed URL per line. You can optionally add a display label after a `|` character:

```
https://example.com/news/rss
https://example.com/tech/rss|Tech News
https://www.youtube.com/feeds/videos.xml?channel_id=ABC123|My YouTube Channel
```

- Lines without a `|` will use the feed's own title as the label
- Lines with `|Label` will show your custom label instead

### Step 4: Adjust Settings

| Setting | What It Does | Default |
|---------|-------------|---------|
| **Refresh Interval** | How often to check for new feeds (seconds, min 10) | 60 |
| **Max Entries** | Maximum number of entries to show (1-100) | 20 |
| **Show Date** | Show or hide the publication date | On |
| **Show Feed Label** | Show or hide the feed's display label | On |
| **Show Description** | Show or hide the entry summary | On |
| **Description Max Length** | Maximum characters for the description (20-500) | 150 |
| **New Entry Color** | Highlight color for new entries | Cyan (#1ccad8ff) |
| **Feed Label Color** | Color for the feed label text | Blue (#3daee9ff) |
| **Open URL Location** | Where to open URLs — "host" (PC browser) or "client" (panel browser) | host |

### Step 5: Save and Test

1. Save the panel
2. Open your panel in a browser
3. Wait a few seconds — the latest entries should appear
4. New entries will be highlighted in the configured color

---

## How "New Entry" Highlighting Works

Each device that opens your panel tracks which entries it has seen independently. This means:

- **Your tablet** might see 5 new entries highlighted
- **Your phone** might see 8 new entries highlighted (if it opened the panel later)

The newest entry (most recent article/video) always stays highlighted as "new" — it won't lose the highlight when the feed refreshes. Only when an even newer entry arrives does the previous one lose its "new" status. This way, you always have a visible indicator of the latest content, and you won't miss anything between feed refreshes.

This is per-device tracking, so you always know what's new on each screen.

---

## Clicking Entries to Open URLs

When you tap or click an RSS entry on your panel, you can choose where the URL opens:

- **Host** (default): Sends a command to your host PC to open the URL in your default browser. The article opens on your gaming PC, not on your tablet's browser.
- **Client**: Opens the URL directly in the panel's browser (your tablet/phone). Useful when you want to read the article on the device you're holding.

Set this per-block using the **Open URL Location** setting in the block's properties.

---

## Using Multiple Feeds

You can add as many feeds as you want. All entries from all feeds are merged together and sorted by date (newest first). This gives you a unified news stream from all your sources.

### Example: News Dashboard

```
https://feeds.bbci.co.uk/news/rss.xml|BBC News
https://rss.nytimes.com/services/xml/rss/nyt/HomePage.xml|NY Times
https://www.theverge.com/rss/index.xml|The Verge
https://hnrss.org/frontpage|Hacker News
```

This would show the latest stories from all four sources in one block, sorted by publication time.

---

## Troubleshooting

### "No feeds configured" message

- Make sure you entered at least one valid URL in the Feed URLs text area
- Each URL should be on its own line
- Check that the URLs are valid RSS/Atom feeds (not regular web pages)
- If the message appears briefly when the panel first loads, wait a moment — the feed is connecting to the server and entries will appear shortly

### No entries showing

- The feed might be temporarily unavailable — check the URL in your browser
- Some feeds require authentication or are not publicly accessible
- Check the server logs for fetch errors

### Entries not updating

- The **Refresh Interval** setting controls how often feeds are checked (minimum 10 seconds)
- If set to 300 (5 minutes), you'll need to wait up to 5 minutes for new entries
- Try lowering the refresh interval for faster updates

### Feed label shows the wrong name

- If you didn't add a `|Label` to the URL, the block uses the feed's own title
- Some feeds have generic titles like "RSS Feed" or "Latest Posts"
- Add a custom label using the `URL|Label` format to override it

---

## Tips

- **Use custom labels** to distinguish between similar feeds (e.g., "BBC Tech" vs "BBC Sport")
- **Set a lower refresh interval** (like 30 seconds) for time-sensitive feeds like news
- **Set a higher refresh interval** (like 300 seconds) for feeds that update rarely
- **Limit max entries** to keep the block from getting too long on small screens
- **Turn off descriptions** if you only want headlines for a compact view

---

[← Back: Media Player](15-media-player.md)
