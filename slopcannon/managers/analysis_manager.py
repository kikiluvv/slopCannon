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
    Combines audio and visual heuristics (scene changes + motion).
    """

    def __init__(self, log_callback=print):
        self.log = log_callback

    # --------------------
    # Visual Heuristics
    # --------------------
    def _calc_scene_change_scores(self, video_path, window_sec, stride_sec):
        self.log("[Visual] Calculating scene change scores...")
        cap = cv2.VideoCapture(str(video_path))
        fps = cap.get(cv2.CAP_PROP_FPS)
        frame_count = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))
        duration = frame_count / fps
        num_windows = max(1, int(duration / stride_sec))
        scores = np.zeros(num_windows)

        prev_hist = None
        frame_idx = 0
        log_interval = max(1, frame_count // 20)

        while True:
            ret, frame = cap.read()
            if not ret:
                break
            gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
            hist = cv2.calcHist([gray], [0], None, [256], [0,256])
            hist = cv2.normalize(hist, hist).flatten()

            if prev_hist is not None:
                diff = cv2.compareHist(prev_hist, hist, cv2.HISTCMP_BHATTACHARYYA)
                window_idx = int(frame_idx / fps / stride_sec)
                if window_idx < len(scores):
                    scores[window_idx] += diff
            prev_hist = hist
            frame_idx += 1

            if frame_idx % log_interval == 0:
                self.log(f"[Visual][Scene] Processed frame {frame_idx}/{frame_count} ({frame_idx/frame_count:.0%})")

        cap.release()
        self.log("[Visual][Scene] Done calculating scene changes")
        scores = (scores - np.min(scores)) / (np.ptp(scores)+1e-6)
        return scores

    def _calc_motion_scores(self, video_path, window_sec, stride_sec):
        self.log("[Visual] Calculating motion scores (ultra-fast mode)...")
        cap = cv2.VideoCapture(str(video_path))
        ret, prev_frame = cap.read()
        if not ret:
            return np.zeros(1)

        scale_w, scale_h = 128, 72
        prev_gray = cv2.cvtColor(cv2.resize(prev_frame, (scale_w, scale_h)), cv2.COLOR_BGR2GRAY)

        fps = cap.get(cv2.CAP_PROP_FPS)
        frame_count = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))
        duration = frame_count / fps
        num_windows = max(1, int(duration / stride_sec))
        scores = np.zeros(num_windows)

        frame_idx = 1
        log_interval = max(1, frame_count // 50)
        frame_skip = max(1, int(fps // 2))  # ~2 frames per second

        while True:
            ret, frame = cap.read()
            if not ret:
                break

            if frame_idx % frame_skip != 0:
                frame_idx += 1
                continue

            gray = cv2.cvtColor(cv2.resize(frame, (scale_w, scale_h)), cv2.COLOR_BGR2GRAY)
            motion = np.mean(np.abs(gray.astype(np.float32) - prev_gray.astype(np.float32)))
            window_idx = int(frame_idx / fps / stride_sec)
            if window_idx < len(scores):
                scores[window_idx] += motion

            prev_gray = gray
            frame_idx += 1

            if frame_idx % log_interval == 0:
                self.log(f"[Visual][Motion] Processed frame {frame_idx}/{frame_count} ({frame_idx/frame_count:.0%})")

        cap.release()
        self.log("[Visual][Motion] Done calculating motion")
        scores = (scores - np.min(scores)) / (np.ptp(scores) + 1e-6)
        return scores

    # --------------------
    # Main Suggest Clips
    # --------------------
    def suggest_clips(self, input_file, window_sec=20, stride_sec=5, sr=16000, max_clips=5, allowed_overlap_sec=2):
        self.log(f"[Analysis] Starting analysis of {input_file}")

        # --- extract audio ---
        with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp_audio:
            audio_path = tmp_audio.name
        subprocess.run([
            "ffmpeg", "-y", "-i", str(input_file), "-vn", "-ac", "1", "-ar", str(sr), audio_path
        ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

        y, sr = librosa.load(audio_path, sr=sr)
        os.remove(audio_path)

        frame_length = int(0.5 * sr)
        hop_length = int(0.25 * sr)

        # --- audio features ---
        rms = librosa.feature.rms(y=y, frame_length=frame_length, hop_length=hop_length)[0]
        zcr = librosa.feature.zero_crossing_rate(y, frame_length=frame_length, hop_length=hop_length)[0]
        spec_cent = librosa.feature.spectral_centroid(y=y, sr=sr, n_fft=frame_length, hop_length=hop_length)[0]
        spec_bw = librosa.feature.spectral_bandwidth(y=y, sr=sr, n_fft=frame_length, hop_length=hop_length)[0]
        mfccs = librosa.feature.mfcc(y=y, sr=sr, n_mfcc=13, hop_length=hop_length)
        mfcc_var = np.mean(np.var(mfccs, axis=0))
        rms_diff = np.abs(np.diff(rms, prepend=rms[0]))
        rms_diff_n = (rms_diff - np.min(rms_diff)) / (np.ptp(rms_diff)+1e-6)
        tempo, beats = librosa.beat.beat_track(y=y, sr=sr)
        beat_density = len(beats) / (librosa.get_duration(y=y, sr=sr) + 1e-6)

        def normalize(arr):
            return (arr - np.min(arr)) / (np.ptp(arr)+1e-6)
        rms_n = normalize(rms)
        zcr_n = normalize(zcr)
        spec_cent_n = normalize(spec_cent)
        spec_bw_n = normalize(spec_bw)
        mfcc_var_n = mfcc_var / (np.max(mfccs)+1e-6)
        beat_density_n = beat_density

        duration = librosa.get_duration(y=y, sr=sr)

        # --- visual heuristics ---
        scene_scores = self._calc_scene_change_scores(input_file, window_sec, stride_sec)
        motion_scores = self._calc_motion_scores(input_file, window_sec, stride_sec)

        # --- windowing + combined score ---
        windows = []
        for w_idx, start in enumerate(np.arange(0, duration - window_sec, stride_sec)):
            end = start + window_sec
            i0 = int(start * sr / hop_length)
            i1 = int(end * sr / hop_length)

            mean_rms = float(np.mean(rms_n[i0:i1]))
            mean_zcr = float(np.mean(zcr_n[i0:i1]))
            mean_spec_cent = float(np.mean(spec_cent_n[i0:i1]))
            mean_spec_bw = float(np.mean(spec_bw_n[i0:i1]))
            mean_rms_diff = float(np.mean(rms_diff_n[i0:i1]))

            silence_penalty = 0.0 if mean_rms > 0.02 else -0.2

            scene_score = float(scene_scores[min(w_idx, len(scene_scores)-1)])
            motion_score = float(motion_scores[min(w_idx, len(motion_scores)-1)])

            score = (
                0.20 * mean_rms +
                0.10 * mean_zcr +
                0.10 * mean_spec_cent +
                0.10 * mean_spec_bw +
                0.10 * mfcc_var_n +
                0.05 * beat_density_n +
                0.10 * mean_rms_diff +
                0.15 * scene_score +
                0.10 * motion_score +
                silence_penalty
            )
            windows.append((int(start*1000), int(end*1000), score))
            self.log(f"[Suggested Window] {start:.2f}-{end:.2f}s | Score: {score:.4f}")

        # --- pick top non-overlapping clips (allow small overlap) ---
        windows.sort(key=lambda x: x[2], reverse=True)
        top = []
        used_intervals = []

        for start_ms, end_ms, score in windows:
            overlap = False
            for u_start, u_end in used_intervals:
                # allow small overlap
                if not (end_ms <= u_start - allowed_overlap_sec*1000 or start_ms >= u_end + allowed_overlap_sec*1000):
                    overlap = True
                    break
            if not overlap:
                top.append((start_ms, end_ms, score))
                used_intervals.append((start_ms, end_ms))
            if len(top) >= max_clips:
                break

        self.log(f"[Analysis] Top {len(top)} suggested clips ready (non-overlapping with {allowed_overlap_sec}s tolerance)")
        return top
