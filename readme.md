# 💣 slopCannon 💣
Generate slop and see what sticks to the wall.

## Requirements
- Python 3.10+ (tested with 3.11)
- ffmpeg + headers 
- ffprobe 
- OpenCV + headers
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

## Usage

1. **Load a video** into the app.
2. **Configure subtitle settings**  
   - Customize line breaks, font, size, and colors.  
   - Choose a Whisper model (larger models are more accurate but slower).  
   - Click `Save Settings` to apply your edits.
3. **Trim clips** for short-form content 
   - ***Optional*** - Use the `Find Viral Clips` button to algorithmically trim out clips based on a "virality score" from the clip analyzer. 
   - Use `Mark Start` to set the beginning of a clip.  
   - Use `Mark End` to set the ending of a clip.  
   - You can mark multiple clips before exporting, but note that processing them concurrently may slow things down.
   - Use the `Manage Clips` button to edit or delete existing clips
4. **Export clips**  
   - Click `Export Clips` and select an output folder.  
   - The app will trim each marked clip, apply filters, and generate subtitles.  
   - Temporary files (intermediate MP4s, audio WAVs, and `.ass` subtitle files) are automatically cleaned up, leaving only the final exported clips.


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
- Animated text effects
- AI "Hook" preamble 
    - AI voiced preamble before clip starts

### Video Processing
- Custom output resolution & frame rate for portrait, TikTok, YouTube shorts
- Overlay flexibility: dynamic selection of overlay video or image
- Optional audio normalization for consistent volume
- Parallel processing / multithreading to speed up batch jobs 
- GPU acceleration option for faster encoding 
- Manual and AI generated sound effects
- More AI slop

### UX / Workflow
- CLI + GUI support for clip selection, styling, and output folder
- Realtime transcription / preview of subtitles while clip plays
- Progress reporting: percentage, ETA for long clips
- Error handling & logging: skip bad files and log failures

### Advanced Features
- Integration with TikTok / YouTube: auto-format clips for different platforms
- Auto clipping for clips deemed "probable of going viral" by AI or algorithm
- AI virality scoring 