package LogConsolidateLib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pschlump/Go-FTL/server/sizlib"
	"github.com/pschlump/radix.v2/redis"
)

type ReadType struct {
	FileToRead     string `json:"File"`
	PrefixDataWith string // add stuff to front of data to can identify were it came from - template %{ip%} %{timestamp%} %{hostname%}
	Key            string //
}

type RedisConnectType struct {
	RedisHost string `json:"RedisHost"`
	RedisPort string `json:"RedisPort"`
	RedisAuth string `json:"RedisAuth"`
}

type DefaultType struct {
	Key             string
	MaxMsg          int
	MaxListSize     int
	BackupLocalFile string
	OutputFile      string // default output file if 'write'
}

type ConfigType struct {
	Read         []ReadType        // set of files to read
	RedisConnect *RedisConnectType // destination to send data to
	Default      DefaultType       // default key in Redis to send data to
	IAmAlive     bool              // if true send periodic messages to monitor to say that this service is up and running
	StatusPort   string            // host:port to listen to for request from the process monitor for /api/status and /api/test (IAmAlive must be true)
	EFile        *os.File          `json:"-"`
}

// ReadConfig reads in the configuration file and replaces values with defaults if they were not specified.
func ReadConfig(cfgFn, HostName string) (cfg ConfigType) {
	mdata := make(map[string]string)
	mdata["hostname"] = HostName
	fn := sizlib.Qt(cfgFn, mdata)
	fmt.Printf("Configuration File Used : %s\n", fn)
	s, err := ioutil.ReadFile(fn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal! Unable to read configuration file %s, error=%s\n", cfgFn, err)
		os.Exit(1)
	}
	err = json.Unmarshal(s, &cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal! Unable to unmarshal/parse configuration file %s, error=%s\n", cfgFn, err)
		os.Exit(1)
	}
	// add in defaults for not-set values
	if cfg.RedisConnect != nil {
		if cfg.RedisConnect.RedisHost == "" {
			cfg.RedisConnect.RedisHost = "127.0.0.1"
		}
		if cfg.RedisConnect.RedisPort == "" {
			cfg.RedisConnect.RedisPort = "6379"
		}
	}
	if cfg.Default.MaxListSize == 0 {
		cfg.Default.MaxListSize = 50000
	}
	if cfg.Default.BackupLocalFile == "" {
		cfg.Default.BackupLocalFile = "./backup.log"
	}
	if cfg.Default.Key == "" {
		cfg.Default.Key = "log:"
	}
	return
}

// RedisClient connect to Redis, optionally using authorization.  Return the client and true if connected.
func RedisClient(RedisHost, RedisPort, RedisAuth string) (client *redis.Client, conFlag bool) {
	var err error
	conFlag = true
	client, err = redis.Dial("tcp", RedisHost+":"+RedisPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error on connect to redis, %s\n", err)
		return nil, false
	}
	if RedisAuth != "" {
		err = client.Cmd("AUTH", RedisAuth).Err
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error on auth to redis, %s\n", err)
			return client, false
		}
	}
	return
}
