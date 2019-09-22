package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zosv2/modules/upgrade"

	"github.com/blang/semver"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Usage = "upgradectl help to generate proper upgraded files for upgraded"
	app.Flags = []cli.Flag{}

	app.Commands = []cli.Command{
		{
			Name:        "release",
			Aliases:     []string{"r"},
			Usage:       "release an flist to given name and version",
			Description: "This command simply moves the given `flist` to `release:version.flist` and makes sure that `release:latest.flist` points to it.",
			ArgsUsage:   "<version>",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "release, r",
					Usage: "published release name (output)",
				},
				cli.StringFlag{
					Name:  "flist, f",
					Usage: "the flist name to release (input)",
				},
				cli.StringFlag{
					Name:  "jwt, t",
					Usage: "iyo token",
				},
			},
			Action: release,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

func release(c *cli.Context) error {
	var (
		flist   = c.String("flist")
		release = c.String("release")
		version = c.Args().First()
		jwt     = c.String("jwt")
	)

	if flist == "" {
		return fmt.Errorf("flist must be specified")
	}

	if release == "" {
		return fmt.Errorf("release must be specified")
	}

	if jwt == "" {
		return fmt.Errorf("jwt must be specified")
	}

	if version == "" {
		return fmt.Errorf("version must be specified")
	}

	user, err := JWTUser(jwt)
	if err != nil {
		return err
	}

	fmt.Println("user:", user)
	v, err := semver.Parse(version)
	if err != nil {
		return err
	}

	if err := appendVersion(v); err != nil {
		return err
	}

	return writeUpgrade(v, upgrade.Upgrader{})
}

func writeUpgrade(v semver.Version, u upgrade.Upgrader) error {
	b, err := json.Marshal(u)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(v.String(), b, 0660); err != nil {
		return err
	}

	b, err = json.Marshal(v)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("latest", b, 0660)
}

func appendVersion(v semver.Version) error {
	versions := []semver.Version{}
	b, err := ioutil.ReadFile("versions")
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// versions list does not exist yet, let's created it
		if os.IsNotExist(err) {
			versions = append(versions, v)
			b, err := json.Marshal(versions)
			if err != nil {
				return err
			}
			return ioutil.WriteFile("versions", b, 0660)
		}
	}

	if err := json.Unmarshal(b, &versions); err != nil {
		return err
	}

	versions = append(versions, v)
	b, err = json.Marshal(versions)
	if err != nil {
		return err
	}

	return ioutil.WriteFile("versions", b, 0660)
}
