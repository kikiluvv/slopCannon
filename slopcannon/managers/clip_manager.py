class ClipManager:
    """
    Stores multiple clips for a single video.
    Each clip is a tuple: (start_ms, end_ms, score)
    """
    def __init__(self):
        self.clips = []
        self.current_start = None

    def mark_start(self, position_ms):
        self.current_start = position_ms

    def mark_end(self, position_ms, score=1.0):
        if self.current_start is None:
            raise ValueError("Start not set before end")
        if position_ms <= self.current_start:
            raise ValueError("End must be after start")
        self.clips.append((self.current_start, position_ms, score))
        self.current_start = None

    def add_clip(self, start_ms, end_ms, score=1.0):
        """Directly add a clip (used by AnalysisManager suggestions)."""
        if end_ms > start_ms:
            self.clips.append((start_ms, end_ms, score))

    def clear_clips(self):
        self.clips = []
        self.current_start = None

    def get_clips(self):
        return self.clips
