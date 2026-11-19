package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"sort"
)

// makeGIF assembles the PNG of every frame of a set (rendered from the SVGs by a browser) into a GIF.
// The palette is the 256 most used colours of all frames (a terminal has few), so text stays sharp
// without dithering.
func makeGIF(in, set, out string) error {
	b, err := os.ReadFile(filepath.Join(in, "frames.json"))
	if err != nil {
		return err
	}
	var info []frameInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return err
	}
	var imgs []image.Image
	var holds []int
	for _, f := range info {
		if f.Set != set {
			continue
		}
		fh, err := os.Open(filepath.Join(in, f.File+".png"))
		if err != nil {
			return err
		}
		img, err := png.Decode(fh)
		fh.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", f.File, err)
		}
		imgs = append(imgs, img)
		holds = append(holds, max(1, f.Hold))
	}
	if len(imgs) == 0 {
		return fmt.Errorf("no frames in set %q", set)
	}
	pal := palette(imgs)
	g := &gif.GIF{LoopCount: 0}
	for i, img := range imgs {
		p := image.NewPaletted(img.Bounds(), pal)
		draw.Draw(p, p.Rect, img, img.Bounds().Min, draw.Src) // nearest colour, no dithering
		g.Image = append(g.Image, p)
		g.Delay = append(g.Delay, holds[i]*90) // hundredths of a second per hold unit
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := gif.EncodeAll(f, g); err != nil {
		return err
	}
	fmt.Printf("%s: %d frames\n", out, len(imgs))
	return nil
}

func palette(imgs []image.Image) color.Palette {
	count := map[color.RGBA]int{}
	for _, img := range imgs {
		r := img.Bounds()
		for y := r.Min.Y; y < r.Max.Y; y += 2 {
			for x := r.Min.X; x < r.Max.X; x += 2 {
				c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
				c.A = 255
				count[c]++
			}
		}
	}
	type kv struct {
		c color.RGBA
		n int
	}
	var all []kv
	for c, n := range count {
		all = append(all, kv{c, n})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n })
	var pal color.Palette
	for _, x := range all {
		if len(pal) == 256 {
			break
		}
		pal = append(pal, x.c)
	}
	return pal
}
