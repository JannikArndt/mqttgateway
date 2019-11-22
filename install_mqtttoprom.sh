#!/bin/bash

[ -z "$MQTTTOPROM_USER" ] && MQTTTOPROM_USER="mqtttoprom"
[ -z "$MQTTTOPROM_GROUP" ] && MQTTTOPROM_GROUP="mqtttoprom"
if ! getent group "$MQTTTOPROM_GROUP" > /dev/null 2>&1 ; then
    addgroup --system "$MQTTTOPROM_GROUP" --quiet
fi

mkdir -p /usr/share/mqtttoprom

if ! id $MQTTTOPROM_USER > /dev/null 2>&1 ; then
    adduser --system --home /usr/share/mqtttoprom --no-create-home \
	--ingroup "$MQTTTOPROM_GROUP" --disabled-password --shell /bin/false \
	"$MQTTTOPROM_USER"
fi

# Set user permissions on /var/log/mqtttoprom, /var/lib/mqtttoprom
mkdir -p /var/log/mqtttoprom /var/lib/mqtttoprom
chown -R $MQTTTOPROM_USER:$MQTTTOPROM_GROUP /var/log/mqtttoprom /var/lib/mqtttoprom
chmod 755 /var/log/mqtttoprom /var/lib/mqtttoprom

cp ./mqtttoprom /usr/local/bin/mqtttoprom
cp ./mqtttoprom.service /etc/systemd/system/mqtttoprom.service
sudo systemctl daemon-reload
sudo systemctl start mqtttoprom.service
sudo systemctl enable mqtttoprom.service