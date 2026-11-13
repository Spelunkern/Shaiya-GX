// Command datamigrate imports a Parsec-compatible SAH/SAF into loose GX assets.
// It keeps unchanged resources byte-for-byte and converts UI textures to PNG
// and sounds to Ogg Vorbis. Outputs are prepared separately from a live client.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unsafe"

	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func init() { platform.LockMainThread() }

type record struct {
	Source           string `json:"source"`
	Output           string `json:"output"`
	OldBytes         uint32 `json:"old_bytes"`
	NewBytes         int    `json:"new_bytes"`
	SHA256           string `json:"sha256"`
	Kind             string `json:"kind"`
	Mips             int    `json:"mips,omitempty"`
	ReferenceChanges int    `json:"reference_changes,omitempty"`
}
type job struct {
	file  string
	entry *archive.File
}

var waveRef = regexp.MustCompile(`(?i)\.wav`)

func main() {
	source := flag.String("source", "", "directory containing data.sah/data.saf")
	output := flag.String("output", "", "new, empty loose data directory")
	report := flag.String("report", "", "conversion manifest outside the data directory")
	workers := flag.Int("workers", 4, "bounded audio encoder workers")
	quality := flag.String("quality", "6", "Vorbis VBR quality")
	resume := flag.Bool("resume", false, "reuse verified results from a report for the identical source")
	flag.Parse()
	if *source == "" || *output == "" || *report == "" || *workers < 1 || *workers > 8 {
		log.Fatal("require -source, -output, -report and workers 1..8")
	}
	if _, err := os.Stat(*output); !os.IsNotExist(err) && !*resume {
		log.Fatal("output must not exist")
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		log.Fatal(err)
	}
	a, err := archive.Open(*source)
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()
	waveNames := map[string]bool{}
	a.Walk(func(file string, _ *archive.File) {
		if strings.EqualFold(path.Ext(file), ".wav") {
			waveNames[strings.ToLower(path.Base(file))] = true
		}
	})
	sourceSHA := map[string]string{}
	for _, name := range []string{"data.sah", "data.saf"} {
		f, e := os.Open(filepath.Join(*source, name))
		if e != nil {
			log.Fatal(e)
		}
		h := sha256.New()
		_, e = io.Copy(h, f)
		f.Close()
		if e != nil {
			log.Fatal(e)
		}
		sourceSHA[name] = hex.EncodeToString(h.Sum(nil))
	}
	cached := map[string]record{}
	if *resume {
		var previous struct {
			Records   []record          `json:"records"`
			SourceSHA map[string]string `json:"source_sha256"`
			Quality   string            `json:"vorbis_quality"`
		}
		b, e := os.ReadFile(*report)
		if e != nil {
			log.Fatal(e)
		}
		if e = json.Unmarshal(b, &previous); e != nil {
			log.Fatal(e)
		}
		if previous.Quality != *quality || previous.SourceSHA["data.sah"] != sourceSHA["data.sah"] || previous.SourceSHA["data.saf"] != sourceSHA["data.saf"] {
			log.Fatal("resume source/quality does not match report")
		}
		for _, r := range previous.Records {
			cached[r.Output] = r
		}
	}
	var records []record
	var mu sync.Mutex
	var errors []string
	var shadowed int
	add := func(r record, err error) {
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", r.Source, err))
			log.Println("FAILED", r.Source, err)
		} else {
			records = append(records, r)
			if len(records)%250 == 0 {
				log.Printf("processed %d", len(records))
			}
		}
	}
	encodePNG := func(p *img.RGBA) ([]byte, error) {
		var b bytes.Buffer
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		err := enc.Encode(&b, &image.NRGBA{Pix: p.Pix, Stride: p.W * 4, Rect: image.Rect(0, 0, p.W, p.H)})
		return b.Bytes(), err
	}
	// Only legacy DXT images need the GPU. Readback uses the same decompressor
	// as the existing client, avoiding color/alpha rounding or orientation drift.
	win, err := platform.Open(platform.Config{Title: "GX data conversion", Width: 16, Height: 16, Hidden: true})
	if err != nil {
		log.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
		log.Fatal(err)
	}
	jobs := make(chan job)
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				r := record{Source: j.file, Output: portablePath(strings.TrimSuffix(j.file, path.Ext(j.file)) + ".ogg"), OldBytes: j.entry.Length, Kind: "ogg"}
				raw, e := a.Read(j.entry)
				if e != nil {
					add(r, e)
					continue
				}
				dest := filepath.Join(*output, filepath.FromSlash(r.Output))
				if e = os.MkdirAll(filepath.Dir(dest), 0755); e != nil {
					add(r, e)
					continue
				}
				channels := 2
				if at := bytes.Index(raw, []byte("fmt ")); at >= 0 && len(raw) >= at+12 && binary.LittleEndian.Uint16(raw[at+10:]) == 1 {
					channels = 1
				}
				cmd := exec.Command("ffmpeg", "-nostdin", "-hide_banner", "-loglevel", "error", "-i", "pipe:0", "-map", "0:a:0", "-map_metadata", "-1", "-ar", "44100", "-ac", fmt.Sprint(channels), "-c:a", "libvorbis", "-q:a", *quality, "-threads", "1", "-f", "ogg", "-y", dest)
				cmd.Stdin = bytes.NewReader(raw)
				message, e := cmd.CombinedOutput()
				if e != nil {
					add(r, fmt.Errorf("ffmpeg: %w: %s", e, message))
					continue
				}
				result, e := os.ReadFile(dest)
				if e != nil {
					add(r, e)
					continue
				}
				pcm, e := audio.DecodeOGG(result)
				if e != nil {
					add(r, e)
					continue
				}
				if old, e := audio.DecodeWAV(raw); e == nil {
					delta := len(old) - len(pcm)
					if delta < 0 {
						delta = -delta
					}
					if delta > SampleTolerance {
						add(r, fmt.Errorf("duration changed: old %d PCM bytes, new %d", len(old), len(pcm)))
						continue
					}
				}
				sum := sha256.Sum256(result)
				r.NewBytes = len(result)
				r.SHA256 = hex.EncodeToString(sum[:])
				add(r, nil)
			}
		}()
	}
	a.Walk(func(file string, f *archive.File) {
		effective, ok := a.Lookup(file)
		if !ok || effective != f {
			shadowed++
			return
		}
		ext := strings.ToLower(path.Ext(file))
		target := portablePath(file)
		if ext == ".wav" {
			target = portablePath(strings.TrimSuffix(file, path.Ext(file)) + ".ogg")
		} else if strings.HasPrefix(strings.ToLower(file), "interface/") {
			stem := strings.TrimSuffix(file, path.Ext(file))
			if ext == ".dds" {
				stem += "_dds"
			}
			target = portablePath(stem + ".png")
		}
		if old, ok := cached[target]; ok && old.OldBytes == f.Length {
			if b, e := os.ReadFile(filepath.Join(*output, filepath.FromSlash(target))); e == nil {
				sum := sha256.Sum256(b)
				if hex.EncodeToString(sum[:]) == old.SHA256 {
					add(old, nil)
					return
				}
			}
		}
		if ext == ".wav" {
			jobs <- job{file, f}
			return
		}
		raw, e := a.Read(f)
		if e != nil {
			add(record{Source: file}, e)
			return
		}
		r := record{Source: file, Output: portablePath(file), OldBytes: f.Length, Kind: "copy"}
		result := raw
		if strings.HasPrefix(strings.ToLower(file), "interface/") {
			r.Kind = "png"
			name := strings.TrimSuffix(file, path.Ext(file))
			if ext == ".dds" {
				name += "_dds"
			}
			r.Output = portablePath(name + ".png")
			t, e := img.Decode(raw)
			if e != nil {
				add(r, e)
				return
			}
			var levels []*img.RGBA
			if t.RGBA != nil {
				levels = []*img.RGBA{t.RGBA}
			} else {
				d := t.DDS
				w, h := d.W, d.H
				var tex *render.Texture
				if d.Format != img.FormatRGBA {
					tex = render.Upload(t)
					defer tex.Delete()
				}
				for i, l := range d.Levels {
					pixels := l
					if tex != nil {
						pixels = make([]byte, w*h*4)
						gl.GetTextureImage(tex.ID, int32(i), gl.RGBA, gl.UNSIGNED_BYTE, int32(len(pixels)), unsafe.Pointer(&pixels[0]))
					}
					levels = append(levels, &img.RGBA{W: w, H: h, Pix: pixels})
					w, h = max(1, w/2), max(1, h/2)
				}
			}
			result, e = encodePNG(levels[0])
			if e != nil {
				add(r, e)
				return
			}
			dest := filepath.Join(*output, filepath.FromSlash(r.Output))
			if e = os.MkdirAll(filepath.Dir(dest), 0755); e != nil {
				add(r, e)
				return
			}
			if e = os.WriteFile(dest, result, 0644); e != nil {
				add(r, e)
				return
			}
			// Lossless palette/bit-depth and deflate optimization. Verify every pixel.
			cmd := exec.Command("optipng", "-quiet", "-o2", "-strip", "all", dest)
			if msg, e := cmd.CombinedOutput(); e != nil {
				add(r, fmt.Errorf("optipng: %w: %s", e, msg))
				return
			}
			result, e = os.ReadFile(dest)
			if e != nil {
				add(r, e)
				return
			}
			decoded, e := img.DecodePNG(result)
			if e != nil || decoded.RGBA == nil || !bytes.Equal(decoded.RGBA.Pix, levels[0].Pix) {
				add(r, fmt.Errorf("lossless PNG pixel check failed: %v", e))
				return
			}
			r.Mips = len(levels)
			if len(levels) > 1 {
				var payload bytes.Buffer
				binary.Write(&payload, binary.LittleEndian, uint32(len(levels)-1))
				for _, mip := range levels[1:] {
					encoded, e := encodePNG(mip)
					if e != nil {
						add(r, e)
						return
					}
					binary.Write(&payload, binary.LittleEndian, uint32(len(encoded)))
					payload.Write(encoded)
				}
				result = addChunk(result, "gxMP", payload.Bytes())
				decoded, e = img.DecodePNG(result)
				if e != nil || decoded.DDS == nil || len(decoded.DDS.Levels) != len(levels) {
					add(r, fmt.Errorf("PNG mipmap verification: %v", e))
					return
				}
				for i, mip := range levels {
					if !bytes.Equal(mip.Pix, decoded.DDS.Levels[i]) {
						add(r, fmt.Errorf("PNG mipmap %d changed", i))
						return
					}
				}
			}
		} else if structured(ext) {
			result = append([]byte(nil), raw...)
			for _, at := range waveRef.FindAllIndex(raw, -1) {
				// Binary float/vertex data can coincidentally contain .wav.
				// Only retarget a complete, terminated known sound name.
				if !knownWaveReference(raw, at[0], at[1], waveNames) {
					continue
				}
				if at[0] == 0 || !filenameByte(raw[at[0]-1]) {
					continue
				}
				if at[1] < len(raw) && filenameByte(raw[at[1]]) {
					continue
				}
				copy(result[at[0]:at[1]], []byte(".ogg"))
				r.ReferenceChanges++
			}
			if r.ReferenceChanges > 0 {
				r.Kind = "references"
			}
		}
		dest := filepath.Join(*output, filepath.FromSlash(r.Output))
		if e = os.MkdirAll(filepath.Dir(dest), 0755); e == nil {
			e = os.WriteFile(dest, result, 0644)
		}
		sum := sha256.Sum256(result)
		r.NewBytes = len(result)
		r.SHA256 = hex.EncodeToString(sum[:])
		add(r, e)
	})
	close(jobs)
	wg.Wait()
	summary := map[string]any{"records": records, "errors": errors, "shadowed_original_entries": shadowed, "source_sha256": sourceSHA, "vorbis_quality": *quality, "png": "lossless; original authored mipmaps in gxMP ancillary chunk"}
	data, _ := json.MarshalIndent(summary, "", "  ")
	if err = os.WriteFile(*report, data, 0600); err != nil {
		log.Fatal(err)
	}
	if len(errors) > 0 {
		log.Fatalf("conversion failed for %d assets; see %s", len(errors), *report)
	}
	verified, err := archive.OpenDirectory(*output)
	if err != nil {
		log.Fatal(err)
	}
	defer verified.Close()
	if int(verified.FileCount) != len(records) {
		log.Fatal("file count mismatch")
	}
	log.Printf("conversion complete: %d assets; %d shadowed originals excluded", len(records), shadowed)
}

const SampleTolerance = 441 * 4 // 10 ms allows codec/resampler rounding, never trimming.

func knownWaveReference(raw []byte, start, end int, names map[string]bool) bool {
	if end < len(raw) && raw[end] != 0 && raw[end] != '\r' && raw[end] != '\n' && raw[end] != '"' {
		return false
	}
	for i := start - 1; i >= 0 && start-i <= 255 && filenameByte(raw[i]); i-- {
		if names[strings.ToLower(string(raw[i:end]))] {
			return true
		}
	}
	return false
}

func filenameByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-'
}
func structured(ext string) bool {
	switch ext {
	case ".wld", ".mon", ".alt", ".mlt", ".eft", ".seff", ".ctl", ".itm", ".tol", ".zon", ".vani", ".mani", ".cfg", ".ini", ".txt":
		return true
	}
	return false
}
func portablePath(p string) string {
	parts := strings.Split(strings.ReplaceAll(p, `\`, "/"), "/")
	for i, name := range parts {
		if name == "" || name == "." || name == ".." {
			log.Fatalf("unsafe archive path %q", p)
		}
		parts[i] = archive.PortableName(name)
	}
	return strings.Join(parts, "/")
}
func addChunk(pngData []byte, kind string, payload []byte) []byte {
	var b bytes.Buffer
	b.Write(pngData[:len(pngData)-12])
	binary.Write(&b, binary.BigEndian, uint32(len(payload)))
	b.WriteString(kind)
	b.Write(payload)
	crc := crc32.NewIEEE()
	crc.Write([]byte(kind))
	crc.Write(payload)
	binary.Write(&b, binary.BigEndian, crc.Sum32())
	b.Write(pngData[len(pngData)-12:])
	return b.Bytes()
}
