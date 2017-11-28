#!/bin/bash

DB_DIR=$(cd $(dirname $0) && pwd)
cd $DB_DIR

for i in $(seq 0 1); do
dbname=isu7fportal_day$i
mysql -uroot -e "DROP DATABASE IF EXISTS $dbname; CREATE DATABASE $dbname;"
mysql -uroot $dbname < ./schema.sql
mysql -uroot $dbname -e "INSERT INTO setting (name, value) VALUES ('day', $i)"
done
