"""
Progress tracking utilities for long-running operations.
Provides real-time progress updates for export and analysis operations.
"""
from PyQt5.QtCore import QObject, pyqtSignal
from typing import Optional, Callable
import time


class ProgressTracker(QObject):
    """Tracks progress of operations and emits signals for UI updates."""
    
    # Signals
    progress_update = pyqtSignal(int, int, str)  # current, total, message
    progress_percent = pyqtSignal(int, str)  # percentage, message
    operation_complete = pyqtSignal(str)  # message
    operation_failed = pyqtSignal(str)  # error message
    
    def __init__(self, operation_name: str, total_items: int, log_callback: Optional[Callable] = None):
        super().__init__()
        self.operation_name = operation_name
        self.total_items = total_items
        self.current_item = 0
        self.log_callback = log_callback or print
        self.start_time = time.time()
        self.completed_items = []
        self.failed_items = []
    
    def update(self, increment: int = 1, message: str = ""):
        """Update progress by increment and emit signals."""
        self.current_item += increment
        percent = int((self.current_item / self.total_items) * 100) if self.total_items > 0 else 0
        
        elapsed = time.time() - self.start_time
        if self.current_item > 0:
            avg_time = elapsed / self.current_item
            remaining = avg_time * (self.total_items - self.current_item)
            eta_msg = f"ETA: {int(remaining)}s"
        else:
            eta_msg = "calculating..."
        
        full_message = f"[{self.operation_name}] {self.current_item}/{self.total_items} ({percent}%) - {message} - {eta_msg}"
        
        self.progress_update.emit(self.current_item, self.total_items, full_message)
        self.progress_percent.emit(percent, full_message)
        
        if self.log_callback:
            self.log_callback(full_message)
    
    def mark_completed(self, item_id: str):
        """Mark an item as successfully completed."""
        self.completed_items.append(item_id)
        self.update(message=f"Completed: {item_id}")
    
    def mark_failed(self, item_id: str, error: str):
        """Mark an item as failed."""
        self.failed_items.append((item_id, error))
        error_msg = f"Failed: {item_id} - {error}"
        self.update(message=error_msg)
        self.operation_failed.emit(error_msg)
    
    def complete(self):
        """Mark the entire operation as complete."""
        elapsed = time.time() - self.start_time
        success_count = len(self.completed_items)
        fail_count = len(self.failed_items)
        
        summary = (f"[{self.operation_name}] Complete! "
                  f"Success: {success_count}, Failed: {fail_count}, "
                  f"Time: {elapsed:.1f}s")
        
        self.operation_complete.emit(summary)
        if self.log_callback:
            self.log_callback(summary)
    
    def get_summary(self) -> dict:
        """Get summary of operation progress."""
        return {
            "operation": self.operation_name,
            "total": self.total_items,
            "current": self.current_item,
            "completed": len(self.completed_items),
            "failed": len(self.failed_items),
            "elapsed": time.time() - self.start_time,
            "percent": int((self.current_item / self.total_items) * 100) if self.total_items > 0 else 0
        }


class ExportProgressTracker(ProgressTracker):
    """Specialized progress tracker for clip export operations."""
    
    def __init__(self, num_clips: int, log_callback: Optional[Callable] = None):
        super().__init__("Export Clips", num_clips, log_callback)
        self.clips_info = {}
    
    def start_clip(self, clip_index: int, clip_info: tuple):
        """Mark start of clip export."""
        start_ms, end_ms, score = clip_info
        duration = (end_ms - start_ms) / 1000
        self.clips_info[clip_index] = {
            "start_ms": start_ms,
            "end_ms": end_ms,
            "duration": duration,
            "score": score
        }
        self.update(0, f"Starting clip {clip_index}: {duration:.1f}s, score={score:.2f}")
    
    def complete_clip(self, clip_index: int, output_file: str):
        """Mark clip export as complete."""
        self.mark_completed(f"Clip {clip_index}: {output_file}")


class AnalysisProgressTracker(ProgressTracker):
    """Specialized progress tracker for video analysis operations."""
    
    def __init__(self, video_path: str, log_callback: Optional[Callable] = None):
        # Total steps: audio extraction, audio features, scene analysis, motion analysis, windowing
        super().__init__(f"Analyze {video_path}", 5, log_callback)
        self.phase_names = [
            "Audio Extraction",
            "Audio Features",
            "Scene Analysis",
            "Motion Analysis",
            "Window Scoring"
        ]
        self.current_phase = 0
    
    def start_phase(self, phase: str):
        """Start a new analysis phase."""
        self.current_phase = self.phase_names.index(phase) if phase in self.phase_names else self.current_phase
        self.update(1, f"Phase: {phase}")
    
    def update_frame_progress(self, current_frame: int, total_frames: int, phase: str):
        """Update progress within a phase based on frame processing."""
        frame_percent = int((current_frame / total_frames) * 100) if total_frames > 0 else 0
        msg = f"{phase}: {current_frame}/{total_frames} frames ({frame_percent}%)"
        
        if self.log_callback:
            self.log_callback(f"[Analysis] {msg}")
