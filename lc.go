package main

//
// TODO:
//		2. "prefix" by system - read, by writer etc.
//		2a. Testing
//		3. Test IAmAlive and Status
//		3a. Testing
// 		+3. Documentation README.md													1hr
//

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/pschlump/Go-FTL/server/sizlib"
	"github.com/pschlump/godebug"
	"github.com/pschlump/log-consolidate/lib"
	"github.com/pschlump/mon-alive/lib"
	"github.com/pschlump/radix.v2/redis"
	"github.com/urfave/cli"
)

type MoveData struct {
	Data string
}

func main() {

	app := cli.NewApp()
	app.Name = "log-consolidate"
	app.Usage = "log-consolidate -c cfg.json read|write"
	app.Version = "0.0.8"

	var redisLock = &sync.Mutex{}

	type commonConfig struct {
		MyStatus map[string]interface{} //
		Debug    map[string]bool        // make this a map[string]bool set of flags that you can turn on/off
	}

	cc := commonConfig{
		MyStatus: make(map[string]interface{}),
		Debug:    make(map[string]bool),
	}
	cc.MyStatus["cli"] = "y"

	var mkfifo = "/usr/bin/mkfifo"
	var client *redis.Client
	client = nil
	var ok bool
	var wg sync.WaitGroup
	var cfg LogConsolidateLib.ConfigType
	var mon *MonAliveLib.MonIt
	var redisError = false
	var HostName = ""
	var err error
	HostName, err = os.Hostname()
	if err != nil {
		HostName = "default"
		fmt.Fprintf(os.Stderr, "Unable to get the hostname, it will use 'default' instead error=%s\n", err)
	}
	message := make(chan MoveData)
	NSent := 0
	NErr := 0
	NRead := 0

	app.Before = func(c *cli.Context) error {

		Name := c.GlobalString("name")
		if Name != "" {
			mdata := make(map[string]string)
			mdata["hostname"] = HostName
			t := sizlib.Qt(Name, mdata)
			fmt.Printf("Application Name: %s\n", t)
			app.Name = t
		}

		DebugFlags := c.GlobalString("debug")
		ds := strings.Split(DebugFlags, ",")
		for _, dd := range ds {
			cc.Debug[dd] = true
		}

		// do setup - common function -- Need to be able to skip for i-am-alive remote!
		cfgFn := c.GlobalString("cfg")
		if cfgFn == "" {
			cfgFn = "%{hostname%}-cfg.json"
		}

		cfg = LogConsolidateLib.ReadConfig(cfgFn, HostName)
		if cc.Debug["dump-cfg"] {
			fmt.Printf("Cfg = %s\n", godebug.SVarI(cfg))
		}
		ok = false
		client, ok = LogConsolidateLib.RedisClient(cfg.RedisConnect.RedisHost, cfg.RedisConnect.RedisPort, cfg.RedisConnect.RedisAuth)
		if !ok {
			// Must run anyhow!
			redisError = true
		}
		if cfg.IAmAlive {
			// func NewMonIt(GetConn func() (conn *redis.Client), FreeConn func(conn *redis.Client)) (rv *MonIt) {
			mon = MonAliveLib.NewMonIt(func() *redis.Client { redisLock.Lock(); return client }, func(conn *redis.Client) { redisLock.Unlock() })
			mon.SendPeriodicIAmAlive(app.Name)
		}

		for _, vv := range cfg.Read {
			// TODO: if vv.FileToRead is a standard file then should rename it, and replace with FIFO
			if !sizlib.Exists(vv.FileToRead) {
				fmt.Printf("Warning: creating missing fifo: %s\n", vv.FileToRead)
				out, err := exec.Command(mkfifo, vv.FileToRead).Output()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: error running %s %s, error=%s\n", mkfifo, vv.FileToRead, err)
				}
				if string(out) != "" {
					fmt.Printf("Output From Creating FIFO: %s\n", out)
				}
			}
		}

		return nil
	}

	create_ReadFIFO := func() func(*cli.Context) error {
		return func(ctx *cli.Context) error {

			// Set Up Reader's push data onto channel
			wg.Add(len(cfg.Read))
			for _, vv := range cfg.Read {
				go func(rdr LogConsolidateLib.ReadType) {
					defer wg.Done()
					if cc.Debug["dump-read"] {
						fmt.Printf("Read From %s\n", rdr.FileToRead)
					}
					// do read in infinite loop
					fp, err := sizlib.Fopen(rdr.FileToRead, "r")
					if err != nil {
						fmt.Fprintf(os.Stderr, "Unable to open %s for input, error=%s\n", rdr.FileToRead, err)
					}
					defer fp.Close()

					for kk := 0; ; {
						reader := bufio.NewReader(fp)
						for {
							data, err := reader.ReadString('\n')
							if len(data) > 0 {
								if cc.Debug["dump-read"] {
									fmt.Printf("%d: sender data->%s<-\n", kk, data)
								}
								kk++
								message <- MoveData{Data: string(data)}
							}
							if err != nil || len(data) == 0 { // TODO - should check for error of EOF, if EOF that's ok, else report error
								// fmt.Printf("Sleep, err=%s\n", err)
								time.Sleep(200 * time.Millisecond)
								break
							}
						}
					}
				}(vv)
			}

			// Set up Writer(s) - pull from channel, send to Redis, later remote
			wg.Add(1)
			go func() {
				defer wg.Done()

				if redisError {
					NErr++
					d0 := "log-consolidate: Error: failed to connect to Redis\n"
					EmergencyBackup(&cfg, d0)
				}

				for kk := 0; ; {
					for md := range message {
						if cc.Debug["dump-write"] {
							fmt.Printf("Received %d: %s\n", kk, md.Data)
						}
						kk++

						if redisError {
							NErr++
							EmergencyBackup(&cfg, md.Data)
						} else {
							listKey := cfg.Default.Key

							redisLock.Lock()

							// LLEN length of list, if list grows too large then just write data to file.  If error just write to file.
							n, err := client.Cmd("LLEN", listKey).Int()
							if err == nil && n < cfg.Default.MaxListSize {
								// LPUSH data onto the list.
								err = client.Cmd("LPUSH", listKey, string(md.Data)).Err
								if err != nil {
									NErr++
									d0 := fmt.Sprintf("log-consolidate: Error: Redis LPUSH, %s, returned error %s\n", listKey, err)
									EmergencyBackup(&cfg, d0)
									EmergencyBackup(&cfg, md.Data)
								} else {
									NSent++
								}
							} else {
								NErr++
								d0 := fmt.Sprintf("log-consolidate: Error: Too many items in list, or error=%s, reader may be stuck, n_items=%d\n", err, n)
								EmergencyBackup(&cfg, d0)
								EmergencyBackup(&cfg, md.Data)
							}

							redisLock.Unlock()

						}
					}
				}
			}()

			return nil
		}
	}

	create_WriteFromList := func() func(*cli.Context) error {
		return func(ctx *cli.Context) error {

			FileName := ctx.String("filename")
			if FileName == "./consolidate-log.log" && cfg.Default.OutputFile != "" {
				FileName = cfg.Default.OutputFile
			}

			fp, err := sizlib.Fopen(FileName, "a")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to open consolidated log file, %s, error=%s -- output send to stderr\n", FileName, err)
				fp = os.Stderr
				NErr++
			}

			listKey := cfg.Default.Key
			for {
				redisLock.Lock()
				sss, err := client.Cmd("BRPOP", listKey, 0).List()
				redisLock.Unlock()
				NRead++
				if err != nil {
					fmt.Fprintf(os.Stderr, "Unable to pop data from Redis List, key=%s, error=%s\n", listKey, err)
					NErr++
				} else {
					for i, s := range sss {
						if i > 0 {
							fmt.Fprintf(fp, "%s\n", s)
						}
					}
				}
			}

			return nil
		}
	}

	create_CheckCfg := func() func(*cli.Context) error {
		return func(ctx *cli.Context) error {
			fmt.Printf("Syntax OK\n") // if it gets to this point then the syntax is GOOD.
			return nil
		}
	}

	if cfg.IAmAlive && cfg.StatusPort != "" {
		getStatus := func() string {
			return fmt.Sprintf(`{"status":%q, "n_err":%d, "n_sent":%d, "n_read":%d, "redisError":%v}`, NErr, NSent, NRead, redisError)
		}
		circuitTest := func() bool {
			var befNErr = NErr
			var befNSent = NSent
			var befNRead = NRead
			message <- MoveData{Data: `{"test":"test"}`}
			message <- MoveData{Data: `{"test":"test"}`}
			message <- MoveData{Data: `{"test":"test"}`}
			time.Sleep(300 * time.Millisecond)
			if befNErr < NErr {
				return false
			}
			if befNSent == NSent && befNRead == NRead {
				return false
			}
			return true
		}
		mon.SetupStatus(cfg.StatusPort, getStatus, circuitTest)
	}

	app.Commands = []cli.Command{
		{
			Name:   "read",
			Usage:  "Read FIFO/Named Pipes and put on Redis list.",
			Action: create_ReadFIFO(),
		},
		{
			Name:   "checkCfg",
			Usage:  "check configuration file for syntax.",
			Action: create_CheckCfg(),
		},
		{
			Name:   "write",
			Usage:  "Read Redis list and write data to file.",
			Action: create_WriteFromList(),
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "filename, f",
					Value: "./consolidate-log.log",
					Usage: "output file name",
				},
			},
		},
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "cfg, c",
			Value: "%{hostname%}-cfg.json",
			Usage: "Template Substituted Global Configuration File.",
		},
		cli.StringFlag{
			Name:  "name, n",
			Value: "%{hostname%}-log-consolidate",
			Usage: "Name user for mon-alive monitoring",
		},
		cli.StringFlag{
			Name:  "debug, D",
			Value: "",
			Usage: "Set debug flags [ show-feedback ]",
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	wg.Wait()
	client.Close()
}

// EmergencyBackup is the convert sending stuff to Redis to a local file in an emergency.
func EmergencyBackup(cfg *LogConsolidateLib.ConfigType, d0 string) {
	if cfg.EFile == nil {
		fn := cfg.Default.BackupLocalFile
		fx, err := sizlib.Fopen(fn, "a")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to open emergency file, %s, error=%s\n", fn, err)
			return
		}
		cfg.EFile = fx
	}
	fmt.Fprintf(cfg.EFile, "%s", d0)
}

/* vim: set noai ts=4 sw=4: */
