package display

import (
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"strings"

	_ "golang.org/x/image/webp"
)

var termW int
var termH int
var termType string

func init() {
	termW, termH = 80, 24
	if ws, err := getTermSize(); err == nil {
		termW = ws[0]
		termH = ws[1]
	}
	termType = detectTermProtocol()
}

func detectTermProtocol() string {
	if os.Getenv("KITTY_WINDOW_ID") != "" || strings.Contains(os.Getenv("TERM"), "kitty") {
		return "kitty"
	}
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" || os.Getenv("TERM_PROGRAM") == "WezTerm" {
		return "iterm2"
	}
	if strings.Contains(os.Getenv("TERM"), "xterm") || strings.Contains(os.Getenv("TERM"), "screen") {
		return "sixel"
	}
	return "braille"
}

func ShowImage(data []byte) {
	switch termType {
	case "kitty":
		renderKitty(data)
	case "iterm2":
		renderITerm2(data)
	case "sixel":
		renderSixel(data)
	default:
		renderBraille(data)
	}
}

func ShowInline(data []byte) string {
	switch termType {
	case "kitty":
		return renderKittyData(data)
	case "iterm2":
		return renderITerm2Data(data)
	case "sixel":
		return renderSixelData(data)
	default:
		return renderBrailleData(data)
	}
}

func renderKitty(data []byte) {
	fmt.Fprint(os.Stdout, renderKittyData(data))
}

func renderKittyData(data []byte) string {
	b64 := base64.StdEncoding.EncodeToString(data)
	var sb strings.Builder
	chunkSize := 4096
	for i := 0; i < len(b64); i += chunkSize {
		end := i + chunkSize
		if end > len(b64) {
			end = len(b64)
		}
		more := byte('1')
		if end == len(b64) {
			more = '0'
		}
		sb.WriteString(fmt.Sprintf("\033_Gf=100,m=%c;%s\033\\", more, b64[i:end]))
	}
	return sb.String()
}

func renderITerm2(data []byte) {
	fmt.Fprint(os.Stdout, renderITerm2Data(data))
}

func renderITerm2Data(data []byte) string {
	b64 := base64.StdEncoding.EncodeToString(data)
	name := base64.StdEncoding.EncodeToString([]byte("img"))
	return fmt.Sprintf("\033]1337;File=name=%s;inline=1;width=auto:%s\a", name, b64)
}

func renderSixel(data []byte) {
	fmt.Fprint(os.Stdout, renderSixelData(data))
}

func renderSixelData(data []byte) string {
	img, _, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return ""
	}

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	scale := math.Min(float64(termW*10)/float64(w), float64(termH*12)/float64(h))
	if scale < 1 {
		w = int(float64(w) * scale)
		h = int(float64(h) * scale)
		img = resizeNN(img, w, h)
	}

	levels := []uint8{0, 36, 73, 109, 146, 182, 219, 255}

	var sb strings.Builder
	sb.WriteString("\033Pq")

	for _, gray := range levels {
		r, g, b := gray, gray, gray
		sb.WriteString(fmt.Sprintf("#%d;2;%d;%d;%d", gray, r, g, b))
	}

	for y := 0; y < h; y += 6 {
		for li, gray := range levels {
			sb.WriteString(fmt.Sprintf("#%d", gray))
			for x := 0; x < w; x++ {
				var band byte
				for dy := 0; dy < 6 && y+dy < h; dy++ {
					cr, cg, cb, _ := img.At(x, y+dy).RGBA()
					lum := uint8((uint16(cr>>8)*77 + uint16(cg>>8)*150 + uint16(cb>>8)*29) >> 8)
					bestLevel := 0
					bestDist := 255
					for i, g := range levels {
						d := int(lum) - int(g)
						if d < 0 {
							d = -d
						}
						if d < bestDist {
							bestDist = d
							bestLevel = i
						}
					}
					if bestLevel == li {
						band |= 1 << dy
					}
				}
				if band > 0 {
					sb.WriteByte(band + 63)
				}
			}
		}
		sb.WriteString("$")
		sb.WriteString("-")
	}
	sb.WriteString("\033\\")
	return sb.String()
}

func renderBraille(data []byte) {
	fmt.Fprint(os.Stdout, renderBrailleData(data))
}

var brailleBase = rune(0x2800)

var brailleOffsets = [8][2]int{
	{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}, {0, 3}, {1, 3},
}

func renderBrailleData(data []byte) string {
	img, _, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return ""
	}

	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()

	maxW := termW * 2
	maxH := termH * 4
	if imgW > maxW || imgH > maxH {
		scale := math.Min(float64(maxW)/float64(imgW), float64(maxH)/float64(imgH))
		imgW = int(float64(imgW) * scale)
		imgH = int(float64(imgH) * scale)
		img = resizeNN(img, imgW, imgH)
	}

	cols := imgW / 2
	rows := imgH / 4

	var sb strings.Builder
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			var dots [8]float64
			for i, off := range brailleOffsets {
				px := x*2 + off[0]
				py := y*4 + off[1]
				if px < imgW && py < imgH {
					r, g, b, _ := img.At(px, py).RGBA()
					dots[i] = 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
				}
			}
			avg := 0.0
			for _, d := range dots {
				avg += d
			}
			avg /= 8
			threshold := avg * 0.8

			var code rune
			for i, d := range dots {
				if d > threshold {
					code |= 1 << i
				}
			}
			sb.WriteRune(brailleBase + code)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func getTermSize() ([2]int, error) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return [2]int{80, 24}, err
	}
	var rows, cols int
	fmt.Sscanf(string(out), "%d %d", &rows, &cols)
	if cols < 1 {
		cols = 80
	}
	return [2]int{cols, rows}, nil
}

func resizeNN(img image.Image, newW, newH int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xRatio := float64(img.Bounds().Dx()) / float64(newW)
	yRatio := float64(img.Bounds().Dy()) / float64(newH)
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := int(float64(x) * xRatio)
			srcY := int(float64(y) * yRatio)
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}
	return dst
}
