#! /bin/bash
### BEGIN INIT INFO
# Provides:		log-consolidate
# Required-Start:	$syslog $remote_fs
# Required-Stop:	$syslog $remote_fs
# Should-Start:		$local_fs
# Should-Stop:		$local_fs
# Default-Start:	2 3 4 5
# Default-Stop:		0 1 6
# Short-Description:	log-consolidate - HTTP server
# Description:		log-consolidate - HTTP server
### END INIT INFO

# chagne all lines with "pschlump" in them to reflect where you installed this

PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin
DAEMON=/home/pschlump/go/src/www-2c-why.com/log-consolidate/log-consolidate
NAME=log-consolidate
DESC=log-consolidate

RUNDIR=/home/pschlump/go/src/www-2c-why.com/log-consolidate
PIDFILE=$RUNDIR/log-consolidate.pid

test -x $DAEMON || exit 0

if [ -r /etc/default/$NAME ]
then
	. /etc/default/$NAME
fi

. /lib/lsb/init-functions

set -e

case "$1" in
  start)
	echo -n "Starting $DESC: "
	mkdir -p $RUNDIR
	touch $PIDFILE

	cd $RUNDIR 

	$DAEMON read > ,log 2>&1 & 
	THE_PID=$! 
	echo "$THE_PID" >$PIDFILE
	;;

  stop)
	echo -n "Stopping $DESC: "
	if [ -f $PIDFILE ] ; then
		#touch -f /tmp/$NAME.stop
		kill $( cat $PIDFILE )
		rm -f $PIDFILE
	fi
	sleep 1
	;;

  restart|force-reload)
	${0} stop
	${0} start
	;;

  status)
	echo "Unknown:TBD"
	;;

  *)
	echo "Usage: /etc/init.d/$NAME {start|stop|restart|force-reload|status}" >&2
	exit 1
	;;
esac

exit 0
