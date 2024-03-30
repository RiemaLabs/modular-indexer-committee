#!/bin/bash

timestamp=$(date +"%Y%m%d-%H%M%S")
output_file="cjk_${timestamp}.log"

tmp_file=$(mktemp)
# Assume all files are coded in UTF-8.
# Perl support searching for Chinese characters in all planes, not only BMP.
# Including full result with filenames and line numbers of recursive search via perl
find . -type f -exec file {} + | grep text | cut -d: -f1 | \
    xargs perl -Mopen=locale -ne 'print "$ARGV:$.\t$_" if /[\x{4e00}-\x{9fa5}\x{20000}-\x{2a6df}\x{2a700}-\x{2b73f}\x{2b740}-\x{2b81f}\x{2b820}-\x{2ceaf}\x{2ceb0}-\x{2ebef}\x{2ebf0}-\x{2fbff}]/' \
    2>&1 >> ${tmp_file}

if [ -s "$tmp_file" ]; then
    echo "Warning: Probable Chinese comments found! See $output_file for details."
else
    echo "No Chinese characters found." >> ${tmp_file}
fi

cp ${tmp_file} ${output_file}
rm ${tmp_file}

cat ${output_file}