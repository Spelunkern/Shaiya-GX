package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFieldLoaderDecodesWithoutGLAndCancelsBackpressure(t *testing.T) {
	dir := t.TempDir()
	file, err := os.Create(filepath.Join(dir, "test.png"))
	if err != nil {
		t.Fatal(err)
	}
	im := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	im.SetNRGBA(0, 0, color.NRGBA{10, 20, 30, 255})
	if err = png.Encode(file, im); err != nil {
		t.Fatal(err)
	}
	file.Close()
	arc, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	l := newFieldLoader(arc)
	l.jobs <- fieldAssetJob{fieldAssetKey{fieldTexture, "test.png"}, l.generation}
	select {
	case result := <-l.results:
		if result.err != nil || result.texture.RGBA == nil || result.texture.RGBA.Pix[0] != 10 {
			l.close()
			t.Fatal("worker decode", result.err)
		}
	case <-time.After(time.Second):
		l.close()
		t.Fatal("decode stalled")
	}
	// Old jobs may complete after a camera generation change; the result carries
	// its original generation, never mutating the new render-thread state map.
	old := l.generation
	l.jobs <- fieldAssetJob{fieldAssetKey{fieldTexture, "test.png"}, old}
	l.reset()
	select {
	case result := <-l.results:
		if result.job.generation != old || l.generation == old {
			t.Fatal("lost generation identity")
		}
	case <-time.After(time.Second):
		l.close()
		t.Fatal("generation result stalled")
	}
	// Fill the result channel and block both workers sending more results.
	for range 8 {
		l.jobs <- fieldAssetJob{fieldAssetKey{fieldTexture, "test.png"}, l.generation}
	}
	done := make(chan struct{})
	go func() { l.close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("workers leaked under result backpressure")
	}
}

func TestPinnedMissingTextureRemainsReady(t *testing.T) {
	arc, err := archive.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	assets := NewAssets(arc)
	assets.textures["world:missing.dds"] = nil
	l := newFieldLoader(arc)
	defer l.close()
	key := fieldAssetKey{fieldTexture, "missing.dds"}
	l.pinned[key] = true
	l.states[key] = &fieldAssetState{}
	s := &fieldStream{world: &World{}, assets: assets, loader: l}
	s.trim()
	if !s.ensureTexture("missing.dds") {
		t.Fatal("evicted negative cache stalled a dependency")
	}
}
