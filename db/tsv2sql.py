import sys
import csv

for row in csv.reader(sys.stdin, delimiter = '\t'):
    sys.stdout.write("INSERT INTO m_item VALUES(" + ", ".join(row) + ");\n")
