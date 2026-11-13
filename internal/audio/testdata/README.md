# Synthetic audio fixture

`tone.ogg` is an authored 80 ms, 440 Hz mono test tone at 22050 Hz, generated locally with FFmpeg/libvorbis. It contains no game audio. It tests decoding, mono duplication and resampling to the mixer's 44100 Hz stereo PCM format.

To regenerate:

```sh
ffmpeg -f lavfi -i 'sine=frequency=440:sample_rate=22050:duration=0.08' \
  -ac 1 -c:a libvorbis -q:a 6 -map_metadata -1 -y tone.ogg
```
