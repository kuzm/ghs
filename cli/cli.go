package cli

import (
	"os"

	"github.com/urfave/cli"
)

func Start() {
	app := cli.NewApp()
	app.Name = "Github Review Statistics"
	app.Usage = "Displays statistics for reviews KPIs"
	app.Version = "0.0.1"
	app.Authors = []cli.Author{{Name: "Ondrej Burkert"}}

	flags := []cli.Flag{
		cli.StringFlag{
			Name:  "org",
			Value: "",
			Usage: "Organization",
		},
		cli.StringFlag{
			Name:  "team",
			Value: "",
			Usage: "Team name",
		},
		cli.StringFlag{
			Name:  "owner",
			Value: "",
			Usage: "Owner (if organization and team are not specified)",
		},
		cli.IntFlag{
			Name:  "start, s",
			Value: -14,
			Usage: "Start of range - days from now. E. g. -14",
		},
		cli.IntFlag{
			Name:  "end, e",
			Value: -1,
			Usage: "End of range - days from now. E. g. -7",
		},
		cli.StringFlag{
			Name:  "api-token, t",
			Value: "",
			Usage: "Github API token with repo and org scope",
		},
		cli.StringFlag{
			Name:  "file, f",
			Value: "",
			Usage: "Path to output CSV file",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "stats",
			Usage:  "Get Statistics",
			Action: stats,
			Flags:  flags,
		},
	}

	app.Run(os.Args)
}
