from faster_whisper import WhisperModel
from pathlib import Path
from typing import List

class SubtitleGenerator:
    def __init__(self, model_size="small"):
        print(f"[SubtitleGenerator] Loading Whisper model: {model_size}")
        self.model = WhisperModel(model_size, device="cpu", compute_type="int8")

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
        model_size: str = "small",
    ):
        """
        Generate styled ASS subtitles with word-level karaoke + fade effect.
        """

        print(f"[SubtitleGenerator] Starting transcription: {audio_path}")
        segments, info = self.model.transcribe(
            str(audio_path),
            beam_size=5,
            word_timestamps=True
        )

        print(f"[SubtitleGenerator] Detected language: {info.language}, "
              f"Probability: {info.language_probability:.2f}")

        if output_path.suffix.lower() != ".ass":
            output_path = output_path.with_suffix(".ass")

        with open(output_path, "w", encoding="utf-8") as f:
            # --------------------
            # Header
            # --------------------
            f.write("[Script Info]\n")
            f.write("Title: slopCannon subs\n")
            f.write("ScriptType: v4.00+\n")
            f.write("PlayResX: 1080\n")
            f.write("PlayResY: 1920\n\n")

            # --------------------
            # Styles
            # --------------------
            f.write("[V4+ Styles]\n")
            f.write(
                "Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, "
                "OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, "
                "ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, "
                "Alignment, MarginL, MarginR, MarginV, Encoding\n"
            )
            f.write(
                f"Style: Default,{font},{font_size},{primary_color},{secondary_color},"
                f"{outline_color},{back_color},-1,0,0,0,100,100,0,0,1,3,0,5,40,40,40,1\n\n"
            )

            # --------------------
            # Events
            # --------------------
            f.write("[Events]\n")
            f.write(
                "Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n"
            )

            # flatten words from all segments
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

            # chunk into windows
            chunk_size = words_per_line * lines_per_event
            chunks: List[List] = [
                all_words[i:i + chunk_size]
                for i in range(0, len(all_words), chunk_size)
            ]

            for chunk in chunks:
                start = self._format_timestamp(chunk[0].start)
                end = self._format_timestamp(chunk[-1].end)

                # build stacked lines
                lines = []
                for line_i in range(lines_per_event):
                    line_words = chunk[line_i * words_per_line:(line_i + 1) * words_per_line]
                    if not line_words:
                        continue
                    parts = []
                    for w in line_words:
                        dur_cs = int((w.end - w.start) * 100)
                        parts.append(f"{{\\k{dur_cs}}}{w.word.strip()}")
                    lines.append(" ".join(parts))

                # combine lines w/ newline
                text = "\\N".join(lines)
                text = f"{{\\fad({fade_ms},{fade_ms})}}{text}"
                line = f"Dialogue: 0,{start},{end},Default,,0,0,0,,{text}\n"
                f.write(line)

        print(f"[SubtitleGenerator] ASS subtitles saved to {output_path}")
        return output_path

    @staticmethod
    def _format_timestamp(seconds: float):
        h = int(seconds // 3600)
        m = int((seconds % 3600) // 60)
        s = int(seconds % 60)
        cs = int((seconds - int(seconds)) * 100)
        return f"{h:d}:{m:02d}:{s:02d}.{cs:02d}"
