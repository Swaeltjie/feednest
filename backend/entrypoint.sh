#!/bin/sh
chown -R feednest:feednest /data
exec su-exec feednest ./feednest
