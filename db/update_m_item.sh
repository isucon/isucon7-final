#!/bin/bash
cd $(dirname $0)
python3 tsv2sql.py < ../bench/data/m_item.tsv > m_item.sql 
