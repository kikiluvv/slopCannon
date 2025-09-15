slopCannon/
├─ cmd/
│  └─ slopCannon/          # cobra-generated entrypoint
│     ├─ main.go
│     └─ process.go        # cobra "process" command
├─ internal/
│  ├─ pipeline/
│  │   └─ process.go       # orchestrates pipeline steps
│  ├─ ffmpeg/
│  │   ├─ split.go         # cut into chunks
│  │   ├─ overlay.go       # add subtitles + bg
│  │   └─ utils.go         # generic ffmpeg exec
│  ├─ subtitles/
│  │   ├─ transcribe.go    # Whisper/OpenAI integration
│  │   └─ format.go        # text → .srt
│  └─ config/
│      └─ config.go        # global settings
├─ assets/
│  └─ backgrounds/         # minecraft bg loops
├─ output/
└─ go.mod
