# 🌑 slopCannon Architecture

slopCannon is built like a haunted machine —
clean layers, modular guts, everything replaceable, nothing sacred.
a Go project that doesn’t trip over itself when it grows fatter and meaner.

this doc defines:
- directory layout
- responsibilities
- conventions
- how every subsystem talks to every other subsystem without screaming

## 📁 Project Structure

```
slopcannon/
│
├── cmd/
│   └── slopcannon/
│       └── main.go
│
├── internal/
│   ├── ai/
│   ├── clips/
│   ├── config/
│   ├── ffmpeg/
│   ├── logging/
│   ├── overlays/
│   ├── pipeline/
│   ├── subtitles/
│   └── ui/
│
├── pkg/
│   └── util/
│
├── docs/
│
└── go.mod
```

### 📦 cmd/ — entrypoints (the summoning circle)

`cmd/slopcannon` holds the real executable.

Rules:
- one folder per binary
- minimal logic — just glue and bootstrapping
- initialize config, logging, dependency graph
- wire up CLI commands

Think of it as:

> the doorway the user steps through before falling into the internal abyss.

### 🖤 `internal/` — the guts (private, messy, powerful)

all the real machinery lives in here.
no other module can import this stuff — it’s our private playground.

modules inside `internal/`:

#### `internal/ai/`— models, heuristics, “viral potential”
- ai scoring
- tiny models
- ranking logic combining heuristic + ai
- feature extraction for clips

must expose a clean interface like:

```
type Scorer interface {
    Score(clip Clip) (float64, error)
}
```

#### `internal/clips/` — clip detection, trimming, stitching

responsible for:
- scanning long videos
- slicing into candidates
- timestamp fixing
- merging or splitting
- manual editing helpers
- pure logic, no UI, no ffmpeg calls.

#### `internal/config/` — config loader (yaml / toml / env)

holds:
- app-wide config
- default values
- dynamic reload in future

#### `internal/ffmpeg/` — ffmpeg integration layer

a wrapper around ffmpeg that:
- provides typed Go functions
- manages subprocesses
- streams logs back to pipeline/UI
- handles error wrapping
- handles cancellation with context
- this must stay thin and fast.

#### `internal/logging/` — structured zerolog wrapper
- unified logger
- timestamped
- leveled
- UI-safe log streaming

#### `internal/overlays/` — parkour, surfing, stickers, SFX
- each overlay = module
- stickers = composable
- sound effects = timeline nodes
export clean interfaces:
```
type Overlay interface {
    Apply(*Frame) error
}
```

#### `internal/pipeline/` — the beating heart

this is where the magic and suffering happen.

pipeline responsibilities:
- orchestrates multi-step clip generation
- parallelizes analysis + detection
- assembles timeline
- hands final output to ffmpeg
- streams progress + logs
- supports cancellation
worker-pool architecture:
```
ingest → detect → score → rank → refine → render
```
each step its own unit.

#### `internal/subtitles/` — whisper engine + .ass generator

- whisper inference
- ass styling
- timing adjustments
- offset correction
- export & preview

#### `internal/ui/` — optional user interface

thin layer glued on top of CLI.

### 📦 `pkg/` — public utilities (shared with the world)

contains tiny helper libs that are NOT slopCannon-specific:
- safe file IO
- timestamp math
- mini event bus
- frame or clip primitives
- reusable wrappers

this stuff is intentionally importable by other Go projects.

### 📁 `docs/` — everything Copilot reads so it behaves

contains:
- ARCHITECTURE.md
- MODULES.md
- ROADMAP.md

Copilot uses these as its “brain” to scaffold new functions correctly.

### 📜 Dependency Rules
Allowed direction of flow:

```
cmd → internal/* → pkg/*
internal/*  → other internal modules (sparingly)
pkg/*       → nothing depends on slopcannon
docs/*      → read by humans + copilot only
```

### Forbidden

- internal importing cmd
- submodules mutually depending on each other
- ffmpeg logic leaking into ai logic
- UI touching pipeline internals directly
- god objects (they kill performance & joy)

### 🌪 Concurrency Model

- goroutines everywhere
- worker pools for heavy analysis
- channels for streaming progress/logs
- context.Context for all long tasks
- avoid global state like the plague

### 🔥 Design Principles

- CLI-first, UI is optional ornament
- each module replaceable (pluggable design)
- favor composition over inheritance
- fail loudly, log quietly
- pure logic in modules
- ffmpeg called at the edges only
- deterministic output where possible
- every part testable without video files

### 🖤 Mood of the Code

the code should feel like:
- clean
- readable
- no mysterious side effects
- each file less than 300 lines
- short functions
- clear naming
- comments only where needed
- subtle sadness humming underneath
