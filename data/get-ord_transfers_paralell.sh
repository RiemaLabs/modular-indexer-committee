#!/bin/bash

if [ $# -ne 2 ]; then
  echo "Usage: $0 <start_height> <num_heights>"
  exit 1
fi

eventUrl="https://www.okx.com/fullnode/brc20/nubit/public/rpc"

START_HEIGHT=$1
NUM_HEIGHTS=$2
END_HEIGHT=$((START_HEIGHT + NUM_HEIGHTS - 1))
CSV_FILE="${END_HEIGHT}-okx-brc20_block_hashes.csv"

if [ ! -f "$CSV_FILE" ]; then
  echo "CSV file $CSV_FILE does not exist."
  exit 1
fi

# Create a temporary directory
TMP_DIR=$(mktemp -d)
OUTPUT_CSV="${END_HEIGHT}-okx-ord_transfers.csv"
echo '"block_height","event_type","tick","inscription_id","inscription_num","old_satpoint","new_satpoint","from_address","to_address","valid","msg","supply","limit_per_mint","decimal","amount"' > $OUTPUT_CSV

# Split the work into 70 parts
split -l $(( (NUM_HEIGHTS + 69) / 70 )) $CSV_FILE $TMP_DIR/block_part_

# Function to process a part
process_part() {
  local part_file=$1
  local part_output="${part_file}_output.csv"

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

            echo "\"${block_height}\",\"${event_type}\",\"${tick}\",\"${inscription_id}\",\"${inscription_num}\",\"${old_satpoint}\",\"${new_satpoint}\",\"${from_address}\",\"${to_address}\",\"${valid}\",\"${msg}\",\"${supply}\",\"${limit_per_mint}\",\"${decimal}\",\"${amount}\"" >> $part_output
          done
        fi
      else
        echo "Failed to retrieve or parse events for block hash: $block_hash" >&2
      fi
    fi
  done < $part_file
}

export -f process_part
export eventUrl

# Run parts in parallel
ls $TMP_DIR/block_part_* | parallel process_part {}

# Concatenate all part outputs into the final CSV
for part_output in $TMP_DIR/block_part_*_output.csv; do
  cat $part_output >> $OUTPUT_CSV
done

# Clean up temporary directory
rm -rf $TMP_DIR

echo "CSV file $OUTPUT_CSV generated successfully."
