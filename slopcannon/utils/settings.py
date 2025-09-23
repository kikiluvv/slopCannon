# slopcannon/utils/settings.py

class SubtitleSettings:
    def __init__(
        self,
        words_per_line=5,
        lines_per_event=2,
        fade_ms=100,
        font="Comic Sans MS",
        font_size=72,
        primary_color="&H00FFFFFF",
        secondary_color="&H0000FFFF",
        outline_color="&H00000000",
        back_color="&H64000000",
        model_size="small",
    ):
        self.words_per_line = words_per_line
        self.lines_per_event = lines_per_event
        self.fade_ms = fade_ms
        self.font = font
        self.font_size = font_size
        self.primary_color = primary_color
        self.secondary_color = secondary_color
        self.outline_color = outline_color
        self.back_color = back_color
        self.model_size = model_size
