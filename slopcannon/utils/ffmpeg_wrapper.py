import subprocess
from pathlib import Path
from .subtitle_generator import SubtitleGenerator
from concurrent.futures import ThreadPoolExecutor

class FFmpegWrapper:
    def __init__(self, ffmpeg_path=None, log_callback=print):
        self.ffmpeg = str(ffmpeg_path) if ffmpeg_path else "ffmpeg"
        self.overlay_video = Path("assets/overlays/mc.mp4")
        self.log_callback = log_callback
        self.subs = SubtitleGenerator(log_callback=log_callback)
        self.executor = ThreadPoolExecutor(max_workers=1)
        self.log_callback("[FFmpegWrapper] Initialized")

    def run_cmd(self, cmd):
        self.log_callback(f"[FFmpeg] Running command: {' '.join(cmd)}")
        process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        stdout, stderr = process.communicate()

        if process.returncode != 0:
            self.log_callback(f"[FFmpeg][ERROR] Command failed with code {process.returncode}")
            if stdout:
                self.log_callback(f"[FFmpeg][stdout]\n{stdout.strip()}")
            if stderr:
                self.log_callback(f"[FFmpeg][stderr]\n{stderr.strip()}")
            raise subprocess.CalledProcessError(process.returncode, cmd)

        self.log_callback(f"[FFmpeg] Command finished successfully with code {process.returncode}")
        return process.returncode

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
            try:
                start_sec = start_ms / 1000
                duration_sec = (end_ms - start_ms) / 1000
                self.log_callback(f"[FFmpeg] Exporting clip {input_file} → {output_file}")

                # base ffmpeg command
                cmd = [self.ffmpeg, "-y", "-i", str(input_file)]
                if overlay:
                    self.log_callback(f"[FFmpeg] Adding overlay: {self.overlay_video}")
                    cmd += ["-i", str(self.overlay_video)]
                    filter_complex = "[0:v]scale=1080:960,setsar=1[v0];[1:v]scale=1080:960,setsar=1[v1];[v0][v1]vstack=inputs=2[outv]"
                    cmd += ["-filter_complex", filter_complex, "-map", "[outv]", "-map", "0:a?",
                            "-c:v", "libx264", "-preset", "ultrafast", "-crf", "23",
                            "-ss", str(start_sec), "-t", str(duration_sec), str(output_file)]
                else:
                    if portrait:
                        cmd += ["-vf", "scale=1080:1920:force_original_aspect_ratio=decrease,pad=1080:1920:(ow-iw)/2:(oh-ih)/2",
                                "-c:v", "libx264", "-preset", "ultrafast", "-crf", "23"]
                    else:
                        cmd += ["-c", "copy"]
                    cmd += ["-ss", str(start_sec), "-t", str(duration_sec), str(output_file)]

                self.run_cmd(cmd)
                final_file = Path(output_file)

                if subtitles:
                    ass_file = final_file.with_suffix(".ass")
                    audio_file = final_file.with_suffix(".wav")

                    self.run_cmd([self.ffmpeg, "-y", "-i", str(final_file), "-vn", "-ac", "1", "-ar", "16000", str(audio_file)])

                    # separate model vs rendering settings
                    render_kwargs = {}
                    if subtitle_settings:
                        render_kwargs = {
                            k: v for k, v in vars(subtitle_settings).items()
                            if k in ["words_per_line","lines_per_event","fade_ms","font","font_size",
                                    "primary_color","secondary_color","outline_color","back_color"]
                        }
                        # update model if needed
                        self.subs.update_model(model_size=subtitle_settings.model_size)

                    self.subs.generate_subtitles(audio_file, ass_file, **render_kwargs)

                    # burn subtitles
                    sub_output = final_file.with_name(f"{final_file.stem}_sub.mp4")
                    burn_cmd = [self.ffmpeg, "-y", "-i", str(final_file), "-vf", f"subtitles='{ass_file}'",
                                "-c:v", "libx264", "-preset", "ultrafast", "-crf", "23", "-c:a", "copy", str(sub_output)]
                    self.run_cmd(burn_cmd)
                    final_file = sub_output

                if callback:
                    callback(final_file)
            except Exception as e:
                self.log_callback(f"[FFmpeg][ERROR] Export failed: {e}")
                if callback:
                    callback(None, e)

        self.executor.submit(worker)

