// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"unicode"

	"github.com/TOSIO/go-tos/node"
	"github.com/TOSIO/go-tos/services/dashboard"

	"gopkg.in/urfave/cli.v1"

	//"github.com/TOSIO/go-ethereum/dashboard"
	"github.com/TOSIO/go-tos/app/utils"
	"github.com/TOSIO/go-tos/sdag"

	//"github.com/TOSIO/go-tos/params"
	//whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/naoina/toml"
)

var (
	dumpConfigCommand = cli.Command{
		Action:      utils.MigrateFlags(dumpConfig),
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(append(nodeFlags, rpcFlags...)),
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

type TOSstatsConfig struct {
	URL string `toml:",omitempty"`
}

type tosConfig struct {
	Sdag      sdag.Config
	Node      node.Config
	TOSstats  TOSstatsConfig
	Dashboard dashboard.Config
	//Shh       whisper.Config
}

func loadConfig(file string, cfg *tosConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

/*func loadConfigtest(filePath string, cfg *tosConfig) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(filePath + ", " + err.Error())
	}
	return err
}
*/
/*
func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit)
	cfg.HTTPModules = append(cfg.HTTPModules, "eth", "shh")
	cfg.WSModules = append(cfg.WSModules, "eth", "shh")
	cfg.IPCPath = "geth.ipc"
	return cfg
}
*/

// enableWhisper returns true in case one of the whisper flags is set.
/* func enableWhisper(ctx *cli.Context) bool {
	for _, flag := range whisperFlags {
		if ctx.GlobalIsSet(flag.GetName()) {
			return true
		}
	}
	return false
}
*/

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, cfg := makeConfigNode(ctx)
	comment := ""

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}
	io.WriteString(os.Stdout, comment)
	os.Stdout.Write(out)
	return nil
}
