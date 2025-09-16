import subprocess
from pathlib import Path

class FFmpegWrapper:
    def __init__(self, ffmpeg_path=None):
        """
        ffmpeg_path: optional Path to bundled ffmpeg binary.
                     if None, assumes 'ffmpeg' is in PATH
        """
        self.ffmpeg = str(ffmpeg_path) if ffmpeg_path else "ffmpeg"
        self.overlay_video = Path("assets/overlays/mc.mp4")  # default overlay

    def export_clip(self, input_file, start_ms, end_ms, output_file, portrait=False, overlay=False):
        """
        Export a clip from input_file [start_ms, end_ms] to output_file.
        - portrait=True scales/crops to 1080x1920 (portrait)
        - overlay=True stacks input video with overlay video vertically
        """
        start_sec = start_ms / 1000
        duration_sec = (end_ms - start_ms) / 1000

        cmd = [self.ffmpeg, "-y", "-i", str(input_file)]

        if overlay:
            # add overlay input
            cmd += ["-i", str(self.overlay_video)]
            # scale both videos to half height (1080x960) and stack vertically
            filter_complex = (
                "[0:v]scale=1080:960,setsar=1[v0];"
                "[1:v]scale=1080:960,setsar=1[v1];"
                "[v0][v1]vstack=inputs=2[outv]"
            )
            cmd += [
                "-filter_complex", filter_complex,
                "-map", "[outv]",
                "-map", "0:a?",  # optional audio from main video
                "-ss", str(start_sec),
                "-t", str(duration_sec),
                str(output_file)
            ]
        else:
            if portrait:
                cmd += [
                    "-vf",
                    "scale=1080:1920:force_original_aspect_ratio=decrease,"
                    "pad=1080:1920:(ow-iw)/2:(oh-ih)/2"
                ]
            else:
                cmd += ["-c", "copy"]

            cmd += [
                "-ss", str(start_sec),
                "-t", str(duration_sec),
                str(output_file)
            ]

        print("Running ffmpeg:", " ".join(cmd))
        subprocess.run(cmd, check=True)
