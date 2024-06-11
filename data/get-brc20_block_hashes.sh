#!/bin/bash

HASH_URL="https://hardworking-light-meme.btc.quiknode.pro/68929884d240f652"

if [ $# -ne 2 ]; then
  echo "Usage: $0 <start_height> <num_heights>"
  exit 1
fi

START_HEIGHT=$1
NUM_HEIGHTS=$2
END_HEIGHT=`expr $START_HEIGHT + $NUM_HEIGHTS - 1`
CSV_FILE="${END_HEIGHT}-okx-brc20_block_hashes.csv"

if [ -f "$CSV_FILE" ]; then
  echo "CSV file $CSV_FILE already exist."
  exit 1
fi

echo '"block_height","block_hash"' > $CSV_FILE

for (( i=0; i<NUM_HEIGHTS; i++ ))
do
  BLOCK_HEIGHT=$((START_HEIGHT + i))
  RESPONSE=$(curl -s -X POST $HASH_URL \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"getblockhash\",\"params\":[${BLOCK_HEIGHT}]}")
  BLOCK_HASH=$(echo $RESPONSE | jq -r '.result')

  echo "${BLOCK_HEIGHT},\"${BLOCK_HASH}\"" >> $CSV_FILE
done

echo "CSV file $CSV_FILE generated successfully."
