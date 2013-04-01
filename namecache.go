package main

import (
	"fmt"
	"log"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	ncPeriod = 1 * time.Hour
)

var (
	ncPath = filepath.Join(hkHome, "apps")
	ncPart = filepath.Join(hkHome, "apps.part")
)

func ncUpdate() {
	if ncWantUpdate() {
		l := exec.Command("logger", "-thk")
		c := exec.Command("hk", "cachenames")
		if w, err := l.StdinPipe(); err == nil && l.Start() == nil {
			c.Stdout = w
			c.Stderr = w
		}
		c.Start()
	}
}

func ncWantUpdate() bool {
	s, err := os.Stat(ncPath)
	if os.IsNotExist(err) {
		return true
	}
	return time.Since(s.ModTime()) > ncPeriod
}

var cmdCacheNames = &Command{
	Run:   runCacheNames,
	Usage: "cachenames",
	Long: `
Cachenames fetches the current list of app names and saves it to
$HOME/.hk/apps.

This command is unlisted, since users never have to run it directly.
`,
}

func runCacheNames(cmd *Command, args []string) {
	defer os.Exit(0) // prevent deferred ncUpdate from running in main
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "Too many args. Run 'hk help cachenames'.")
		os.Exit(2)
	}
	f, err := os.OpenFile(ncPart, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	switch {
	case os.IsNotExist(err): // missing dir
		os.Mkdir(hkHome, 0777)
		return
	case os.IsExist(err): // .part file exists; stale?
		if s, err1 := os.Stat(ncPart); err1 != nil {
			log.Println(err1)
		} else if time.Since(s.ModTime()) > ncPeriod {
			os.Remove(ncPart)
		}
		return
	case err != nil:
		log.Fatal(err)
	}
	var apps []*App
	if err = Get(&apps, "/apps"); err != nil {
		os.Remove(f.Name())
		log.Fatal(err)
	}
	sort.Sort(appsByName(apps))
	for _, app := range apps {
		if _, err = io.WriteString(f, app.Name+"\n"); err != nil {
			os.Remove(f.Name())
			log.Fatal(err)
		}
	}
	if err = os.Rename(f.Name(), ncPath); err != nil {
		os.Remove(f.Name())
		log.Fatal(err)
	}
}
