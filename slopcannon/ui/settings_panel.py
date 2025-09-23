# slopcannon/ui/settings_panel.py
from PyQt5.QtWidgets import (
    QWidget, QFormLayout, QLineEdit, QSpinBox, QPushButton, QComboBox, QColorDialog
)
from PyQt5.QtCore import pyqtSignal
from slopcannon.utils.settings import SubtitleSettings


class SettingsPanel(QWidget):
    settings_changed = pyqtSignal(SubtitleSettings)

    def __init__(self, settings: SubtitleSettings, parent=None):
        super().__init__(parent)
        self.settings = settings

        # --------------------
        # Layout
        # --------------------
        layout = QFormLayout()
        layout.setSpacing(10)
        layout.setContentsMargins(10, 10, 10, 10)
        self.setLayout(layout)

        # --------------------
        # Numeric Inputs
        # --------------------
        self.words_per_line = QSpinBox()
        self.words_per_line.setValue(settings.words_per_line)
        layout.addRow("Words per line:", self.words_per_line)

        self.lines_per_event = QSpinBox()
        self.lines_per_event.setValue(settings.lines_per_event)
        layout.addRow("Lines per event:", self.lines_per_event)

        self.fade_ms = QSpinBox()
        self.fade_ms.setMaximum(5000)
        self.fade_ms.setValue(settings.fade_ms)
        layout.addRow("Fade (ms):", self.fade_ms)

        self.font = QLineEdit(settings.font)
        layout.addRow("Font:", self.font)

        self.font_size = QSpinBox()
        self.font_size.setMaximum(200)
        self.font_size.setValue(settings.font_size)
        layout.addRow("Font size:", self.font_size)

        # --------------------
        # Color Buttons
        # --------------------
        self.primary_color_btn = QPushButton(settings.primary_color)
        self.primary_color_btn.clicked.connect(
            lambda: self.pick_color(self.primary_color_btn)
        )
        layout.addRow("Primary color:", self.primary_color_btn)

        self.secondary_color_btn = QPushButton(settings.secondary_color)
        self.secondary_color_btn.clicked.connect(
            lambda: self.pick_color(self.secondary_color_btn)
        )
        layout.addRow("Secondary color:", self.secondary_color_btn)

        self.outline_color_btn = QPushButton(settings.outline_color)
        self.outline_color_btn.clicked.connect(
            lambda: self.pick_color(self.outline_color_btn)
        )
        layout.addRow("Outline color:", self.outline_color_btn)

        self.back_color_btn = QPushButton(settings.back_color)
        self.back_color_btn.clicked.connect(
            lambda: self.pick_color(self.back_color_btn)
        )
        layout.addRow("Background color:", self.back_color_btn)

        # --------------------
        # Model Size Dropdown
        # --------------------
        self.model_size = QComboBox()
        self.model_size.addItems(["tiny", "small", "medium", "large"])
        self.model_size.setCurrentText(settings.model_size)
        layout.addRow("Model size:", self.model_size)

        # --------------------
        # Save Button
        # --------------------
        self.save_btn = QPushButton("Save Settings")
        self.apply_button_style(self.save_btn)
        layout.addRow(self.save_btn)
        self.save_btn.clicked.connect(self.save)

        # --------------------
        # Apply Styles to All Buttons
        # --------------------
        for btn in [
            self.primary_color_btn,
            self.secondary_color_btn,
            self.outline_color_btn,
            self.back_color_btn
        ]:
            self.apply_button_style(btn)

        # --------------------
        # Apply Dark/Transparent Style to Labels
        # --------------------
        self.style_labels(layout)

        # --------------------
        # Optional Background
        # --------------------
        self.setStyleSheet("background-color: #222; color: #eee;")

    # --------------------
    # Color Picker
    # --------------------
    def pick_color(self, btn: QPushButton):
        color = QColorDialog.getColor()
        if color.isValid():
            hex_color = color.name()  # "#RRGGBB"
            bgr = hex_color[5:7] + hex_color[3:5] + hex_color[1:3]
            ass_color = f"&H00{bgr.upper()}"
            btn.setText(ass_color)

    # --------------------
    # Save Settings
    # --------------------
    def save(self):
        self.settings.words_per_line = self.words_per_line.value()
        self.settings.lines_per_event = self.lines_per_event.value()
        self.settings.fade_ms = self.fade_ms.value()
        self.settings.font = self.font.text()
        self.settings.font_size = self.font_size.value()
        self.settings.primary_color = self.primary_color_btn.text()
        self.settings.secondary_color = self.secondary_color_btn.text()
        self.settings.outline_color = self.outline_color_btn.text()
        self.settings.back_color = self.back_color_btn.text()
        self.settings.model_size = self.model_size.currentText()

        self.settings_changed.emit(self.settings)

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

    # --------------------
    # Label Styling
    # --------------------
    @staticmethod
    def style_labels(layout: QFormLayout):
        for i in range(layout.rowCount()):
            label_item = layout.itemAt(i, QFormLayout.LabelRole)
            if label_item:
                widget = label_item.widget()
                if widget:
                    widget.setStyleSheet("background: transparent; color: #eee;")
