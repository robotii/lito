#!/bin/sh
set -e

# Run the specs
for file in ./specs/*.lito; do
    ./lito $file
done
