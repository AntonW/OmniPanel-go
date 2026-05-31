# Chapter 14: Tips and Troubleshooting

This chapter covers best practices for designing great panels, common issues you might encounter, and how to fix them.

## Design Best Practices

### Plan Your Layout First

Before opening the editor, sketch your panel on paper:

1. What controls do you need?
2. Which ones do you use most often?
3. How should they be grouped?
4. What size should each control be?

A quick sketch saves time and results in a better layout.

### Prioritize Frequently Used Controls

- Place **most-used controls** in easy-to-reach areas (center or bottom of the screen)
- Make them **larger** for easier tapping
- Use **distinctive colors** so you can find them quickly

### Use Consistent Colors

Create a color scheme and stick to it:

| Color | Suggested Use |
|-------|--------------|
| Green | Navigation, movement, positive actions |
| Red | Weapons, combat, emergency actions |
| Blue | Systems, utilities, information |
| Yellow | Warnings, alerts, caution |
| Gray | Disabled or inactive controls |

### Leave Space Between Controls

Don't pack controls too tightly. Leave at least one grid unit between blocks so you don't accidentally tap the wrong one during intense gameplay.

### Use Pages Wisely

- Keep related controls on the same page
- Don't create too many pages (3-5 is ideal)
- Name pages clearly ("Flight," "Combat," not "Page 1," "Page 2")

### Test on Your Actual Device

Design on your PC, but always test on the tablet or phone you'll actually use while gaming:

- Check that buttons are big enough to tap
- Verify colors are readable in your gaming environment
- Make sure the layout feels natural in your hands

---

## Common Issues and Fixes

### OmniPanel-go Won't Start

**Problem:** You double-click `omnipanel-go.exe` (or run `./omnipanel-go`) and nothing happens.

**Solutions:**
- **Windows:** Open a command prompt, navigate to the OmniPanel-go folder, and run `omnipanel-go.exe` from there. You'll see error messages if something is wrong.
- **Linux:** Run `./omnipanel-go` in a terminal to see error messages.
- **Port already in use:** If port 3000 is already in use by another program, change the port in `config.json`:
  ```json
  {
    "port": 3001
  }
  ```

### Can't Connect from Tablet

**Problem:** Your tablet shows "This site can't be reached."

**Solutions:**
1. Verify both devices are on the **same WiFi network**
2. Check that OmniPanel-go is still running on your PC
3. Verify the IP address is correct (it may have changed)
4. Check your PC's firewall — OmniPanel-go might be blocked
5. Try accessing from your PC browser first using `http://localhost:3000`

### Panel Not Responding to Touches

**Problem:** You tap buttons on your panel, but nothing happens in the game.

**Solutions:**
1. Check the connection log on the Start Page — is your device connected? Is the host agent connected?
2. In relay mode, look for the **Host IP indicator** in the top-left corner of the panel when a host connects (it fades out after 5 seconds) — if it doesn't appear, no host is connected. You can also check the connection log on the Start Page.
3. Verify the **Joystick Index** and **Button/Slider ID** are correct
4. Make sure your game is configured to listen to the correct virtual joystick
5. Test inputs using your OS's joystick testing tool (see Chapter 12)

### Virtual Joystick Not Detected by Game

**Problem:** Your game doesn't see the virtual joystick.

**Solutions:**
- **Windows:**
  - Make sure vJoy is installed (version 2.2.2.0)
  - Open vJoy configuration and verify devices are enabled
  - Restart OmniPanel-go after installing vJoy
  - Restart your game after starting OmniPanel-go

- **Linux:**
  - Check permissions: `ls -l /dev/uinput`
  - Run OmniPanel-go with `sudo` if needed
  - Verify the `uinput` module is loaded: `lsmod | grep uinput`

### Speech Recognition Not Working

**Problem:** You speak but nothing happens.

**Solutions:**
1. Check that speech is enabled in `config.json` (`"enabled": true`)
2. Verify your browser has microphone permission
3. Check the browser console for errors (press F12)
4. If using Vosk, verify the model downloaded correctly
5. If using llama-cpp, verify the server is running and the URL is correct
6. Try push-to-talk mode first to isolate the issue

### Commands Not Executing

**Problem:** You tap a Command Block but nothing happens.

**Solutions:**
1. Check the toast notification for error messages
2. Test the command in a terminal first
3. Verify the command path is correct (watch for typos)
4. On Windows, wrap paths with spaces in quotes: `"C:\Program Files\App\app.exe"`
5. Check that OmniPanel-go has permission to run the command

### Panel Looks Different on Tablet vs PC

**Problem:** Your panel layout looks fine on your PC but messed up on your tablet.

**Solutions:**
1. Check the aspect ratio setting in the editor — match it to your tablet
2. Use percentage-based sizing (the grid system handles this automatically)
3. Test on your actual tablet during design, not just the PC preview
4. Avoid fixed pixel sizes — use grid units instead

---

## FAQ

### Can I use OmniPanel-go over the internet?

No. OmniPanel-go is designed for local network use only. Both your PC and tablet must be on the same WiFi network. This is intentional for privacy and security.

### Can multiple people use OmniPanel-go at the same time?

Yes. Multiple devices can connect to OmniPanel-go simultaneously, and each can open different panels.

### Do I need to install anything on my tablet?

No. OmniPanel-go runs entirely in your tablet's web browser. No apps to install.

### Will OmniPanel-go work with any game?

OmniPanel-go works with games that support joystick or mouse input. Most PC games support these input methods. Games that only support keyboard input won't work directly (though you could use Command Blocks to run keyboard simulation tools).

### How do I back up my panels?

Your panels are saved as JSON files in the `user/panels/` folder. Copy this folder to back up all your panels.

### How do I share my panels with someone else?

Copy the panel JSON file from `user/panels/` and place it in the same folder on another OmniPanel-go installation.

### Can I use OmniPanel-go on a Mac?

You can run the OmniPanel-go server and design panels on a Mac, but virtual joystick and mouse input won't work. macOS is not supported for game input simulation.

### Does OmniPanel-go work with controllers like Xbox or PlayStation pads?

OmniPanel-go creates virtual joysticks that games see as generic controllers. It doesn't interact with physical controllers. You can use a physical controller and OmniPanel-go panels at the same time — they'll appear as separate devices to the game.

### How do I update OmniPanel-go?

Download the latest version and replace your existing `omnipanel-go` binary. Your `user/` folder (panels, blocks, config) is preserved, so you won't lose any of your work.

### Can I change the port number?

Yes. Edit `config.json`:
```json
{
  "port": 8080
}
```
Or use the environment variable `OMNIPANEL_PORT=8080`.

### How do I increase the number of virtual joysticks?

Edit `config.json`:
```json
{
  "numJoysticks": 8
}
```
Or use the environment variable `OMNIPANEL_NUMJOYSTICKS=8`.

### How do I set up authentication for my server?

If you're running OmniPanel-go in **Server** mode (`./omnipanel-go serve`), you can protect the WebUI and host connections with a secret token:

1. Edit `config.json` on the server:
    ```json
    {
      "auth_token": "your-secret-token"
    }
    ```
2. Restart the server: `./omnipanel-go serve`
3. Access the WebUI — you'll see a login screen where you can enter your token
4. On the host machine, set the same token in `config.json` or use `OMNIPANEL_AUTH_TOKEN`

> **Important:** Both the server and the host agent must use the same token. If they don't match, the host will be rejected with an "unauthorized" error.

### I get "unauthorized" when connecting to the server

This means the server has authentication enabled but your token is missing or incorrect:

1. Check that `auth_token` in your `config.json` matches the server's token
2. If using the environment variable, verify `OMNIPANEL_AUTH_TOKEN` is set correctly
3. For browser access, enter your token on the login page or include `?token=your-secret-token` in the URL
4. Ask your server administrator for the correct token

### How does the login page work?

When authentication is enabled, OmniPanel-go shows a login screen instead of requiring you to type the token in the URL. Here's what happens:

1. You visit the server URL (e.g., `http://server:3000`)
2. The browser checks if a token is already saved or in the URL
3. If not, it asks the server if auth is required (by calling `/api/config`)
4. If auth is required, you're redirected to the login page
5. Enter your token and click **Login** — the server validates it
6. Your token is saved and you're redirected back to the page you wanted

If you check **Remember token**, it stays saved in your browser for future visits. If you don't check it, the token is only saved for that browser tab.

---

## Getting Help

If you're stuck:

1. **Check this manual** — your issue might be covered in an earlier chapter
2. **Check the connection log** — it shows connection and disconnection events
3. **Check the terminal output** — OmniPanel-go logs errors and warnings to the terminal. Use `--log-format=text` for cleaner output when copying error messages.
4. **Test in isolation** — try one feature at a time to identify the problem
5. **Check the config file** — verify your settings in `config.json`

---

## Quick Reference

### Default Port
```
3000
```

### Access URLs
```
Start Page:  http://YOUR-PC-IP:3000/
Editor:      http://YOUR-PC-IP:3000/editor
Panel:       http://YOUR-PC-IP:3000/panel?name=PanelName
```

### File Locations
```
Config:          config.json
User Data:       user/ (overridable via user_path in config.json)
Panels:          user/panels/
Custom Blocks:   user/blocks/
Speech Commands: user/speech_commands.json
Assets:          user/assets/
```

### Joystick Limits
```
Joysticks:  10 (indices 0-9)
Buttons:    16 per joystick (IDs 0-15)
Axes:       8 per joystick (IDs 0-7)
```

### Axis Names
```
0: X         1: Y         2: Z         3: RX
4: RY        5: RZ        6: Throttle  7: Rudder
```

### System Metrics Data Keys
```
cpu_usage      memory_usage    disk_usage
network_rx     network_tx
```

---

## Congratulations!

You've completed the OmniPanel-go User Manual. You now know how to:

- Set up and run OmniPanel-go
- Create and customize control panels
- Use all block types (buttons, sliders, touchpads, command blocks, data displays)
- Organize panels with pages and sections
- Display live data
- Control your panel with voice commands
- Control media playback with the Media Player block
- Display live news and blogs with RSS Feeds
- Create custom blocks
- Troubleshoot common issues

Now go build some awesome control panels and enjoy your games with a custom cockpit dashboard!

Continue reading about [Media Player](15-media-player.md) to display and control music/video playback, or check out [RSS Feeds](16-rss-feeds.md) for live news and blogs.
