package display

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
)

type Terminal int

const (
	Unknown Terminal = iota
	ITerm2
	Kitty
	WezTerm
)

func DetectTerminal() Terminal {
	termProgram := os.Getenv("TERM_PROGRAM")
	switch termProgram {
	case "iTerm.app":
		return ITerm2
	case "WezTerm":
		return WezTerm
	}

	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return Kitty
	}

	term := os.Getenv("TERM")
	if term == "xterm-kitty" {
		return Kitty
	}

	return Unknown
}

func ShowImage(imageData []byte, filename string) {
	term := DetectTerminal()

	if term == ITerm2 || term == WezTerm {
		showITerm2(imageData)
	} else if term == Kitty {
		showKitty(imageData)
	} else {
		showITerm2(imageData)
	}

	if term == Unknown {
		tryChafa(imageData, filename)
	}
}

func showITerm2(imageData []byte) {
	b64 := base64.StdEncoding.EncodeToString(imageData)
	name := base64.StdEncoding.EncodeToString([]byte("openimage"))
	fmt.Fprintf(os.Stdout, "\033]1337;File=name=%s;inline=1;width=auto:%s\a\n", name, b64)
}

func showKitty(imageData []byte) {
	b64 := base64.StdEncoding.EncodeToString(imageData)
	chunkSize := 4096
	for i := 0; i < len(b64); i += chunkSize {
		end := i + chunkSize
		if end > len(b64) {
			end = len(b64)
		}
		chunk := b64[i:end]
		more := byte('1')
		if end == len(b64) {
			more = '0'
		}
		fmt.Fprintf(os.Stdout, "\033_Gf=100,m=%c;%s\033\\", more, chunk)
	}
}

func tryChafa(imageData []byte, filename string) {
	if filename == "" {
		return
	}
	if _, err := exec.LookPath("chafa"); err == nil {
		cmd := exec.Command("chafa", "--size=80x24", filename)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}
}
