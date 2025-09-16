class ClipManager:
    """
    Stores multiple clips for a single video.
    Each clip is a tuple: (start_ms, end_ms)
    """
    def __init__(self):
        self.clips = []
        self.current_start = None

    def mark_start(self, position_ms):
        self.current_start = position_ms

    def mark_end(self, position_ms):
        if self.current_start is None:
            raise ValueError("Start not set before end")
        if position_ms <= self.current_start:
            raise ValueError("End must be after start")
        self.clips.append((self.current_start, position_ms))
        self.current_start = None

    def clear_clips(self):
        self.clips = []
        self.current_start = None

    def get_clips(self):
        return self.clips
