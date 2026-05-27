# Chapter 2: Getting Started

This chapter walks you through running OmniPanel-go for the first time and accessing it from your tablet or phone.

## Running Modes

OmniPanel-go can run in three different ways depending on your setup:

| Mode | When to use | Command |
|------|-------------|---------|
| **Default** | Everything on one PC | `./omnipanel-go` |
| **Server** | WebUI on a central PC, host behind firewall | `./omnipanel-go serve` |
| **Host Agent** | Connect to a central server | `./omnipanel-go connect <IP>:<PORT>` or `./omnipanel-go connect wss://<IP>:<PORT>` |

Most users should use the **Default** mode — it's the simplest and works great on a local network.

The **Server** and **Host Agent** modes are for advanced setups where you want the WebUI on one machine (always-on server) and the actual input simulation on another machine (possibly behind a firewall). See [Chapter 14: Tips and Troubleshooting](14-tips-and-troubleshooting.md) for more details.

### Authentication for Server and Host Agent Modes

If your server administrator has set up a secret token, you'll need to enter it when accessing the server. This protects the WebUI and host connections from unauthorized access.

**Accessing the WebUI with a token:**

When you visit the server URL, OmniPanel-go will automatically detect if authentication is required. If so, you'll see a login screen where you can enter your token:

1. Open `http://server-ip:3000` in your browser
2. If authentication is enabled, you'll see the login page
3. Enter your token and click **Login**
4. Optional: Check **Remember token** to stay logged in on future visits

You can also include the token directly in the URL (useful for bookmarks):
```
http://server-ip:3000/?token=your-secret-token
```

**Connecting the host agent with a token:**
Set the token in `config.json` on the host machine:
```json
{
  "server_address": "10.0.0.1:3000",
  "auth_token": "your-secret-token"
}
```
Or use a secure WebSocket URL (`wss://`) if your server is behind a reverse proxy with TLS:
```json
{
  "server_address": "wss://10.0.0.1:3000",
  "auth_token": "your-secret-token"
}
```
Or use the environment variable:
```bash
export OMNIPANEL_AUTH_TOKEN=your-secret-token
./omnipanel-go connect 10.0.0.1:3000
```

> **Note:** If no token is configured, everything works exactly as before — authentication is optional.

### Docker Container (Server Mode)

For users comfortable with Docker, the server mode can also run as a container:

```bash
docker run --rm -p 3000:3000 \
  -v $(pwd)/user:/var/run/ko/user \
  -w /var/run/ko \
  omnipanel-go:latest serve
```

The container automatically copies default panels, blocks, and themes into your `user/` folder on first run. Your customizations persist across container restarts. Speech recognition via Vosk is not available in the container — use llama-cpp-server (HTTP API) instead.

## Step 1: Run OmniPanel-go on Your PC

### On Windows (Default Mode)

1. Find the `omnipanel-go.exe` file in your OmniPanel-go folder
2. Double-click it to run
3. A command window will open — **leave it open** while you're using OmniPanel-go
4. You should see a message saying the server is running on port 3000

### On Linux (Default Mode)

1. Open a terminal
2. Navigate to your OmniPanel-go folder
3. Run the command:
   ```
   ./omnipanel-go
   ```
4. You should see a message saying the server is running on port 3000

> **Important:** Keep this window open. If you close it, OmniPanel-go stops running and your panels won't work.

### Log Output

OmniPanel-go shows colored log messages by default. You can change the log format if needed:

| Format | How to use | Best for |
|--------|-----------|----------|
| **Colored** (default) | `./omnipanel-go` | Normal use — easy to read |
| **Plain text** | `./omnipanel-go --log-format=text` | Saving to a log file |
| **JSON** | `./omnipanel-go --log-format=json` | Advanced users with log tools |

You can also set the `LOG_FORMAT` environment variable so you don't have to type the flag every time:
```
export LOG_FORMAT=json
./omnipanel-go
```

## Step 2: Find Your PC's IP Address

Your tablet needs to know where to find OmniPanel-go on your network. This address is your PC's IP address.

### On Windows

1. Press `Windows key + R`
2. Type `cmd` and press Enter
3. In the black window, type:
   ```
   ipconfig
   ```
4. Look for **IPv4 Address** under your WiFi or Ethernet adapter
5. It will look something like: `192.168.1.100`

### On Linux

1. Open a terminal
2. Type:
   ```
   ip addr show
   ```
3. Look for `inet` under your `wlan0` (WiFi) or `eth0` (Ethernet) interface
4. It will look something like: `192.168.1.100`

## Step 3: Open OmniPanel-go on Your PC

1. Open your web browser (Chrome, Firefox, Edge, etc.)
2. In the address bar, type:
   ```
   http://localhost:3000
   ```
3. Press Enter
4. You should see the **Start Page** with your panel list

> **Note:** `localhost` means "this computer." It only works when you're on the PC running OmniPanel-go.

## Step 4: Open OmniPanel-go on Your Tablet or Phone

1. Open the web browser on your tablet or phone
2. In the address bar, type:
   ```
   http://YOUR-PC-IP:3000
   ```
   Replace `YOUR-PC-IP` with the IP address you found in Step 2. For example:
   ```
   http://192.168.1.100:3000
   ```
3. Press Go/Enter
4. You should see the same **Start Page** as on your PC

> **Tip:** Bookmark this page on your tablet so you don't have to type it every time.

## Step 5: Verify It's Working

On the Start Page, you should see:

- A list of saved panels (might be empty if this is your first time)
- A section showing your connection URL
- A connection log at the bottom

If you see all of this, congratulations — OmniPanel-go is running and your tablet is connected!

## Common Issues

### "This site can't be reached" on your tablet

- Make sure both your PC and tablet are on the **same WiFi network**
- Check that OmniPanel-go is still running on your PC (the command window should be open)
- Double-check the IP address — it might have changed if your router restarted

### Firewall blocking the connection (Windows)

When you first run OmniPanel-go, Windows might ask if you want to allow it through the firewall. Click **Allow**.

If you missed this prompt:

1. Open Windows Security
2. Go to Firewall & network protection
3. Click "Allow an app through firewall"
4. Find `omnipanel-go.exe` and make sure both Private and Public are checked

### Wrong IP address

Your PC's IP address can change if your router restarts. If the connection suddenly stops working, check your IP address again using the steps in Step 2.

### Storing user data in a custom location

By default, OmniPanel-go saves all your panels and settings in a folder called `user/`. If you want this folder on a different drive (e.g., D:), in another folder, or synced via a cloud drive, you can set a custom path in `config.json`:

1. Open `config.json` in a text editor
2. Add this line next to `"numJoysticks"`:
   ```json
   "user_path": "D:\\Games\\OmniPanelData"
   ```
   On Linux/Mac, use forward slashes without backslashes: `"user_path": "/media/mydrive/omnipanel-user"`
3. Save the file and restart OmniPanel-go

A custom `user_path` can be an absolute path (like above) or a path relative to where OmniPanel-go is running. OmniPanel-go will create the folder if it doesn't exist.

## What's Next?

Now that OmniPanel-go is running, let's explore the [Start Page](03-the-start-page.md) — your dashboard for managing panels and connections.
