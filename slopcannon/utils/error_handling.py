"""
Error handling and retry utilities for robust operation.
Provides retry logic with exponential backoff for transient failures.
"""
import time
import functools
from typing import Callable, Optional, Tuple, Type
import subprocess


class RetryConfig:
    """Configuration for retry behavior."""
    
    def __init__(self, 
                 max_attempts: int = 3,
                 initial_delay: float = 1.0,
                 max_delay: float = 30.0,
                 exponential_base: float = 2.0,
                 jitter: bool = True):
        self.max_attempts = max_attempts
        self.initial_delay = initial_delay
        self.max_delay = max_delay
        self.exponential_base = exponential_base
        self.jitter = jitter
    
    def get_delay(self, attempt: int) -> float:
        """Calculate delay for given attempt number."""
        delay = min(self.initial_delay * (self.exponential_base ** attempt), self.max_delay)
        if self.jitter:
            import random
            delay *= (0.5 + random.random())  # Add 0-50% jitter
        return delay


def with_retry(config: Optional[RetryConfig] = None,
              exceptions: Tuple[Type[Exception], ...] = (subprocess.CalledProcessError, IOError),
              log_callback: Optional[Callable] = None):
    """
    Decorator to add retry logic to functions.
    
    Args:
        config: Retry configuration (uses defaults if None)
        exceptions: Tuple of exception types to retry on
        log_callback: Optional callback for logging retry attempts
    """
    if config is None:
        config = RetryConfig()
    
    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            last_exception = None
            
            for attempt in range(config.max_attempts):
                try:
                    return func(*args, **kwargs)
                except exceptions as e:
                    last_exception = e
                    
                    if attempt < config.max_attempts - 1:
                        delay = config.get_delay(attempt)
                        msg = f"[Retry] Attempt {attempt + 1}/{config.max_attempts} failed: {e}. Retrying in {delay:.1f}s..."
                        
                        if log_callback:
                            log_callback(msg)
                        
                        time.sleep(delay)
                    else:
                        msg = f"[Retry] All {config.max_attempts} attempts failed for {func.__name__}"
                        if log_callback:
                            log_callback(msg)
            
            # Re-raise the last exception after all retries exhausted
            raise last_exception
        
        return wrapper
    return decorator


class ErrorRecovery:
    """Handles error recovery strategies for different failure modes."""
    
    @staticmethod
    def recover_from_ffmpeg_error(error: subprocess.CalledProcessError, 
                                  cmd: list,
                                  log_callback: Optional[Callable] = None) -> Optional[list]:
        """
        Attempt to recover from FFmpeg errors by adjusting command parameters.
        
        Returns:
            Modified command if recovery is possible, None otherwise
        """
        stderr = error.stderr if hasattr(error, 'stderr') else ""
        
        # Check for common recoverable errors
        if "Conversion failed!" in stderr or "Invalid data" in stderr:
            # Try with different encoding parameters
            if "-preset" in cmd:
                preset_idx = cmd.index("-preset") + 1
                if cmd[preset_idx] == "ultrafast":
                    cmd[preset_idx] = "fast"
                    if log_callback:
                        log_callback("[Recovery] Retrying with 'fast' preset instead of 'ultrafast'")
                    return cmd
        
        if "Output file is empty" in stderr:
            # Try removing copy codec and force re-encode
            if "-c" in cmd and "copy" in cmd:
                copy_idx = cmd.index("copy")
                cmd[copy_idx] = "libx264"
                if log_callback:
                    log_callback("[Recovery] Retrying with re-encoding instead of stream copy")
                return cmd
        
        return None
    
    @staticmethod
    def handle_disk_space_error(required_space: int, 
                                log_callback: Optional[Callable] = None) -> bool:
        """
        Check if there's sufficient disk space for operation.
        
        Args:
            required_space: Required space in bytes
            log_callback: Optional logging callback
        
        Returns:
            True if sufficient space available, False otherwise
        """
        import shutil
        
        try:
            stat = shutil.disk_usage(".")
            available = stat.free
            
            if available < required_space:
                msg = f"[Error] Insufficient disk space. Required: {required_space / (1024**3):.2f}GB, Available: {available / (1024**3):.2f}GB"
                if log_callback:
                    log_callback(msg)
                return False
            
            return True
        except Exception as e:
            if log_callback:
                log_callback(f"[Warning] Could not check disk space: {e}")
            return True  # Assume OK if check fails
    
    @staticmethod
    def handle_model_load_error(model_size: str, 
                                log_callback: Optional[Callable] = None) -> Optional[str]:
        """
        Fallback to smaller model if loading fails.
        
        Args:
            model_size: Current model size that failed
            log_callback: Optional logging callback
        
        Returns:
            Smaller model size to try, or None if no fallback available
        """
        model_hierarchy = ["large-v3", "large-v2", "large", "medium", "small", "base", "tiny"]
        
        try:
            current_idx = model_hierarchy.index(model_size)
            if current_idx < len(model_hierarchy) - 1:
                fallback = model_hierarchy[current_idx + 1]
                if log_callback:
                    log_callback(f"[Recovery] Model '{model_size}' failed to load. Falling back to '{fallback}'")
                return fallback
        except ValueError:
            pass
        
        if log_callback:
            log_callback(f"[Error] No smaller model available than '{model_size}'")
        return None
