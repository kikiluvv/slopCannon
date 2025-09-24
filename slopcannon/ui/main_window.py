from PyQt5.QtWidgets import (
    QMainWindow, QWidget, QVBoxLayout, QHBoxLayout, QPushButton, QSlider, QLabel,
    QFileDialog, QDockWidget, QListWidget, QInputDialog, QMessageBox, QDialog
)
from PyQt5.QtCore import Qt, QTimer, QUrl
from PyQt5.QtMultimedia import QMediaPlayer, QMediaContent
from PyQt5.QtMultimediaWidgets import QVideoWidget
from pathlib import Path
import subprocess
import sys

from slopcannon.managers.clip_manager import ClipManager
from slopcannon.utils.ffmpeg_wrapper import FFmpegWrapper
from slopcannon.utils.settings import SubtitleSettings
from slopcannon.ui.settings_panel import SettingsPanel
from slopcannon.ui.log_panel import LogPanel
from slopcannon.utils.log_stream import EmittingStream
from slopcannon.managers.analysis_manager import AnalysisManager


class MainWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.setWindowTitle("💣 slopCannon 💣")
        self.setGeometry(100, 100, 800, 800)

        # --------------------
        # Managers
        # --------------------
        self.clip_manager = ClipManager()
        self.loaded_video_path = None
        self.subtitle_settings = SubtitleSettings()

        # --------------------
        # Central Widget
        # --------------------
        central = QWidget()
        self.setCentralWidget(central)
        main_layout = QVBoxLayout()
        main_layout.setSpacing(15)
        main_layout.setContentsMargins(15, 15, 15, 15)
        central.setLayout(main_layout)

        # --------------------
        # Load Video
        # --------------------
        self.load_btn = QPushButton("Load Video")
        self.apply_button_style(self.load_btn)
        main_layout.addWidget(self.load_btn)

        # --------------------
        # Video Player
        # --------------------
        self.video_widget = QVideoWidget()
        self.video_widget.setStyleSheet("border: 1px solid #555; border-radius: 5px;")
        main_layout.addWidget(self.video_widget)
        self.player = QMediaPlayer()
        self.player.setVideoOutput(self.video_widget)

        # --------------------
        # Playback Controls
        # --------------------
        playback_layout = QHBoxLayout()
        playback_layout.setSpacing(10)

        self.play_btn = QPushButton("Play")
        self.pause_btn = QPushButton("Pause")
        self.slider = QSlider(Qt.Horizontal)
        self.slider.setRange(0, 1000)
        self.time_label = QLabel("00:00 / 00:00")

        for btn in [self.play_btn, self.pause_btn]:
            self.apply_button_style(btn)

        self.slider.setStyleSheet("""
            QSlider::groove:horizontal { height: 6px; background: #ccc; border-radius: 3px; }
            QSlider::handle:horizontal { background: #555; width: 14px; margin: -4px 0; border-radius: 7px; }
        """)
        playback_layout.addWidget(self.play_btn)
        playback_layout.addWidget(self.pause_btn)
        playback_layout.addWidget(self.slider)
        playback_layout.addWidget(self.time_label)
        main_layout.addLayout(playback_layout)

        # --------------------
        # Clip Control
        # --------------------
        clip_layout = QHBoxLayout()
        clip_layout.setSpacing(10)

        self.analyze_btn = QPushButton("Find Viral Clips")
        self.apply_button_style(self.analyze_btn)
        clip_layout.addWidget(self.analyze_btn)
        self.mark_start_btn = QPushButton("Mark Start")
        self.mark_end_btn = QPushButton("Mark End")
        self.export_btn = QPushButton("Export Clips")

        for btn in [self.mark_start_btn, self.mark_end_btn, self.export_btn]:
            btn.setMinimumWidth(120)
            self.apply_button_style(btn)
            clip_layout.addWidget(btn)

        main_layout.addLayout(clip_layout)


        # --------------------
        # Signals
        # --------------------
        self.load_btn.clicked.connect(self.load_video)
        self.play_btn.clicked.connect(self.player.play)
        self.pause_btn.clicked.connect(self.player.pause)
        self.mark_start_btn.clicked.connect(self.mark_start)
        self.mark_end_btn.clicked.connect(self.mark_end)
        self.export_btn.clicked.connect(self.export_clips)
        self.slider.sliderMoved.connect(self.scrub)
        self.analyze_btn.clicked.connect(self.analyze_video)

        # --------------------
        # Timer to update UI
        # --------------------
        self.timer = QTimer()
        self.timer.setInterval(100)
        self.timer.timeout.connect(self.update_ui)
        self.timer.start()

        # --------------------
        # Subtitle Settings Dock
        # --------------------
        self.settings_dock = QDockWidget("Subtitle Settings", self)
        self.settings_dock.setAllowedAreas(Qt.RightDockWidgetArea | Qt.LeftDockWidgetArea)
        self.settings_panel = SettingsPanel(self.subtitle_settings)
        self.settings_panel.settings_changed.connect(self.apply_settings)
        self.settings_dock.setWidget(self.settings_panel)
        self.addDockWidget(Qt.RightDockWidgetArea, self.settings_dock)

        # --------------------
        # Log Panel Dock
        # --------------------
        self.log_dock = QDockWidget("Processing Log", self)
        self.log_dock.setAllowedAreas(Qt.BottomDockWidgetArea | Qt.TopDockWidgetArea)
        self.log_panel = LogPanel()
        self.log_dock.setWidget(self.log_panel)
        self.addDockWidget(Qt.BottomDockWidgetArea, self.log_dock)

        # redirect stdout/stderr to log panel
        sys.stdout = EmittingStream(self.log_panel.append)
        sys.stderr = EmittingStream(self.log_panel.append)

        self.ffmpeg = FFmpegWrapper(log_callback=self.log_panel.append)
        self.analysis_manager = AnalysisManager(log_callback=self.log_panel.append)
        
        # --------------------
        # Clip Manager Button
        # --------------------
        self.manage_clips_btn = QPushButton("Manage Clips")
        self.apply_button_style(self.manage_clips_btn)
        clip_layout.addWidget(self.manage_clips_btn)
        self.manage_clips_btn.clicked.connect(self.open_clip_manager_panel)
        
    # --------------------
    # Video Methods
    # --------------------
    def load_video(self):
        file_path, _ = QFileDialog.getOpenFileName(
            self, "Open Video", str(Path.home()), "Videos (*.mp4 *.mov *.mkv)"
        )
        if file_path:
            self.loaded_video_path = Path(file_path)
            self.player.setMedia(QMediaContent(QUrl.fromLocalFile(file_path)))
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
            
    def analyze_video(self):
        if not self.loaded_video_path:
            self.log_panel.append("❌ No video loaded for analysis")
            return

        self.log_panel.append("🔍 Running viral clip analysis...")
        suggestions = self.analysis_manager.suggest_clips(self.loaded_video_path)

        if not suggestions:
            self.log_panel.append("❌ No clips suggested.")
            return

        # inject into ClipManager
        for (start, end, score) in suggestions:
            self.clip_manager.add_clip(start, end, score)

        self.log_panel.append(f"✅ Added {len(suggestions)} suggested clips")
        self.log_panel.append(f"Current clips: {self.clip_manager.get_clips()}")


    # --------------------
    # Clip Methods
    # --------------------
    def mark_start(self):
        pos = self.player.position()
        self.clip_manager.mark_start(pos)
        self.log_panel.append(f"Start marked at {self.format_ms(pos)}")

    def mark_end(self):
        pos = self.player.position()
        try:
            self.clip_manager.mark_end(pos)
            self.log_panel.append(f"End marked at {self.format_ms(pos)}")
            self.log_panel.append(f"Current clips: {self.clip_manager.get_clips()}")
        except ValueError as e:
            self.log_panel.append(f"❌ Error marking clip: {e}")

    # --------------------
    # Export
    # --------------------
    def export_clips(self):
        if not self.clip_manager.get_clips():
            self.log_panel.append("❌ No clips to export!")
            return
        if not self.loaded_video_path:
            self.log_panel.append("❌ No video loaded!")
            return

        output_dir = QFileDialog.getExistingDirectory(
            self, "Select output directory", str(Path.home())
        )
        if not output_dir:
            return

        clips = list(self.clip_manager.get_clips())

        self.log_panel.append(f"Starting export of {len(clips)} clip(s)...")

        # helper callback for each clip
        def clip_done(final_file, error=None):
            if error:
                self.log_panel.append(f"❌ Clip export failed: {error}")
            else:
                self.log_panel.append(f"✅ Clip export finished: {final_file}")

        # submit all clips to FFmpegWrapper
        for idx, (start, end, score) in enumerate(clips, start=1):
            out_file = Path(output_dir) / f"clip_{idx}.mp4"
            self.log_panel.append(f"[{idx}/{len(clips)}] Exporting clip {idx} (score={score:.2f})...")
            self.ffmpeg.export_clip(
                input_file=self.loaded_video_path,
                start_ms=start,
                end_ms=end,
                output_file=out_file,
                portrait=True,
                overlay=True,
                subtitles=True,
                subtitle_settings=self.subtitle_settings,
                callback=clip_done
            )

    # --------------------
    # Settings Handler
    # --------------------
    def apply_settings(self, settings: SubtitleSettings):
        self.subtitle_settings = settings
        self.log_panel.append(f"✅ Subtitle settings updated: {vars(settings)}")
    
    # --------------------
    # Clip Manager Panel
    # --------------------
    def open_clip_manager_panel(self):
        dlg = QDialog(self)
        dlg.setWindowTitle("Clip Manager")
        dlg.setMinimumWidth(450)  # wider for long text
        dlg.setModal(True)

        layout = QVBoxLayout()
        dlg.setLayout(layout)

        # clip list
        self.clip_list_widget = QListWidget()
        self.clip_list_widget.setSelectionMode(QListWidget.SingleSelection)
        for idx, (s, e, score) in enumerate(self.clip_manager.get_clips()):
            self.clip_list_widget.addItem(f"[{idx}] {self.format_ms(s)} → {self.format_ms(e)} | Score: {score:.2f}")
        layout.addWidget(self.clip_list_widget)

        # buttons row
        btn_layout = QHBoxLayout()
        edit_btn = QPushButton("Edit Clip")
        delete_btn = QPushButton("Delete Clip")
        close_btn = QPushButton("Close")

        for btn in [edit_btn, delete_btn, close_btn]:
            self.apply_button_style(btn)
            btn_layout.addWidget(btn)

        layout.addLayout(btn_layout)

        # signals
        edit_btn.clicked.connect(lambda: self.edit_clip_in_dialog())
        delete_btn.clicked.connect(lambda: self.delete_clip_in_dialog())
        close_btn.clicked.connect(dlg.accept)

        dlg.exec_()
        
        # --- log updated clips ---
        self.log_panel.append(f"Updated clips: {self.clip_manager.get_clips()}")

    # --------------------
    # Dialog Helpers
    # --------------------
    def delete_clip_in_dialog(self):
        idx = self.clip_list_widget.currentRow()
        if idx < 0:
            return
        confirm = QMessageBox.question(
            self, "Delete Clip", "Are you sure you want to delete this clip?",
            QMessageBox.Yes | QMessageBox.No
        )
        if confirm == QMessageBox.Yes:
            self.clip_manager.remove_clip(idx)
            self.clip_list_widget.takeItem(idx)

    def edit_clip_in_dialog(self):
        idx = self.clip_list_widget.currentRow()
        if idx < 0:
            return
        s, e, score = self.clip_manager.get_clips()[idx]

        new_start, ok = QInputDialog.getInt(self, "Edit Start", "Start ms:", s)
        if not ok: return
        new_end, ok = QInputDialog.getInt(self, "Edit End", "End ms:", e)
        if not ok: return
        new_score, ok = QInputDialog.getDouble(self, "Edit Score", "Score:", score, 0, 10, 2)
        if not ok: return

        try:
            self.clip_manager.update_clip(idx, new_start, new_end, new_score)
            self.clip_list_widget.item(idx).setText(
                f"[{idx}] {self.format_ms(new_start)} → {self.format_ms(new_end)} | Score: {new_score:.2f}"
            )
        except ValueError as ve:
            QMessageBox.warning(self, "❌ Invalid Clip", str(ve))

    # --------------------
    # Helpers
    # --------------------
    @staticmethod
    def format_ms(ms):
        s = ms // 1000
        m, s = divmod(s, 60)
        return f"{m:02}:{s:02}"

    # --------------------
    # Button Styling
    # --------------------
    @staticmethod
    def apply_button_style(btn: QPushButton):
        btn.setStyleSheet("""
            QPushButton {
                background-color: #444;
                color: #fff;
                border-radius: 6px;
                padding: 6px 12px;
            }
            QPushButton:hover {
                background-color: #666;
            }
            QPushButton:pressed {
                background-color: #222;
            }
        """)
