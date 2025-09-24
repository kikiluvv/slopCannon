# slopcannon/managers/analysis_manager.py

import subprocess
import tempfile
import numpy as np
import librosa
import cv2
import os

class AnalysisManager:
    """
    Scans a video file and suggests clip ranges with heuristic 'viral potential' scores.
    """

    def __init__(self, log_callback=print):
        self.log = log_callback

    def suggest_clips(self, input_file, window_sec=20, stride_sec=5, sr=16000):
        """
        Analyze video and return suggested clips.
        Returns: list of (start_ms, end_ms, score)
        """
        self.log(f"[Analysis] Starting analysis of {input_file}")

        # --- extract audio ---
        with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp_audio:
            audio_path = tmp_audio.name
        self.log(f"[Analysis] Extracting audio to temp file: {audio_path}")
        cmd = [
            "ffmpeg", "-y", "-i", str(input_file),
            "-vn", "-ac", "1", "-ar", str(sr), audio_path
        ]
        subprocess.run(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        self.log("[Analysis] Audio extraction complete")

        y, sr = librosa.load(audio_path, sr=sr)
        self.log(f"[Analysis] Loaded audio: {y.shape[0]} samples at {sr}Hz")
        os.remove(audio_path)
        self.log("[Analysis] Temporary audio file deleted")

        # loudness envelope
        frame_length = int(0.5 * sr)
        hop_length = int(0.25 * sr)
        rms = librosa.feature.rms(y=y, frame_length=frame_length, hop_length=hop_length)[0]
        self.log(f"[Analysis] RMS calculated: {len(rms)} frames")

        # speech density proxy = zero-crossing rate
        zcr = librosa.feature.zero_crossing_rate(y, frame_length=frame_length, hop_length=hop_length)[0]
        self.log(f"[Analysis] Zero-crossing rate calculated: {len(zcr)} frames")

        # --- windowing ---
        duration = librosa.get_duration(y=y, sr=sr)
        self.log(f"[Analysis] Audio duration: {duration:.2f}s, Window: {window_sec}s, Stride: {stride_sec}s")
        windows = []
        for start in np.arange(0, duration - window_sec, stride_sec):
            end = start + window_sec
            i0 = int(start * sr / hop_length)
            i1 = int(end * sr / hop_length)
            loud = float(np.mean(rms[i0:i1]))
            density = float(np.mean(zcr[i0:i1]))
            score = 0.6 * loud + 0.4 * density
            windows.append((int(start*1000), int(end*1000), score))
            self.log(f"[Suggested Window] {start:.2f}-{end:.2f}s | Loud: {loud:.4f}, Density: {density:.4f}, Score: {score:.4f}")

        # pick top N windows
        windows.sort(key=lambda x: x[2], reverse=True)
        top = windows[:5]
        self.log(f"[Analysis] Suggested top {len(top)} clips:")
        for idx, (s, e, sc) in enumerate(top, 1):
            self.log(f"  [{idx}] {s}ms → {e}ms | Score: {sc:.4f}")

        return top
