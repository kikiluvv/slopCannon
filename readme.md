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
