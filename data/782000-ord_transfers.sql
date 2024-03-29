SELECT ot.id, ot.inscription_id, ot.block_height, ot.old_satpoint, ot.new_satpoint, ot.new_pkscript, ot.new_wallet, ot.sent_as_fee, oc."content", oc.content_type
FROM ord_transfers ot
LEFT JOIN ord_content oc ON ot.inscription_id = oc.inscription_id
LEFT JOIN ord_number_to_id onti ON ot.inscription_id = onti.inscription_id
WHERE ot.block_height <= 782000
  AND onti.cursed_for_brc20 = false
  AND oc."content" is not null AND oc."content"->>'p' = 'brc-20'
ORDER BY ot.id ASC;
