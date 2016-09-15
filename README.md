Log-consolidate - Consolidate logs from multiple sources and multiple systems.
=============================================================================

This program, depending on options, acts as either the reader for multiple log
files, or as a writer that takes consolidated log files and writes them out to
a single file on a single system.  It uses Redis lists to accomplish this.

Also it can search the logs for patterns and notify when a pattern is found.

The basic idea is to send log files to named pipes (fifo) on each system
and have the reader end pick up the data and send it to Redis on a List.
The other end takes the data from Redis and writes it to a file.

To create a named pipe on a Mac (OS X) and Linux:

	$ mkfifo <name>

The `mkfifo` creates a named file that can be written to by one process and read
by another.  If you are missing a FIFO this will attempt to run mkfifo to create
it.

[Linux Journal](http://www.linuxjournal.com/article/2156) provides a 
nice explanation of named pipes.

Windows - No Joy
----------------

Windows has named pipes that would require substantially different code to work.  
A different system would need to be implemented.



Sample Configuration
--------------------

This is a sample read configuration for a single machine running a server and 3 micro services.
Two machines are involved, 192.168.0.20,`where the server and 3 micro services are running.
The logs are consolidated on 192.168.0.157.

The File r-cfg.json is:

```JavaScript

	 {
		"RedisConnect": {
			  "RedisHost":  "192.168.0.133"
			, "RedisPort":  "6379"
			, "RedisAuth":  "lLJSmKCCYJixETskr8RM2avJaBM"
		}
		, "Read": [
				  { "File":"/var/log/server/log" }
				, { "File":"/var/log/ms-chat/log" }
				, { "File":"/var/log/ms-get/log" }
				, { "File":"/var/log/ms-report/log" }
		]
		, "Default": {
			  "Key": "log:"
			, "MaxMsg": 500
		}
	 }

```

The name of the Redis list is `log:`.  This is set with the Default.Key value.

`Read` lists the set of files that are to be read.

`RedisConnect` is the Redis connection information.   You can set the port that
Redis runs on with `"RedisPort":"6379"`.  If Redis is not using authentication
then leave `RedisAuth` our or set to `""`.

`MaxMsg` sets the size limit on the Redis list to 500.  If the list exceeds this size, then
the data will be written to a backup log file.  The default backup file is: `./consolidate.log.file`.

The writer configuration file is much simpler.

The File w-cfg.json is:

```JavaScript

	 {
		"RedisConnect": {
			  "RedisHost":  "192.168.0.133"
			, "RedisAuth":  "lLJSmKCCYJixETskr8RM2avJaBM"
		}
		, "Default": {
			"OutputFile":"./common-log.log"
		}
	 }

```

A pair of reader/writer is then run.  On the system with the server, and the micro services run:

```
	$ log-consolidate -c r-cfg.json read
```

On the system where the flog files are to be consolidated, run:

```
	$ log-consolidate -c w-cfg.json write 
```


Setup across machines
---------------------

This is a more realistic configuration. It is taken with very little modification from the set of servers
that I am running.  All of the systems need to be able to contact the same Redis instance.

Two web servers are configured to use DNS in a round robin.  These are `ws-01` and `ws-02`.
All of the configuration is stored in a single directory and identified by host name.

Three machines run micro services. These are `ms-01`, `ms-02` and `ms-03`.

On ws-01 the configuration file is ws-01-cfg.json:

```JavaScript

	 {
		"RedisConnect": {
			  "RedisHost":  "192.168.0.133"
			, "RedisAuth":  "lLJSmKCCYJixETskr8RM2avJaBM"
		}
		, "Read": [
			  { "File":"/var/log/server/log" }
		]
		, "Default": {
			  "Key": "log:"
			, "MaxMsg": 500
			, "Name":"ws-20-log-consolidate"
		}
	 }

```

This is run with:

```
	$ log-consolidate read
```


On ws-02, the second web server, the config is ws-02-cfg.json:

```JavaScript

	 {
		"RedisConnect": {
			  "RedisHost":  "192.168.0.133"
			, "RedisAuth":  "lLJSmKCCYJixETskr8RM2avJaBM"
		}
		, "Read": [
			  { "File":"/var/log/server/log" }
		]
		, "Default": {
			  "Key": "log:"
			, "MaxMsg": 500
			, "Name":"%{hostname%}-log-consolidate"
		}
	 }

```

This is run with:

```
	$ log-consolidate read
```

On ms-01, ms-02, ms-03, servers are configured with:

```JavaScript

	 {
		"RedisConnect": {
			  "RedisHost":  "192.168.0.133"
			, "RedisAuth":  "lLJSmKCCYJixETskr8RM2avJaBM"
		}
		, "Read": [
			  { "File":"/var/log/ms-chat/log" }
			, { "File":"/var/log/ms-report/log" }
		]
		, "Default": {
			  "Key": "log:"
			, "MaxMsg": 500
			, "Name":"%{hostname%}-log-consolidate"
		}
	 }

```

This is run on each of the systems with:

```
	$ log-consolidate read
```


The system that consolidates the log files then runs:

```JavaScript

	 {
		"RedisConnect": {
			  "RedisHost":  "192.168.0.133"
			, "RedisAuth":  "lLJSmKCCYJixETskr8RM2avJaBM"
		}
		, "Default": {
			"OutputFile":"./common-log.log"
		}
	 }

```

This is run with:

```
	$ log-consolidate -c w-cfg.json write 
```






Tests
-----

The tests are in the Makefile.  

```
	$ make test1 test2
```

License
-------

MIT

