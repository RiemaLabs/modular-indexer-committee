SELECT block_height, block_hash FROM public.brc20_block_hashes
WHERE block_height >= 779832 AND block_height <= 779838
ORDER BY id ASC