from PyQt5.QtCore import QObject, QThread, pyqtSignal

class AnalysisWorker(QObject):
    progress = pyqtSignal(str)  # logs
    finished = pyqtSignal(list)  # resulting top clips

    def __init__(self, analysis_manager, video_path):
        super().__init__()
        self.manager = analysis_manager
        self.video_path = video_path

    def run(self):
        # temporarily redirect AnalysisManager log to our signal
        orig_log = self.manager.log
        self.manager.log = lambda msg: self.progress.emit(msg)

        try:
            top_clips = self.manager.suggest_clips(self.video_path)
        finally:
            self.manager.log = orig_log

        self.finished.emit(top_clips)
