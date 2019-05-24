#!/bin/bash
sudo mv /var/lib/mysql/isucon-s02581-slow.log /var/lib/mysql/history/isucon-s02581-slow-$(date +%m%d-%H%M%S).log
sudo mysqladmin flush-logs
/home/hashikawa_joichiro/cco/bench/bench --output result.json
