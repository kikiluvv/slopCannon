import sys

class EmittingStream:
    def __init__(self, callback):
        self.callback = callback

    def write(self, text):
        if text.strip():
            self.callback(text.strip())

    def flush(self):
        pass
