// Command mobiledoc-to-markdown converts Mobiledoc format articles to
// Markdown.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jbarone/mobiledoc"
)

type figureData struct {
	Src     string
	Caption string
}

var figureTemplate = template.Must(template.New("figure").Parse(`
<figure>
  <img src="{{.Src}}">
  {{with .Caption}}<figcaption>{{.}}</figcaption>{{end}}
</figure>
`))

func renderImage(src string, caption string) ([]byte, error) {
	var buf bytes.Buffer
	if err := figureTemplate.Execute(&buf, figureData{Src: src, Caption: caption}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type Document struct {
	Title     string `json:"title"`
	Mobiledoc string `json:"mobiledoc"`
}

func convert(r io.Reader, w io.Writer, useFigure bool) error {
	var doc Document
	dec := json.NewDecoder(r)
	if err := dec.Decode(&doc); err != nil {
		return fmt.Errorf("error decoding input JSON: %v", err)
	}

	mdoc := mobiledoc.NewMobiledoc(strings.NewReader(doc.Mobiledoc)).
		WithCard("image", func(payload interface{}) string {
			// TODO this is an insane API. No errors, forcing
			// map[string]interface{} on us. Plus, shouldn't this code reside
			// in the library? Images are pretty standard.
			data := payload.(map[string]interface{})
			src := data["src"].(string)
			caption, ok := data["caption"]
			if !ok {
				caption = ""
			}
			if useFigure {
				buf, err := renderImage(src, caption.(string))
				if err != nil {
					panic(fmt.Errorf("renderImage: %v", err))
				}
				return string(buf)
			} else {
				// TODO hope for the best on escaping
				return fmt.Sprintf("![%s](%s)\n", caption, src)
			}
		}).
		WithCard("gallery", func(payload interface{}) string {
			var result strings.Builder
			data := payload.(map[string]interface{})
			images := data["images"].([]interface{})
			for _, imageData := range images {
				// The keys are: fileName row width height src.
				// Of these, fileName seems unnecessary, src covers that.
				// We're ignoring the layout clues from row width heigh, at least for now.
				image := imageData.(map[string]interface{})
				src := image["src"].(string)
				if useFigure {
					buf, err := renderImage(src, "")
					if err != nil {
						panic(fmt.Errorf("renderImage: %v", err))
					}
					result.Write(buf)
				} else {
					// TODO hope for the best on escaping
					fmt.Fprintf(&result, "![%s](%s)\n", "", src)
				}
			}
			return result.String()
		}).
		WithCard("markdown", func(payload interface{}) string {
			m := payload.(map[string]interface{})
			return m["markdown"].(string)
		}).
		WithCard("html", func(payload interface{}) string {
			m := payload.(map[string]interface{})
			return m["html"].(string)
		})

	if doc.Title != "" {
		if _, err := io.WriteString(w, "# "+doc.Title+"\n\n"); err != nil {
			return fmt.Errorf("cannot write to output: %v", err)
		}
	}
	if err := mdoc.Render(w); err != nil {
		return err
	}
	return nil
}

func processFile(path string, useFigure bool) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open input file: %v", err)
	}
	defer f.Close()
	return convert(f, os.Stdout, useFigure)
}

func processStdin(useFigure bool) error {
	return convert(os.Stdin, os.Stdout, useFigure)
}

var prog = filepath.Base(os.Args[0])

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", prog)
	fmt.Fprintf(os.Stderr, "  %s [OPTS] [FILE]\n", prog)
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
}

var useFigure = flag.Bool("use-figure", false, "Render images with HTML figure tag")

func main() {
	log.SetFlags(0)
	log.SetPrefix(prog + ": ")

	flag.Usage = usage
	flag.Parse()
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}

	if flag.NArg() == 0 {
		if err := processStdin(*useFigure); err != nil {
			log.Fatal(err)
		}
	} else {
		path := flag.Arg(0)
		if err := processFile(path, *useFigure); err != nil {
			log.Fatal(err)
		}
	}
}
