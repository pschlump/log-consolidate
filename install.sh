#!/bin/bash

if [ "$(whoami)" == "root" ] ; then
	:
else
	echo "Usage: !! run as root"
	exit 1
fi

cp log-consolidate-init.sh /etc/init.d/log-consolidate
cd /etc
ln -s /etc/init.d/log-consolidate ./rc0.d/K90log-consolidate
ln -s /etc/init.d/log-consolidate ./rc1.d/K90log-consolidate
ln -s /etc/init.d/log-consolidate ./rc2.d/S90log-consolidate
ln -s /etc/init.d/log-consolidate ./rc3.d/S90log-consolidate
ln -s /etc/init.d/log-consolidate ./rc4.d/S90log-consolidate
ln -s /etc/init.d/log-consolidate ./rc5.d/S90log-consolidate
ln -s /etc/init.d/log-consolidate ./rc6.d/K90log-consolidate

