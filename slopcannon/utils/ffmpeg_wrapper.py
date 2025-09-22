import subprocess
from pathlib import Path
from slopcannon.utils.subtitle_generator import SubtitleGenerator

class FFmpegWrapper:
    def __init__(self, ffmpeg_path=None):
        self.ffmpeg = str(ffmpeg_path) if ffmpeg_path else "ffmpeg"
        self.overlay_video = Path("assets/overlays/mc.mp4")
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
            # generate srt for the segment
            srt_file = Path(output_file).with_suffix(".srt")
            audio_file = Path(output_file).with_suffix(".wav")

            # extract audio for transcription
            audio_cmd = [
                self.ffmpeg, "-y", "-i", str(output_file),
                "-vn", "-ac", "1", "-ar", "16000", str(audio_file),
            ]
            subprocess.run(audio_cmd, check=True)

            self.subs.generate_subtitles(audio_file, srt_file)

            sub_output = Path(output_file).with_name(f"{Path(output_file).stem}_sub.mp4")
            sub_cmd = [
                self.ffmpeg, "-y",
                "-i", str(output_file),
                "-vf", f"subtitles='{str(srt_file)}':force_style='Fontsize=28,PrimaryColour=&H00FFFF&,OutlineColour=&H000000&,BorderStyle=1,Outline=2'",
                "-c:a", "copy",
                str(sub_output),
            ]
            print(f"[FFmpegWrapper] Burning subtitles → {sub_output}")
            subprocess.run(sub_cmd, check=True)
            return sub_output

        print(f"[FFmpegWrapper] Export finished → {output_file}")
        return output_file
