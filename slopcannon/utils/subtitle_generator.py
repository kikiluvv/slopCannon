from pathlib import Path
from typing import List
from faster_whisper import WhisperModel

class SubtitleGenerator:
    def __init__(self, model_size="small", device="cpu", compute_type="int8", log_callback=print):
        self.model_size = model_size
        self.device = device
        self.compute_type = compute_type
        self.log_callback = log_callback
        self.model = None
        self.closed = False
        self.log_callback("[SubtitleGenerator] Initialized (model not loaded yet)")

    def update_model(self, model_size=None, device=None, compute_type=None):
            if model_size:
                self.model_size = model_size
            if device:
                self.device = device
            if compute_type:
                self.compute_type = compute_type
            # reload model if already loaded
            self.model = None

    def _ensure_model(self):
        if self.closed:
            raise RuntimeError("SubtitleGenerator is closed!")
        if self.model is None:
            self.log_callback(f"[SubtitleGenerator] Loading Whisper model: {self.model_size} ({self.device}, {self.compute_type})")
            self.model = WhisperModel(self.model_size, device=self.device, compute_type=self.compute_type)
            self.log_callback("[SubtitleGenerator] Model loaded successfully")

    def generate_subtitles(
        self,
        audio_path: Path,
        output_path: Path,
        words_per_line: int = 5,
        lines_per_event: int = 2,
        fade_ms: int = 100,
        font: str = "Comic Sans MS",
        font_size: int = 72,
        primary_color: str = "&H00FFFFFF",
        secondary_color: str = "&H0000FFFF",
        outline_color: str = "&H00000000",
        back_color: str = "&H64000000",
        log_callback=None,
    ):
        log = log_callback or self.log_callback
        self._ensure_model()
        log(f"[SubtitleGenerator] Starting transcription: {audio_path}")

        segments, info = self.model.transcribe(
            str(audio_path), beam_size=5, word_timestamps=True
        )
        log(f"[SubtitleGenerator] Detected language: {info.language}, Probability: {info.language_probability:.2f}")

        if output_path.suffix.lower() != ".ass":
            output_path = output_path.with_suffix(".ass")
        log(f"[SubtitleGenerator] Writing ASS subtitles to: {output_path}")

        with open(output_path, "w", encoding="utf-8") as f:
            # header
            f.write("[Script Info]\nTitle: slopCannon subs\nScriptType: v4.00+\nPlayResX: 1080\nPlayResY: 1920\n\n")
            f.write("[V4+ Styles]\n")
            f.write(
                "Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, "
                "Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, "
                "Alignment, MarginL, MarginR, MarginV, Encoding\n"
            )
            f.write(
                f"Style: Default,{font},{font_size},{primary_color},{secondary_color},{outline_color},{back_color},"
                "-1,0,0,0,100,100,0,0,1,3,0,5,40,40,40,1\n\n"
            )
            f.write("[Events]\n")
            f.write("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")

            # flatten all words
            all_words = []
            for seg in segments:
                if seg.words:
                    all_words.extend(seg.words)
                else:
                    dummy = type("DummyWord", (), {})()
                    dummy.start = seg.start
                    dummy.end = seg.end
                    dummy.word = seg.text
                    all_words.append(dummy)

            # chunk
            chunk_size = words_per_line * lines_per_event
            chunks: List[List] = [all_words[i:i+chunk_size] for i in range(0, len(all_words), chunk_size)]

            for i, chunk in enumerate(chunks):
                start = self._format_timestamp(chunk[0].start)
                end = self._format_timestamp(chunk[-1].end)
                log(f"[SubtitleGenerator] Writing chunk {i+1}/{len(chunks)}: {start} → {end}, words={len(chunk)}")

                lines = []
                for line_i in range(lines_per_event):
                    line_words = chunk[line_i*words_per_line:(line_i+1)*words_per_line]
                    if not line_words:
                        continue
                    parts = [f"{{\\k{int((w.end-w.start)*100)}}}{w.word.strip()}" for w in line_words]
                    lines.append(" ".join(parts))

                text = "\\N".join(lines)
                text = f"{{\\fad({fade_ms},{fade_ms})}}{text}"
                f.write(f"Dialogue: 0,{start},{end},Default,,0,0,0,,{text}\n")

        log(f"[SubtitleGenerator] ASS subtitles saved to {output_path}")
        return output_path

    @staticmethod
    def _format_timestamp(seconds: float):
        h = int(seconds // 3600)
        m = int((seconds % 3600) // 60)
        s = int(seconds % 60)
        cs = int((seconds - int(seconds)) * 100)
        return f"{h:d}:{m:02d}:{s:02d}.{cs:02d}"

    def close(self):
        if not self.closed:
            self.model = None
            self.closed = True
            self.log_callback("[SubtitleGenerator] Model closed")
