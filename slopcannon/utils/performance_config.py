"""
Performance configuration for slopCannon.
Provides tunable parameters for multi-threaded processing and optimization.
"""
import os
from dataclasses import dataclass
from typing import Optional


@dataclass
class PerformanceConfig:
    """Configuration for performance-related settings."""
    
    # Clip export settings
    max_export_workers: int = None  # Number of parallel clip exports (None = auto-detect)
    
    # Video analysis settings
    analysis_frame_skip: int = None  # Frames to skip during analysis (None = auto-calculate based on FPS)
    max_analysis_workers: int = 2  # Parallel workers for different analysis tasks
    
    # Whisper model settings
    whisper_device: str = "cpu"  # "cpu" or "cuda"
    whisper_compute_type: str = "int8"  # "int8", "int16", or "float32"
    
    # FFmpeg settings
    ffmpeg_preset: str = "ultrafast"  # FFmpeg encoding preset (ultrafast, fast, medium, slow)
    ffmpeg_crf: int = 23  # Constant Rate Factor (18-28, lower = better quality)
    
    def __post_init__(self):
        """Auto-configure based on system resources."""
        if self.max_export_workers is None:
            # Default to CPU count, capped at 4 to avoid overwhelming system
            self.max_export_workers = min(os.cpu_count() or 1, 4)
    
    @classmethod
    def from_env(cls) -> "PerformanceConfig":
        """Create configuration from environment variables."""
        return cls(
            max_export_workers=int(os.getenv("SLOP_MAX_EXPORT_WORKERS", "0")) or None,
            analysis_frame_skip=int(os.getenv("SLOP_FRAME_SKIP", "0")) or None,
            max_analysis_workers=int(os.getenv("SLOP_MAX_ANALYSIS_WORKERS", "2")),
            whisper_device=os.getenv("SLOP_WHISPER_DEVICE", "cpu"),
            whisper_compute_type=os.getenv("SLOP_WHISPER_COMPUTE", "int8"),
            ffmpeg_preset=os.getenv("SLOP_FFMPEG_PRESET", "ultrafast"),
            ffmpeg_crf=int(os.getenv("SLOP_FFMPEG_CRF", "23")),
        )
    
    def get_export_workers(self) -> int:
        """Get the number of workers for clip export."""
        return self.max_export_workers
    
    def get_analysis_workers(self) -> int:
        """Get the number of workers for video analysis."""
        return self.max_analysis_workers
    
    def get_frame_skip(self, fps: float) -> int:
        """Calculate frame skip based on FPS or use configured value."""
        if self.analysis_frame_skip is not None:
            return self.analysis_frame_skip
        # Default: skip frames to achieve ~2 samples per second
        return max(1, int(fps // 2))
