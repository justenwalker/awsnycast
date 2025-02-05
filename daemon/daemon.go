package daemon

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/middleware"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	ec2type "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	log "github.com/sirupsen/logrus"

	"github.com/justenwalker/awsnycast/aws"
	"github.com/justenwalker/awsnycast/config"
	"github.com/justenwalker/awsnycast/instancemetadata"
)

type Daemon struct {
	oneShot           bool
	noop              bool
	ConfigFile        string
	Version           string
	Debug             bool
	Config            *config.Config
	MetadataFetcher   instancemetadata.MetadataFetcher
	RouteTableManager aws.RouteTableManager
	quitChan          chan bool
	loopQuitChan      chan bool
	FetchWait         time.Duration
	instancemetadata.InstanceMetadata
}

func (d *Daemon) setupMetadataFetcher() {
	if d.MetadataFetcher == nil {
		d.MetadataFetcher = instancemetadata.New(d.Debug)
	}
}

func (d *Daemon) Setup() error {
	d.setupMetadataFetcher()
	im, err := instancemetadata.FetchMetadata(d.MetadataFetcher)
	if err != nil {
		return err
	}
	d.InstanceMetadata = im

	if d.RouteTableManager == nil {
		mw := middleware.AddUserAgentKey(fmt.Sprintf("awsnycast/%s", d.Version))
		cfg, err := awsconfig.LoadDefaultConfig(context.Background())
		if err != nil {
			return err
		}
		cfg.APIOptions = append(cfg.APIOptions, mw)
		if d.Region != "" {
			cfg.Region = d.Region
		}
		if d.Debug {
			cfg.ClientLogMode = awsv2.LogRequest | awsv2.LogResponseWithBody
		}
		cfg.RetryMode = awsv2.RetryModeStandard
		cfg.RetryMaxAttempts = 3
		d.RouteTableManager = aws.NewRouteTableManagerEC2(cfg)
	}

	config, err := config.New(d.ConfigFile, d.InstanceMetadata, d.RouteTableManager)
	if err != nil {
		return err
	}
	d.Config = config

	if d.FetchWait == 0 {
		d.FetchWait = time.Second * time.Duration(config.PollTime)
	}

	return setupHealthchecks(d.Config)
}

func setupHealthchecks(c *config.Config) error {
	for _, v := range c.Healthchecks {
		err := v.Setup()
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Daemon) runHealthChecks() {
	log.Debug("Starting healthchecks")
	for _, v := range d.Config.Healthchecks {
		v.Run(d.Debug)
	}
	for _, configRouteTables := range d.Config.RouteTables {
		for _, mr := range configRouteTables.ManageRoutes {
			mr.StartHealthcheckListener(d.noop)
		}
	}
	log.Debug("Started all healthchecks")
}

func (d *Daemon) stopHealthChecks() {
	for _, v := range d.Config.Healthchecks {
		v.Stop()
	}
}

func (d *Daemon) RunOneRouteTable(ctx context.Context, rt []ec2type.RouteTable, name string, configRouteTable *config.RouteTable) error {
	if err := configRouteTable.UpdateEc2RouteTables(ctx, rt); err != nil {
		return err
	}
	return configRouteTable.RunEc2Updates(ctx, d.RouteTableManager, d.noop)
}

func (d *Daemon) RunRouteTables(ctx context.Context) error {
	rt, err := d.RouteTableManager.GetRouteTables(ctx)
	if err != nil {
		return err
	}
	for name, configRouteTables := range d.Config.RouteTables {
		if err := d.RunOneRouteTable(ctx, rt, name, configRouteTables); err != nil {
			return err
		}
	}
	return nil
}

func (d *Daemon) Run(ctx context.Context, oneShot bool, noop bool) int {
	d.oneShot = oneShot
	d.noop = noop
	if err := d.Setup(); err != nil {
		log.WithFields(log.Fields{"err": err.Error()}).Error("Error in initial setup")
		return 1
	}

	if !d.RouteTableManager.InstanceIsRouter(ctx, d.Instance) {
		log.WithFields(log.Fields{"instance_id": d.Instance}).Error("I am not a router (do not have src/destination checking disabled)")
		return 1
	}

	d.quitChan = make(chan bool, 1)
	d.runHealthChecks()
	defer d.stopHealthChecks()
	err := d.RunRouteTables(ctx)
	if err != nil {
		log.WithFields(log.Fields{"err": err.Error()}).Error("Error in initial route table run")
		return 1
	}
	d.loopQuitChan = make(chan bool, 1)
	if oneShot {
		d.quitChan <- true
	} else {
		d.RunSleepLoop()
	}
	<-d.quitChan
	d.loopQuitChan <- true
	return 0
}

func (d *Daemon) RunSleepLoop() {
	go func() {

		ticker := time.NewTicker(d.FetchWait)
		fetch := ticker.C

		for {
			select {
			case <-d.loopQuitChan:
				ticker.Stop()
				return
			case <-fetch:
				err := d.RunRouteTables(context.Background())
				if err != nil {
					log.WithFields(log.Fields{"err": err.Error()}).Warn("Error in route table poll run")
				}
			}
		}
	}()
}
