from faster_whisper import WhisperModel
from pathlib import Path

class SubtitleGenerator:
    def __init__(self, model_size="small"):
        print(f"[SubtitleGenerator] Loading Whisper model: {model_size}")
        self.model = WhisperModel(model_size, device="cpu", compute_type="int8")

    def generate_subtitles(self, audio_path: Path, output_path: Path):
        print(f"[SubtitleGenerator] Starting transcription: {audio_path}")
        segments, info = self.model.transcribe(str(audio_path), beam_size=5)
        
        print(f"[SubtitleGenerator] Detected language: {info.language}, Probability: {info.language_probability:.2f}")
        
        # write to .srt
        with open(output_path, "w", encoding="utf-8") as f:
            for i, seg in enumerate(segments, start=1):
                print(f"[SubtitleGenerator] {seg.start:.2f}–{seg.end:.2f}: {seg.text}")
                start = self._format_timestamp(seg.start)
                end = self._format_timestamp(seg.end)
                text = seg.text.strip()
                f.write(f"{i}\n{start} --> {end}\n{text}\n\n")
        
        print(f"[SubtitleGenerator] Subtitles saved to {output_path}")
        return output_path

    @staticmethod
    def _format_timestamp(seconds: float):
        ms = int((seconds - int(seconds)) * 1000)
        h = int(seconds // 3600)
        m = int((seconds % 3600) // 60)
        s = int(seconds % 60)
        return f"{h:02}:{m:02}:{s:02},{ms:03}"
