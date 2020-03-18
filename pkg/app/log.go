package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/environment"
)

const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite

	colorBold     = 1
	colorDarkGray = 90
)

// colorize returns the string s wrapped in ANSI code c, unless disabled is true.
func colorize(s interface{}, c int) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

func formatLevel(i interface{}) string {
	var l string
	if ll, ok := i.(string); ok {
		switch ll {
		case "debug":
			l = colorize(ll, colorYellow)
		case "info":
			l = colorize(ll, colorGreen)
		case "warn":
			l = colorize(ll, colorRed)
		case "error":
			l = colorize(colorize(ll, colorRed), colorBold)
		case "fatal":
			l = colorize(colorize(ll, colorRed), colorBold)
		case "panic":
			l = colorize(colorize(ll, colorRed), colorBold)
		default:
			l = colorize("???", colorBold)
		}
	} else {
		if i == nil {
			l = colorize("???", colorBold)
		} else {
			l = strings.ToUpper(fmt.Sprintf("%s", i))[0:3]
		}
	}
	return l
}

// Initialize Configure a zos app
func Initialize() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		TimeFormat:  time.RFC3339,
		Out:         os.Stdout,
		FormatLevel: formatLevel,
	})

	env, err := environment.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse node environment")
	}

	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// If running dev, enable debug mode
	if env.RunningMode == environment.RunningDev {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}
