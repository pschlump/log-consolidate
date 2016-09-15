#!/bin/bash

xx=$( ps -ef | grep log-consolidate | grep -v grep | grep testdata/test1 | awk '{print $2}' )
if [ "X$xx" == "X" ] ; then	
	:
else
	echo Killing PIDs $xx
	kill $xx
fi

