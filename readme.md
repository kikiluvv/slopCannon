# 💣 slopCannon 💣

Generate slop and see what sticks to the wall.

## Features

- Modular UI structure (ui/ folder)
- Clean main entrypoint
- Ready for expansion with more UI modules or background tasks

## Requirements

- Python 3.10+ (tested with 3.11)
- PyQt5
- ffmpeg + headers (binary included)
- ffprobe (binary included)
- virtualenv (highly recommended)

## Installation
1. clone repo
```
git clone https://github.com/kikiluvv/slopCannon
cd slopCannon
```
2. create venv
```
python3 -m venv venv
source venv/bin/activate  # macOS / Linux

# OR

.\venv\Scripts\activate  # Windows
```

3. install dependencies

`pip install -r requirements.txt`


## Running the App
1. **Recommended**: module way

`python3 -m slopcannon.main`

2. Direct file (less clean)

`python3 slopcannon/main.py`


*make sure to fix imports in main.py to relative imports if you go this route:*

`from .ui.main_window import MainWindow`

## Project Structure
```
slopCannon/
├─ venv/                 # virtualenv
├─ slopcannon/
│  ├─ __init__.py
│  ├─ main.py            # main entrypoint
│  └─ ui/
│     └─ main_window.py  # main window GUI
```

## Notes

1. Always run from the project root (`slopCannon/`)
2. Add new UI modules inside slopcannon/ui/
3. Keep your imports consistent: relative inside package, absolute outside

## TODO
### Subtitles
- Generate .ass files instead of .srt for advanced styling and karaoke effects
- Support multiple subtitle formats: .srt, .ass, .vtt
- Dynamic karaoke effects: highlight words as spoken, support bold/italic/emphasis
- Adjustable max words per caption for smart wrapping and readability
- Custom styling options: font, size, color, outline, shadow

### Video Processing
- Mass-processing of clips / batch export
- Merge multiple clips into one video with continuous subtitles
- Custom output resolution & frame rate for portrait, TikTok, YouTube shorts
- Overlay flexibility: dynamic selection of overlay video or image
- Optional audio normalization for consistent volume
- Parallel processing / multithreading to speed up batch jobs
- GPU acceleration option for faster encoding (if available)

### UX / Workflow
- CLI + GUI support for clip selection, styling, and output folder
- Realtime transcription / preview of subtitles while clip plays
- Progress reporting: percentage, ETA for long clips
- Error handling & logging: skip bad files and log failures
- Auto-clean temporary files (e.g., .wav, intermediate clips)

### Advanced Features
- Voice-based scene splitting: automatically mark new captions on speaker change or pause
- Integration with TikTok / YouTube: auto-format clips for different platforms