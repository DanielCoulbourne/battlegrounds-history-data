// Command bgh-convert turns a Hearthstone client log into Battlegrounds
// History Data files, one per game.
//
//	bgh-convert Power.log
//	bgh-convert -o out/ Power.log Power_old.log
//	cat Power.log | bgh-convert -stdout
//
// The client must be logging in the first place. Hearthstone reads
// %LOCALAPPDATA%\Blizzard\Hearthstone\log.config once at startup, and it needs
// a [Power] section with Verbose=true and FilePrinting=true. Without it the log
// carries none of the detail this reads, and the converter will say so rather
// than write an empty file.
//
// Display names are left out unless you pass -names. A battletag identifies a
// real person, and a recording gets shared more readily than a log does.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DanielCoulbourne/battlegrounds-history-data/converter/bgh"
	"github.com/DanielCoulbourne/battlegrounds-history-data/converter/convert"
)

func main() {
	out := flag.String("o", ".", "directory to write files into")
	toStdout := flag.Bool("stdout", false, "write one document to standard output instead of files")
	names := flag.Bool("names", false, "include players' display names (off by default: a battletag names a real person)")
	quiet := flag.Bool("q", false, "print nothing but errors")
	flag.Parse()

	if err := run(flag.Args(), *out, *toStdout, *names, *quiet); err != nil {
		fmt.Fprintln(os.Stderr, "bgh-convert:", err)
		os.Exit(1)
	}
}

func run(args []string, outDir string, toStdout, names, quiet bool) error {
	if len(args) == 0 {
		args = []string{"-"}
	}
	opt := convert.Options{
		IncludeNames: names,
		Recorder: bgh.Recorder{
			Name: "bgh-convert", Kind: bgh.RecorderTracker,
			URL: "https://github.com/DanielCoulbourne/battlegrounds-history-data",
		},
	}

	total := 0
	for _, path := range args {
		docs, err := readOne(path, opt)
		if err != nil {
			return err
		}
		for _, doc := range docs {
			if doc == nil {
				// A game the log never named a player for. There is no seat to
				// anchor a one-seat recording to, so there is nothing to write.
				continue
			}
			if err := doc.Validate(); err != nil {
				fmt.Fprintf(os.Stderr, "bgh-convert: %s: %v\n", doc.Recording.ID, err)
			}
			body, err := doc.Marshal()
			if err != nil {
				return err
			}
			total++
			if toStdout {
				os.Stdout.Write(body)
				continue
			}
			name := doc.Recording.ID + ".bgh.json"
			dest := filepath.Join(outDir, name)
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(dest, body, 0o644); err != nil {
				return err
			}
			if !quiet {
				fmt.Printf("%s  %d entries, %d seats\n", dest, len(doc.History), len(doc.Players))
			}
		}
	}
	if total == 0 {
		return fmt.Errorf("no Battlegrounds games found. Check that the client is logging: " +
			"%%LOCALAPPDATA%%\\Blizzard\\Hearthstone\\log.config needs a [Power] section " +
			"with Verbose=true and FilePrinting=true, read once when Hearthstone starts")
	}
	if !quiet && !toStdout {
		fmt.Printf("wrote %d game(s)\n", total)
	}
	return nil
}

func readOne(path string, opt convert.Options) ([]*bgh.Document, error) {
	if path == "-" {
		return convert.Convert(os.Stdin, opt)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	docs, err := convert.Convert(f, opt)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	if len(docs) == 0 && !strings.HasSuffix(path, ".log") {
		return nil, fmt.Errorf("%s: no games found; is this a Power.log?", filepath.Base(path))
	}
	return docs, nil
}
