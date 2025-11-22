# dzjoin
A lightweight DayZ helper tool that:
- Won't be too heavily maintained.

- Fetches server info from BattleMetrics  
- Downloads and updates required workshop mods  
- Syncs mods into your DayZ directory  
- Optionally cleans old mods  
- Launches DayZ with correct `-mod=` arguments  

---

## Requirements

- Linux Steam, SteamCMD
- Go 1.20+
- DayZ installed (Proton AppID 221100)

---
## Environment Variables

Create a `.env` file in:

```
~/.config/dzjoin/.env
```

This directory will be **created automatically** the first time you run `dzjoin`.  
If you prefer to create it manually:

```bash
mkdir -p ~/.config/dzjoin
chmod 700 ~/.config/dzjoin
```

Then place your `.env` file inside that directory, or export the variables globally:

```bash
DZJOIN_SERVER="1234"                        # BattleMetrics server ID
DZJOIN_NAME="blade"                         # Your DayZ/Steam profile name
DZJOIN_STEAMCMD_PATH="/home/user/Steam/steamcmd.sh"   # Optional: manual steamcmd path
```

If `DZJOIN_STEAMCMD_PATH` is not set, dzjoin will:

- Look for `steamcmd` in your PATH  
- If not found, it will exit with an error  

---

## Flags

| Flag | Long | Description |
|------|------|-------------|
| `-u` | `--update` | Download/update workshop mods |
| `-c` | `--clean` | Delete all `@mod` folders in DayZ directory |
| `-d` | `--dont-join` | Skip launching the game |

---

## What dzjoin does

1. Loads environment variables  
2. Retrieves server info from BattleMetrics:
   - Server IP  
   - Port  
   - Required mod IDs  
   - Required mod names  
3. If `-c` is set:
   - Removes all `@mod` folders from DayZ directory  
4. If `-u` is set:
   - Downloads required mods via SteamCMD  
   - Copies/syncs mods into DayZ installation  
5. Unless `-d` is set:
   - Launches DayZ with:

```bash
steam -applaunch 221100 \
  -noLauncher -nosplash -skipintro \
  -connect=<IP> -port=<PORT> \
  -mod=@Mod1;@Mod2;...
```

---
Happy surviving!