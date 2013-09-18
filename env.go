package main

import (
	"fmt"
	"log"
	"net/url"
	"sort"
	"strings"
)

var cmdEnv = &Command{
	Run:   runEnv,
	Usage: "env",
	Short: "list config vars",
	Long:  `Show all config vars.`,
}

func runEnv(cmd *Command, args []string) {
	var config map[string]string
	must(Get(&v2{&config}, "/apps/"+mustApp()+"/config_vars"))
	var configKeys []string
	for k := range config {
		configKeys = append(configKeys, k)
	}
	sort.Strings(configKeys)
	for _, k := range configKeys {
		fmt.Printf("%s=%s\n", k, config[k])
	}
}

var cmdGet = &Command{
	Run:   runGet,
	Usage: "get name",
	Short: "get config var" + extra,
	Long: `
Get the value of a config var.

Example:

  $ hk get BUILDPACK_URL
  http://github.com/kr/heroku-buildpack-inline.git
`,
}

func runGet(cmd *Command, args []string) {
	if len(args) != 1 {
		log.Fatal("Invalid usage. See 'hk help get'")
	}
	var config map[string]string
	must(Get(&v2{&config}, "/apps/"+mustApp()+"/config_vars"))
	value, found := config[args[0]]
	if !found {
		log.Fatalf("No such key as '%s'", args[0])
	}
	fmt.Println(value)
}

var cmdSet = &Command{
	Run:   runSet,
	Usage: "set name=value ...",
	Short: "set config var",
	Long: `
Set the value of a config var.

Example:

  $ hk set BUILDPACK_URL=http://github.com/kr/heroku-buildpack-inline.git
`,
}

func runSet(cmd *Command, args []string) {
	if len(args) < 1 {
		log.Fatal("Invalid usage. See 'hk help set'")
	}
	config := make(map[string]string)
	for _, arg := range args {
		i := strings.Index(arg, "=")
		if i < 0 {
			log.Fatalf("bad format: %#q. See 'hk help set'", arg)
		}
		config[arg[:i]] = arg[i+1:]
	}
	must(Put(v2nil, "/apps/"+mustApp()+"/config_vars", config))
}

var cmdUnset = &Command{
	Run:   runUnset,
	Usage: "unset name ...",
	Short: "unset config var",
	Long: `
Unset a config var.

Example:

  $ hk unset BUILDPACK_URL
`,
}

func runUnset(cmd *Command, args []string) {
	if len(args) < 1 {
		log.Fatal("Invalid usage. See 'hk help unset'")
	}
	for _, key := range args {
		q := url.QueryEscape(key)
		must(APIReq(v2nil, "DELETE", "/apps/"+mustApp()+"/config_vars/"+q, nil))
	}
}
