<!--
COPILOT_INSTRUCTIONS:
- You are assisting with building the project “slopCannon” in Go.
- Follow the structure in /docs.
- NEVER create files at the project root unless explicitly told.
- All new files must live inside the correct existing folders.
- Always propose diffs or file additions rather than overwriting entire folders.
- Respect modular architecture and keep dependencies minimal.
- All code must be idiomatic Go and follow Go module best practices.
-->



# slopCannon 
slopCannon is a modular Go-powered viral-clip generation toolkit.  
It blends AI heuristics, subtitle generation, ffmpeg editing, overlays, and a small UI layer on top of a CLI-first engine.

## 🌑 what this thing *is*
slopCannon takes a longform video  
→ slices it into moments worth caring about  
→ ranks them with AI + heuristic gut-feel  
→ lets you trim/edit/fuck with them  
→ overlays parkour or csgo surfing because the internet is dead inside  
→ spits out a fully-edited export with subtitles, stickers, SFX, whatever you want.

all modular. all configurable. all CLI-first.  
UI sits like a thin ghost on top.

---

# 🧩 core goals
- modular plugin architecture (ai, heuristics, overlays, subtitle engines, etc)
- lightning-fast parallel processing (goroutines, worker pools, optimized ffmpeg integration)
- cli-driven workflow with optional UI wrapper
- fully configurable through settings (yaml/toml/env/flags)
- buildable to a single go binary
- ai model for “viral potential” scoring
- whisper-based `.ass` subtitle generation + style options
- manual clip editing: trimming, reordering, deleting, timestamp fixing
- stickers / overlays / sfx timeline support
- proper error handling across pipeline
- comprehensive structured logging
- logs piped to UI + CLI stream when needed

---

# 🎞️ pipeline overview

```
input video  
→ analyze (ai + heuristics)  
→ detect clips  
→ rank  
→ edit (manual or auto)  
→ subtitle (whisper → .ass)  
→ overlay/stickers/sfx merge  
→ ffmpeg render  
→ export  
```

each step is a module under `internal/`.

---

# ⚡ performance
- goroutine worker pools  
- context cancellation everywhere  
- minimal blocking I/O  
- optimized ffmpeg invocation  
- zero global state  
- thread-safe clip timeline  
- fast model loading w/ caching

---

# 🐚 cli design
the cli is the source of truth — the ui is just a shy little façade.

```
slopcannon analyze input.mp4 --model=mini --overlay=surf --whisper=large
slopcannon render project.scn --verbose
slopcannon clip trim --start=10 --end=25
slopcannon config edit
slopcannon list plugins
```

---

# 🎨 overlays & stickers
- minecraft parkour  
- csgo surfing  
- subway surfers  
- animated stickers  
- sound effects bound to timestamps  
- plugin-based renderer for future chaos

---

# 🎛 config system
- global `config.yaml`
- plugin config sections
- override w/ `--flag` or `$ENV_VAR`
- auto-config generator

---

# 💬 subtitles
- whisper model (selectable size)
- `.ass` output
- style presets (font, color, shadow, outline, pos)
- offset correction
- editable timing

---

# 🧪 testing
- unit tests for each subsystem  
- integration tests for pipeline  
- mock ffmpeg + mock model layers  
- snapshot tests for subtitle rendering

---


