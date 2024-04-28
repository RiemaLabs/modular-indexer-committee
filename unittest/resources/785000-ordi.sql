SELECT DISTINCT ON (block_height, pkscript, tick) *
FROM public.brc20_historic_balances
WHERE block_height <= 785000 AND tick = 'ordi'
ORDER BY block_height, pkscript, tick, id DESC;
