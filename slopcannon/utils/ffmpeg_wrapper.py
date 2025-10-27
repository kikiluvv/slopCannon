import subprocess
from pathlib import Path
from .subtitle_generator import SubtitleGenerator
from .error_handling import with_retry, RetryConfig, ErrorRecovery
from concurrent.futures import ThreadPoolExecutor
import os

class FFmpegWrapper:
    def __init__(self, ffmpeg_path=None, log_callback=print, max_workers=None, preset="ultrafast", crf=23):
        self.ffmpeg = str(ffmpeg_path) if ffmpeg_path else "ffmpeg"
        self.overlay_video = Path("assets/overlays/mc.mp4")
        self.log_callback = log_callback
        self.subs = SubtitleGenerator(log_callback=log_callback)
        self.preset = preset
        self.crf = crf
        self.retry_config = RetryConfig(max_attempts=2, initial_delay=2.0)
        # Default to CPU count for parallel processing, with a reasonable max
        if max_workers is None:
            max_workers = min(os.cpu_count() or 1, 4)
        self.executor = ThreadPoolExecutor(max_workers=max_workers)
        self.log_callback(f"[FFmpegWrapper] Initialized with {max_workers} worker(s), preset={preset}, crf={crf}")

    def run_cmd(self, cmd, allow_retry=True):
        self.log_callback(f"[FFmpeg] Running command: {' '.join(cmd)}")
        
        attempt = 0
        max_attempts = self.retry_config.max_attempts if allow_retry else 1
        last_error = None
        
        while attempt < max_attempts:
            try:
                process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
                stdout, stderr = process.communicate()

                if process.returncode != 0:
                    self.log_callback(f"[FFmpeg][ERROR] Command failed with code {process.returncode}")
                    if stdout:
                        self.log_callback(f"[FFmpeg][stdout]\n{stdout.strip()}")
                    if stderr:
                        self.log_callback(f"[FFmpeg][stderr]\n{stderr.strip()}")
                    
                    # Try error recovery
                    if allow_retry and attempt < max_attempts - 1:
                        recovered_cmd = ErrorRecovery.recover_from_ffmpeg_error(
                            subprocess.CalledProcessError(process.returncode, cmd, stderr=stderr),
                            cmd.copy(),
                            self.log_callback
                        )
                        
                        if recovered_cmd:
                            cmd = recovered_cmd
                            attempt += 1
                            self.log_callback(f"[FFmpeg] Retrying with modified command (attempt {attempt + 1}/{max_attempts})")
                            continue
                    
                    raise subprocess.CalledProcessError(process.returncode, cmd, stderr=stderr)

                self.log_callback(f"[FFmpeg] Command finished successfully with code {process.returncode}")
                return process.returncode
                
            except subprocess.CalledProcessError as e:
                last_error = e
                attempt += 1
                
                if attempt < max_attempts:
                    delay = self.retry_config.get_delay(attempt - 1)
                    self.log_callback(f"[FFmpeg] Attempt {attempt}/{max_attempts} failed. Retrying in {delay:.1f}s...")
                    import time
                    time.sleep(delay)
        
        # All retries exhausted
        raise last_error

    def export_clip(
        self,
        input_file,
        start_ms,
        end_ms,
        output_file,
        portrait=False,
        overlay=False,
        subtitles=False,
        subtitle_settings=None,
        callback=None
    ):
        def worker():
            temp_files = []
            try:
                start_sec = start_ms / 1000
                duration_sec = (end_ms - start_ms) / 1000
                self.log_callback(f"[FFmpeg] Exporting clip {input_file} → {output_file}")

                # base ffmpeg command
                cmd = [self.ffmpeg, "-y", "-i", str(input_file)]
                final_file = Path(output_file)
                
                if overlay:
                    self.log_callback(f"[FFmpeg] Adding overlay: {self.overlay_video}")
                    cmd += ["-stream_loop", "-1", "-i", str(self.overlay_video)]
                    filter_complex = "[0:v]scale=1080:960,setsar=1[v0];[1:v]scale=1080:960,setsar=1[v1];[v0][v1]vstack=inputs=2[outv]"
                    cmd += ["-filter_complex", filter_complex, "-map", "[outv]", "-map", "0:a?",
                            "-c:v", "libx264", "-preset", self.preset, "-crf", str(self.crf),
                            "-ss", str(start_sec), "-t", str(duration_sec), str(final_file)]
                else:
                    if portrait:
                        cmd += ["-vf", "scale=1080:1920:force_original_aspect_ratio=decrease,pad=1080:1920:(ow-iw)/2:(oh-ih)/2",
                                "-c:v", "libx264", "-preset", self.preset, "-crf", str(self.crf)]
                    else:
                        cmd += ["-c", "copy"]
                    cmd += ["-ss", str(start_sec), "-t", str(duration_sec), str(final_file)]

                self.run_cmd(cmd)
                temp_files.append(final_file)
                self.log_callback(f"[FFmpeg] Base clip generated: {final_file}")

                if subtitles:
                    ass_file = final_file.with_suffix(".ass")
                    audio_file = final_file.with_suffix(".wav")
                    temp_files.extend([ass_file, audio_file])

                    self.log_callback(f"[FFmpeg] Extracting audio for transcription: {audio_file}")
                    self.run_cmd([self.ffmpeg, "-y", "-i", str(final_file), "-vn", "-ac", "1", "-ar", "16000", str(audio_file)])

                    render_kwargs = {}
                    if subtitle_settings:
                        render_kwargs = {
                            k: v for k, v in vars(subtitle_settings).items()
                            if k in ["words_per_line","lines_per_event","fade_ms","font","font_size",
                                    "primary_color","secondary_color","outline_color","back_color"]
                        }
                        self.subs.update_model(model_size=subtitle_settings.model_size)

                    self.log_callback(f"[FFmpeg] Generating subtitles: {ass_file}")
                    self.subs.generate_subtitles(audio_file, ass_file, **render_kwargs)

                    sub_output = final_file.with_name(f"{final_file.stem}_sub.mp4")
                    self.log_callback(f"[FFmpeg] Burning subtitles into final clip: {sub_output}")
                    burn_cmd = [self.ffmpeg, "-y", "-i", str(final_file), "-vf", f"subtitles='{ass_file}'",
                                "-c:v", "libx264", "-preset", self.preset, "-crf", str(self.crf), "-c:a", "copy", str(sub_output)]
                    self.run_cmd(burn_cmd)
                    final_file = sub_output
                    temp_files.append(final_file)

                # cleanup temp files (everything except the final _sub.mp4)
                self.log_callback("[FFmpeg] Cleaning up temporary files")
                for f in temp_files:
                    if f.exists() and f != final_file:
                        try:
                            f.unlink()
                            self.log_callback(f"[FFmpeg] Deleted temp file: {f}")
                        except Exception as e:
                            self.log_callback(f"[FFmpeg][WARN] Failed to delete {f}: {e}")

                if callback:
                    callback(final_file)
            except Exception as e:
                self.log_callback(f"[FFmpeg][ERROR] Export failed: {e}")
                if callback:
                    callback(None, e)

        self.executor.submit(worker)
