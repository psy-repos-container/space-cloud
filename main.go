package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/urfave/cli"

	"github.com/spaceuptech/space-cloud/config"
)

func main() {
	app := cli.NewApp()
	app.Version = buildVersion
	app.Name = "space-cloud-ee"
	app.Usage = "core binary to run space cloud"

	app.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "runs the space cloud instance",
			Action: actionRun,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "port",
					Value: "8080",
					Usage: "Start HTTP server on port `PORT`",
				},
				cli.StringFlag{
					Name:  "grpc-port",
					Value: "8081",
					Usage: "Start grpc on port `GRPC_PORT`",
				},
				cli.IntFlag{
					Name:  "nats-port",
					Value: 4222,
					Usage: "Start nats on port `NATS_PORT`",
				},
				cli.IntFlag{
					Name:  "cluster-port",
					Value: 4248,
					Usage: "Start nats on port `NATS_PORT`",
				},
				cli.StringFlag{
					Name:   "db",
					Value:  "mongo",
					Usage:  "Load space cloud config from `DB`",
					EnvVar: "DB",
				},
				cli.StringFlag{
					Name:   "conn",
					Value:  "mongodb://localhost:27017",
					Usage:  "The connection string to connect to config db",
					EnvVar: "CONN",
				},
				cli.StringFlag{
					Name:   "account",
					Value:  "none",
					Usage:  "Start space-cloud with `ACCOUNT`",
					EnvVar: "ACCOUNT",
				},
				cli.BoolFlag{
					Name:   "prod",
					Usage:  "Run space-cloud in production mode",
					EnvVar: "PROD",
				},
				cli.BoolFlag{
					Name:   "disable-metrics",
					Usage:  "Disable anonymous metric collection",
					EnvVar: "DISABLE_METRICS",
				},
				cli.BoolFlag{
					Name:   "disable-nats",
					Usage:  "Disable embedded nats server",
					EnvVar: "DISABLE_NATS",
				},
				cli.StringFlag{
					Name:   "seeds",
					Value:  "none",
					Usage:  "Seed nodes to cluster with",
					EnvVar: "SEEDS",
				},
			},
		},
		{
			Name:   "init",
			Usage:  "creates a config file with sensible defaults",
			Action: actionInit,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func actionRun(c *cli.Context) error {
	// Load cli flags
	port := c.String("port")
	grpcPort := c.String("grpc-port")
	conn := c.String("conn")
	db := c.String("db")
	account := c.String("account")
	natsPort := c.Int("nats-port")
	clusterPort := c.Int("cluster-port")
	isProd := c.Bool("prod")
	disableMetrics := c.Bool("disable-metrics")
	disableNats := c.Bool("disable-nats")
	seeds := c.String("seeds")

	if account == "none" {
		return errors.New("Cannot start space-cloud with no account")
	}

	// Project and env cannot be changed once space cloud has started
	s := initServer(isProd)

	if !disableNats {
		// TODO read nats config from the yaml file if it exists
		if seeds != "" {
			array := strings.Split(seeds, ",")
			urls := []*url.URL{}
			for _, v := range array {
				if v != "" {
					u, err := url.Parse("nats://" + v)
					if err != nil {
						return err
					}
					urls = append(urls, u)
				}
			}
			defaultNatsOptions.Routes = urls
		}
		defaultNatsOptions.Port = natsPort
		defaultNatsOptions.Cluster.Port = clusterPort
		s.runNatsServer(defaultNatsOptions)
		fmt.Println("Started nats server on port ", defaultNatsOptions.Port)
	}

	err := s.projects.LoadConfigFromDB(account, db, conn)
	if err != nil {
		return err
	}

	// Anonymously collect usage metrics if not explicitly disabled
	if !disableMetrics {
		go s.routineMetrics()
	}

	s.routes()
	return s.start(port, grpcPort)
}

func actionInit(*cli.Context) error {
	return config.GenerateConfig()
}
