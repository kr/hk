package main

import (
	"bufio"
	"code.google.com/p/go-netrc/netrc"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	Version = "0.8"
)

var (
	apiURL    = "https://api.heroku.com"
	hkHome    = filepath.Join(homePath, ".hk")
	netrcPath = filepath.Join(os.Getenv("HOME"), ".netrc")
	stdin     = bufio.NewReader(os.Stdin)
)

var updater = Updater{
	url: "https://hk.heroku.com/",
	dir: hkHome + "/update/",
}

type Command struct {
	// args does not include the command name
	Run  func(cmd *Command, args []string)
	Flag flag.FlagSet

	Usage string // first word is the command name
	Short string // `hk help` output
	Long  string // `hk help <cmd>` output
}

func (c *Command) printUsage() {
	if c.Runnable() {
		fmt.Printf("Usage: hk %s\n\n", c.Usage)
	}
	fmt.Println(strings.TrimSpace(c.Long))
}

func (c *Command) Name() string {
	name := c.Usage
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func (c *Command) Runnable() bool {
	return c.Run != nil
}

func (c *Command) ShowUsage() bool {
	return c.Short != ""
}

// Running `hk help` will list commands in this order.
var commands = []*Command{
	cmdCreate,
	cmdRename,
	cmdDestroy,
	cmdCreds,
	//cmdSSHCopyId, removed because it was breaking compile - RM
	cmdEnv,
	cmdUpdate,
	cmdGet,
	cmdSet,
	cmdUnset,
	cmdInfo,
	cmdRels,
	cmdList,
	cmdOpen,
	cmdPs,
	cmdScale,
	cmdRestart,
	cmdTail,
	cmdRun,
	cmdVersion,
	cmdHelp,

	helpEnviron,
	helpPlugins,
}

var (
	flagApp string // convience var for commands that need it
)

func main() {
	if s := os.Getenv("HEROKU_API_URL"); s != "" {
		apiURL = strings.TrimRight(s, "/")
	}
	if s := os.Getenv("HKURL"); s != "" {
		updater.url = strings.TrimRight(s, "/") + "/"
	}
	defer updater.run() // doesn't run if os.Exit is called
	log.SetFlags(0)

	args := os.Args[1:]
	if len(args) < 1 {
		usage()
	}

	for _, cmd := range commands {
		if cmd.Name() == args[0] && cmd.Run != nil {
			cmd.Flag.Usage = usage
			cmd.Flag.Parse(args[1:])
			cmd.Run(cmd, cmd.Flag.Args())
			return
		}
	}

	path := findPlugin(args[0])
	if path == "" {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
		usage()
	}
	err := execPlugin(path, args)
	log.Fatal("exec error: ", err)
}

func getCreds(u *url.URL) (user, pass string) {
	if u.User != nil {
		pw, _ := u.User.Password()
		return u.User.Username(), pw
	}

	m, err := netrc.FindMachine(netrcPath, u.Host)
	if err != nil {
		log.Fatalf("netrc error (%s): %v", u.Host, err)
	}

	return m.Login, m.Password
}

func app() (string, error) {
	if flagApp != "" {
		return flagApp, nil
	}

	b, err := exec.Command("git", "config", "remote.heroku.url").Output()
	if err != nil {
		if isNotFound(err) {
			wdir, _ := os.Getwd()
			return "", fmt.Errorf("could not find git remote heroku in %s", wdir)
		}
		return "", err
	}

	out := strings.Trim(string(b), "\r\n ")

	if !strings.HasPrefix(out, gitURLPre) || !strings.HasSuffix(out, gitURLSuf) {
		return "", fmt.Errorf("could not find app name in heroku git remote")
	}

	// Memoize for later use
	flagApp = out[len(gitURLPre) : len(out)-len(gitURLSuf)]

	return flagApp, nil
}

func isNotFound(err error) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		if ws, ok := ee.ProcessState.Sys().(syscall.WaitStatus); ok {
			return ws.ExitStatus() == 1
		}
	}
	return false
}

func mustApp() string {
	name, err := app()
	if err != nil {
		log.Fatal(err)
	}
	return name
}
