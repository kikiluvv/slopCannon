from PyQt5.QtWidgets import (
    QMainWindow, QWidget, QVBoxLayout, QPushButton, QSlider, QLabel, QFileDialog
)
from PyQt5.QtCore import Qt, QTimer, QUrl
from PyQt5.QtMultimedia import QMediaPlayer, QMediaContent
from PyQt5.QtMultimediaWidgets import QVideoWidget
from pathlib import Path
import subprocess
from slopcannon.managers.clip_manager import ClipManager
from slopcannon.utils.ffmpeg_wrapper import FFmpegWrapper


class MainWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.setWindowTitle("slopCannon")
        self.setGeometry(100, 100, 480, 800)  # portrait mode

        self.clip_manager = ClipManager()
        self.loaded_video_path = None
        self.ffmpeg = FFmpegWrapper()

        # central widget
        central = QWidget()
        self.setCentralWidget(central)
        layout = QVBoxLayout()
        central.setLayout(layout)

        # video player
        self.video_widget = QVideoWidget()
        layout.addWidget(self.video_widget)
        self.player = QMediaPlayer()
        self.player.setVideoOutput(self.video_widget)

        # slider + label
        self.slider = QSlider(Qt.Horizontal)
        self.slider.setRange(0, 1000)
        layout.addWidget(self.slider)
        self.time_label = QLabel("00:00 / 00:00")
        layout.addWidget(self.time_label)

        # buttons
        self.load_btn = QPushButton("Load Video")
        self.play_btn = QPushButton("Play")
        self.pause_btn = QPushButton("Pause")
        self.mark_start_btn = QPushButton("Mark Start")
        self.mark_end_btn = QPushButton("Mark End")
        self.export_btn = QPushButton("Export Clips")

        for btn in [
            self.load_btn, self.play_btn, self.pause_btn,
            self.mark_start_btn, self.mark_end_btn, self.export_btn
        ]:
            layout.addWidget(btn)

        # connect signals
        self.load_btn.clicked.connect(self.load_video)
        self.play_btn.clicked.connect(self.player.play)
        self.pause_btn.clicked.connect(self.player.pause)
        self.mark_start_btn.clicked.connect(self.mark_start)
        self.mark_end_btn.clicked.connect(self.mark_end)
        self.export_btn.clicked.connect(self.export_clips)
        self.slider.sliderMoved.connect(self.scrub)

        # update UI every 100ms
        self.timer = QTimer()
        self.timer.setInterval(100)
        self.timer.timeout.connect(self.update_ui)
        self.timer.start()

    # --------------------
    # Video Methods
    # --------------------
    def load_video(self):
        file_path, _ = QFileDialog.getOpenFileName(
            self, "Open Video", str(Path.home()), "Videos (*.mp4 *.mov *.mkv)"
        )
        if file_path:
            self.loaded_video_path = Path(file_path)
            url = QUrl.fromLocalFile(file_path)
            self.player.setMedia(QMediaContent(url))
            self.player.play()
            self.player.pause()

    def scrub(self, value):
        if self.player.duration() > 0:
            new_pos = int(value / 1000 * self.player.duration())
            self.player.setPosition(new_pos)

    def update_ui(self):
        if self.player.duration() > 0:
            pos = self.player.position()
            dur = self.player.duration()
            self.slider.blockSignals(True)
            self.slider.setValue(int(pos / dur * 1000))
            self.slider.blockSignals(False)
            self.time_label.setText(f"{self.format_ms(pos)} / {self.format_ms(dur)}")

    # --------------------
    # Clip Methods
    # --------------------
    def mark_start(self):
        pos = self.player.position()
        self.clip_manager.mark_start(pos)
        print(f"Start marked at {self.format_ms(pos)}")

    def mark_end(self):
        pos = self.player.position()
        try:
            self.clip_manager.mark_end(pos)
            print(f"End marked at {self.format_ms(pos)}")
            print("Current clips:", self.clip_manager.get_clips())
        except ValueError as e:
            print(f"Error marking clip: {e}")

    # --------------------
    # Export
    # --------------------
    def export_clips(self):
        if not self.clip_manager.get_clips():
            print("No clips to export!")
            return
        if not self.loaded_video_path:
            print("No video loaded!")
            return

        output_dir = QFileDialog.getExistingDirectory(
            self, "Select output directory", str(Path.home())
        )
        if not output_dir:
            return

        for idx, (start, end) in enumerate(self.clip_manager.get_clips(), start=1):
            out_file = Path(output_dir) / f"clip_{idx}.mp4"
            try:
                final_file = self.ffmpeg.export_clip(
                    self.loaded_video_path,
                    start,
                    end,
                    out_file,
                    portrait=True,
                    overlay=True,
                    subtitles=True,  # 👈 burn subs
                )
                print(f"✅ Exported with subs: {final_file}")
            except subprocess.CalledProcessError as e:
                print(f"❌ Error exporting clip {idx}: {e}")

    @staticmethod
    def format_ms(ms):
        s = ms // 1000
        m, s = divmod(s, 60)
        return f"{m:02}:{s:02}"
