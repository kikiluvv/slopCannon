# log_panel.py
from PyQt5.QtWidgets import QWidget, QVBoxLayout, QTextEdit
from PyQt5.QtCore import QTimer
from collections import deque

class LogPanel(QWidget):
    def __init__(self, parent=None):
        super().__init__(parent)
        self.layout = QVBoxLayout()
        self.setLayout(self.layout)

        self.text_edit = QTextEdit()
        self.text_edit.setReadOnly(True)
        self.text_edit.setStyleSheet("""
            background-color: #111;
            color: #eee;
            font-family: Consolas, monospace;
            font-size: 12px;
        """)
        self.layout.addWidget(self.text_edit)

        # queue for batched updates
        self.queue = deque()
        self.timer = QTimer()
        self.timer.setInterval(50)  # flush every 50ms
        self.timer.timeout.connect(self.flush)
        self.timer.start()

    def append(self, message: str):
        self.queue.append(message)

    def flush(self):
        if self.queue:
            self.text_edit.setUpdatesEnabled(False)
            while self.queue:
                self.text_edit.append(self.queue.popleft())
            self.text_edit.verticalScrollBar().setValue(
                self.text_edit.verticalScrollBar().maximum()
            )
            self.text_edit.setUpdatesEnabled(True)
