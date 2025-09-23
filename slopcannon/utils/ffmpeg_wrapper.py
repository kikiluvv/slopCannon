import subprocess
from pathlib import Path
from slopcannon.utils.subtitle_generator import SubtitleGenerator

class FFmpegWrapper:
    def __init__(self, ffmpeg_path=None):
        self.ffmpeg = str(ffmpeg_path) if ffmpeg_path else "ffmpeg"
        self.overlay_video = Path("assets/overlays/mc.mp4")
        # keep a generator instance (it will lazy-load model as needed)
        self.subs = SubtitleGenerator()

    def export_clip(
        self,
        input_file,
        start_ms,
        end_ms,
        output_file,
        portrait=False,
        overlay=False,
        subtitles=False,
        subtitle_settings=None,  # expect an object with attributes (see SubtitleSettings)
    ):
        start_sec = start_ms / 1000
        duration_sec = (end_ms - start_ms) / 1000

        print(f"[FFmpegWrapper] Exporting {input_file} → {output_file}")
        print(f"[FFmpegWrapper] Clip: {start_sec:.2f}s–{start_sec+duration_sec:.2f}s "
              f"(duration {duration_sec:.2f}s)")

        # --------------------
        # STEP 1: export base video
        # --------------------
        cmd = [self.ffmpeg, "-y", "-i", str(input_file)]

        if overlay:
            cmd += ["-i", str(self.overlay_video)]
            filter_complex = (
                "[0:v]scale=1080:960,setsar=1[v0];"
                "[1:v]scale=1080:960,setsar=1[v1];"
                "[v0][v1]vstack=inputs=2[outv]"
            )
            cmd += [
                "-filter_complex", filter_complex,
                "-map", "[outv]",
                "-map", "0:a?",
                "-c:v", "libx264",
                "-preset", "ultrafast",
                "-crf", "23",
                "-ss", str(start_sec),
                "-t", str(duration_sec),
                str(output_file),
            ]
        else:
            if portrait:
                cmd += [
                    "-vf",
                    "scale=1080:1920:force_original_aspect_ratio=decrease,"
                    "pad=1080:1920:(ow-iw)/2:(oh-ih)/2",
                    "-c:v", "libx264",
                    "-preset", "ultrafast",
                    "-crf", "23",
                ]
            else:
                cmd += ["-c", "copy"]

            cmd += [
                "-ss", str(start_sec),
                "-t", str(duration_sec),
                str(output_file),
            ]

        print("[FFmpegWrapper] Running ffmpeg:", " ".join(cmd))
        subprocess.run(cmd, check=True)

        # --------------------
        # STEP 2: generate + burn subtitles
        # --------------------
        if subtitles:
            # generate ass for the segment
            ass_file = Path(output_file).with_suffix(".ass")
            audio_file = Path(output_file).with_suffix(".wav")

            # extract audio for transcription
            audio_cmd = [
                self.ffmpeg, "-y", "-i", str(output_file),
                "-vn", "-ac", "1", "-ar", "16000", str(audio_file),
            ]
            subprocess.run(audio_cmd, check=True)

            # call the subtitle generator with UI settings (if provided)
            if subtitle_settings:
                # pass individual settings explicitly
                self.subs.generate_subtitles(
                    audio_file,
                    ass_file,
                    words_per_line=getattr(subtitle_settings, "words_per_line", 5),
                    lines_per_event=getattr(subtitle_settings, "lines_per_event", 2),
                    fade_ms=getattr(subtitle_settings, "fade_ms", 100),
                    font=getattr(subtitle_settings, "font", "Comic Sans MS"),
                    font_size=getattr(subtitle_settings, "font_size", 72),
                    primary_color=getattr(subtitle_settings, "primary_color", "&H00FFFFFF"),
                    secondary_color=getattr(subtitle_settings, "secondary_color", "&H00FFFFFF"),
                    outline_color=getattr(subtitle_settings, "outline_color", "&H00000000"),
                    back_color=getattr(subtitle_settings, "back_color", "&H64000000"),
                    model_size=getattr(subtitle_settings, "model_size", "small")
                )
            else:
                # fallback to generate_subtitles defaults
                self.subs.generate_subtitles(audio_file, ass_file)

            # burn the ASS file (ffmpeg reads styles from the ASS file itself)
            sub_output = Path(output_file).with_name(f"{Path(output_file).stem}_sub.mp4")
            sub_cmd = [
                self.ffmpeg, "-y",
                "-i", str(output_file),
                "-vf", f"subtitles='{str(ass_file)}'",
                "-c:v", "libx264",
                "-preset", "ultrafast",
                "-crf", "23",
                "-c:a", "copy",
                str(sub_output),
            ]
            print(f"[FFmpegWrapper] Burning subtitles → {sub_output}")
            subprocess.run(sub_cmd, check=True)
            return sub_output

        print(f"[FFmpegWrapper] Export finished → {output_file}")
        return output_file
