package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
)

var (
	follow bool
	lines  int
	source string
	dyno   string
)

var cmdTail = &Command{
	Run:   runTail,
	Usage: "tail [-f] [-n lines] [-s source] [-d dyno]",
	Short: "show the last part of the app log",
	Long:  `Tail prints recent application logs.`,
}

func init() {
	cmdTail.Flag.BoolVar(&follow, "f", false, "do not stop when end of file is reached")
	cmdTail.Flag.IntVar(&lines, "n", -1, "number of log lines to request")
	cmdTail.Flag.StringVar(&source, "s", "", "only display logs from the given source")
	cmdTail.Flag.StringVar(&dyno, "d", "", "only display logs from the given dyno or process type")
}

func runTail(cmd *Command, args []string) {
	var v struct {
		Dyno   string `json:"dyno,omitempty"`
		Lines  int    `json:"lines,omitempty"`
		Source string `json:"source,omitempty"`
		Tail   bool   `json:"tail,omitempty"`
	}

	v.Dyno = dyno
	v.Lines = lines
	v.Source = source
	v.Tail = follow

	var session struct {
		Id         string `json:"id"`
		LogplexURL string `json:"logplex_url"`
	}
	err := APIReq(&session, "POST", "/apps/"+mustApp()+"/log-sessions", v)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := http.Get(session.LogplexURL)
	if err != nil {
		log.Fatal(err)
	}
	must(checkResp(resp))

	writer := LineWriter(WriterAdapter{os.Stdout})

	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		if _, err = writer.Writeln(scanner.Text()); err != nil {
			log.Fatal(err)
			break
		}
	}

	resp.Body.Close()
}

type LineWriter interface {
	Writeln(p string) (int, error)
}

type WriterAdapter struct {
	io.Writer
}

func (w WriterAdapter) Writeln(p string) (n int, err error) {
	return fmt.Fprintln(w, p)
}

type colorizer struct {
	colors      map[string]string
	colorScheme []string
	filter      *regexp.Regexp
	writer      LineWriter
}

func newColorizer(writer LineWriter) *colorizer {
	return &colorizer{
		colors: make(map[string]string),
		colorScheme: []string{
			"36", //cyan
			"33", //yellow
			"32", //green
			"35", //magenta
			"31", //red
		},
		filter: regexp.MustCompile(`(?s)^(.*?\[([\w-]+)(?:[\d\.]+)?\]:)(.*)?$`),
		writer: writer,
	}
}

func (c *colorizer) resolve(p string) string {
	if color, ok := c.colors[p]; ok {
		return color
	}

	color := c.colorScheme[len(c.colors)%len(c.colorScheme)]
	c.colors[p] = color
	return color
}

func (c *colorizer) Writeln(p string) (n int, err error) {
	if c.filter.MatchString(p) {
		submatches := c.filter.FindStringSubmatch(p)
		return c.writer.Writeln(fmt.Sprintf("\033[%sm%s\033[0m%s", c.resolve(submatches[2]), submatches[1], submatches[3]))
	}

	return c.writer.Writeln(p)
}
