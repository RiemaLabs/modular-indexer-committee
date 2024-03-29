#!/bin/bash

tmp_file=$(mktemp)
# including full result with filenames and line numbers of recursive search
grep -rnqHE --text '([\u4e00-\u9fa5]+)' . 2>&1 >> ${tmp_file}

if [ -s "$output_file" ]; then
    echo "Error: Probable Chinese comments found! See $output_file for details."
else
    echo "No Chinese characters found." >> ${tmp_file}
fi

timestamp=$(date +"%Y%m%d-%H%M%S")
output_file="cjk_${timestamp}.log"
cp ${tmp_file} ${output_file}
rm ${tmp_file}