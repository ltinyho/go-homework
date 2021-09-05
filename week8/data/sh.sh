#!/bin/bash

for i in {1..3};
do
  for j in 10 20 50 100 200 1024 5120;do
    touch $j.csv
    redis-benchmark -t get,set --csv -d $j >> $j.csv
  done;
done;
