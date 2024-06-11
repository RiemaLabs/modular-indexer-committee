#!/bin/bash

if [ $# -ne 2 ]; then
  echo "Usage: $0 <start_height> <num_heights>"
  exit 1
fi

eventUrl="https://www.okx.com/fullnode/brc20/nubit/public/rpc"

START_HEIGHT=$1
NUM_HEIGHTS=$2
END_HEIGHT=`expr $START_HEIGHT + $NUM_HEIGHTS - 1`
CSV_FILE="${END_HEIGHT}-okx-brc20_block_hashes.csv"

if [ ! -f "$CSV_FILE" ]; then
  echo "CSV file $CSV_FILE does not exist."
  exit 1
fi

OUTPUT_CSV="${END_HEIGHT}-okx-ord_transfers.csv"
echo '"block_height","event_type","tick","inscription_id","inscription_num","old_satpoint","new_satpoint","from_address","to_address","valid","msg","supply","limit_per_mint","decimal","amount"' > $OUTPUT_CSV

while IFS=',' read -r block_height block_hash; do
  block_height=$(echo "$block_height" | tr -d '" ') 
  if [ "$block_height" != "block_height" ]; then
    block_hash=$(echo "$block_hash" | tr -d '" ')
    response=$(curl -s "${eventUrl}/api/v1/brc20/block/${block_hash}/events")

    if echo "$response" | jq -e . >/dev/null 2>&1; then
      events=$(echo "$response" | jq -c '.data.block[].events[]')

      if [ -n "$events" ]; then
        echo "$events" | while IFS= read -r event; do
          event_type=$(echo "$event" | jq -r '.type')
          tick=$(echo "$event" | jq -r '.tick')
          inscription_id=$(echo "$event" | jq -r '.inscriptionId')
          inscription_num=$(echo "$event" | jq -r '.inscriptionNumber')
          old_satpoint=$(echo "$event" | jq -r '.oldSatpoint')
          new_satpoint=$(echo "$event" | jq -r '.newSatpoint')
          from_address=$(echo "$event" | jq -r '.from.address')
          to_address=$(echo "$event" | jq -r '.to.address')
          valid=$(echo "$event" | jq -r '.valid')
          msg=$(echo "$event" | jq -r '.msg')
          supply=$(echo "$event" | jq -r '.supply // empty')
          limit_per_mint=$(echo "$event" | jq -r '.limitPerMint // empty')
          decimal=$(echo "$event" | jq -r '.decimal // empty')
          amount=$(echo "$event" | jq -r '.amount // empty')
          
          echo "\"${block_height}\",\"${event_type}\",\"${tick}\",\"${inscription_id}\",\"${inscription_num}\",\"${old_satpoint}\",\"${new_satpoint}\",\"${from_address}\",\"${to_address}\",\"${valid}\",\"${msg}\",\"${supply}\",\"${limit_per_mint}\",\"${decimal}\",\"${amount}\"" >> $OUTPUT_CSV
        done
      fi
    else
      echo "Failed to retrieve or parse events for block hash: $block_hash"
    fi
  fi
done < $CSV_FILE

echo "CSV file $OUTPUT_CSV generated successfully."
