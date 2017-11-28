#!/bin/bash
cd $(dirname $0)
mysql --default-character-set=utf8mb4 -uroot < $(pwd)/drop.sql
mysql --default-character-set=utf8mb4 -uroot isudb < $(pwd)/isudb.sql
mysql --default-character-set=utf8mb4 -uroot isudb < $(pwd)/m_item.sql
