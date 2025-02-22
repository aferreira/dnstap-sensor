/*
 * Copyright (c) 2017, 2019 Farsight Security, Inc.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"github.com/farsightsec/go-config"
)

// Config represents the global configuration of the client.
type Config struct {
	Servers       []config.URL    `yaml:"servers"`
	APIKey        config.String   `yaml:"api_key"`
	Channel       uint32          `yaml:"channel"`
	DnstapInput   dnstapInput     `yaml:"dnstap_input"`
	StatsInterval config.Duration `yaml:"stats_interval"`
	Heartbeat     config.Duration `yaml:"heartbeat"`
	Retry         config.Duration `yaml:"retry"`
	Flush         config.Duration `yaml:"flush"`
	Trace         bool            `yaml:"-"`
}

func loadConfig(conf *Config, filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	if err := Validate(b); err != nil {
		return err
	}
	return yaml.Unmarshal(b, conf)
}

func parseConfig(args []string) (conf *Config, err error) {
	var configFilename string
	var statsInterval, heartBeat, retry, flush config.Duration
	var apiKey config.String
	var inputSocket string
	var channel uint
	var trace bool

	fs := flag.NewFlagSet("dnstap-sensor", flag.ExitOnError)

	fs.StringVar(&configFilename, "config", "",
		"Location of client config file")
	fs.StringVar(&inputSocket, "input", "",
		"Path to dnstap input socket")
	fs.Var(&statsInterval, "stats_interval", "statistics logging interval (default 15m)")
	fs.Var(&heartBeat, "heartbeat", "heartbeat interval (default 30s)")
	fs.Var(&retry, "retry", "connection retry interval (default 30s)")
	fs.Var(&flush, "flush", "buffer flush interval (default 500ms)")
	fs.Var(&apiKey, "apikey", "apikey or path to apikey file")
	fs.UintVar(&channel, "channel", 0, "channel to upload dnstap data")
	fs.BoolVar(&trace, "trace", false, "log activity (verbose, recommended for debugging only)")
	fs.Parse(args)

	conf = new(Config)
	conf.StatsInterval.Set("15m")
	conf.Heartbeat.Set("30s")
	conf.Retry.Set("30s")
	conf.Flush.Set("500ms")

	if configFilename != "" {
		err = loadConfig(conf, configFilename)
		if err != nil {
			return
		}
	}

	if statsInterval.Duration != 0 {
		conf.StatsInterval = statsInterval
	}
	if heartBeat.Duration != 0 {
		conf.Heartbeat = heartBeat
	}
	if retry.Duration != 0 {
		conf.Retry = retry
	}
	if flush.Duration != 0 {
		conf.Flush = flush
	}
	if apiKey.String() != "" {
		conf.APIKey = apiKey
	}
	if inputSocket != "" {
		conf.DnstapInput = dnstapInput(inputSocket)
	}
	if channel != 0 {
		conf.Channel = uint32(channel)
	}
	conf.Trace = trace

	if fs.NArg() > 0 {
		servers := make([]config.URL, 0, flag.NArg())
		for _, s := range fs.Args() {
			u := config.URL{}
			if perr := u.Set(s); perr != nil {
				err = fmt.Errorf("Invalid URI %s: %v", s, perr)
				return
			}
			servers = append(servers, u)
		}
		conf.Servers = servers
	}

	if conf.Channel == 0 {
		err = errors.New("no channel specified")
	}
	if len(conf.Servers) == 0 {
		err = errors.New("no servers specified")
	}
	if conf.DnstapInput == "" {
		err = errors.New("no input specified")
	}
	if conf.APIKey.String() == "" {
		err = errors.New("no API key specified")
	}

	for _, u := range conf.Servers {
		switch u.Scheme {
		case "ws", "wss":
		default:
			err = fmt.Errorf("Invalid URI scheme %s in %s",
				u.Scheme, u)
			return
		}
	}

	return
}
