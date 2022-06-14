package main

import (
	"flag"
	"fmt"
	"log/syslog"
	"os"

	"github.com/justenwalker/awsnycast/daemon"
	"github.com/justenwalker/awsnycast/version"
	log "github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
)

var (
	debug        = flag.Bool("debug", false, "Enable debugging")
	f            = flag.String("f", "/etc/awsnycast.yaml", "Configration file")
	oneshot      = flag.Bool("oneshot", false, "Run route table manipulation exactly once, ignoring healthchecks, then exit")
	noop         = flag.Bool("noop", false, "Don't actually *do* anything, just print what would be done")
	printVersion = flag.Bool("version", false, "Print the version number")
	logToSyslog  = flag.Bool("syslog", false, "Log to syslog")
)

func main() {
	flag.Parse()
	if *printVersion {
		fmt.Printf("%s\n", version.Version)
		os.Exit(0)
	}
	d := new(daemon.Daemon)
	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	if *logToSyslog {
		hook, err := logrus_syslog.NewSyslogHook("", "", syslog.LOG_INFO, "")
		if err == nil {
			log.AddHook(hook)
		}
	}
	d.Debug = *debug
	d.ConfigFile = *f
	os.Exit(d.Run(*oneshot, *noop))
}
