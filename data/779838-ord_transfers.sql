SELECT ot.id, ot.inscription_id, ot.block_height, ot.old_satpoint, ot.new_satpoint, ot.new_pkscript, ot.new_wallet, ot.sent_as_fee, oc."content", oc.content_type, onti.parent_id
		FROM ord_transfers ot
		LEFT JOIN ord_content oc ON ot.inscription_id = oc.inscription_id
		LEFT JOIN ord_number_to_id onti ON ot.inscription_id = onti.inscription_id
		WHERE (
			ot.inscription_id = '539e72be24670c7e3e65284474ab7b6b291757a8827dae97862ecd6ac8b6aa1di0'
			OR ot.inscription_id = '521c9ac8c6b5ebb2ef43a679f7b7cfb6eb9d1ee9d26e1269688974434cc929d6i0'
			OR ot.inscription_id = 'f402fecca8a3c67055e0c8650a93be857b8a4afb2e7cc807c46c2c07dfd93b6ci0'
			OR ot.inscription_id = '466b3c698e7c72b6d5a920bd9252d258d255f18067dc8de0ac0f848b3c7a2cbbi0'
			OR ot.inscription_id = '5b6766bd3c8e03a69ae8bec62a6030c190174a4dac2333f4fe2ef1dc0a58b1e7i0'
			OR ot.inscription_id = ''
			OR ot.inscription_id = ''
			OR ot.inscription_id = ''
			  )
			AND onti.cursed_for_brc20 = false
			AND oc."content" is not null AND oc."content"->>'p' = 'brc-20'
		ORDER BY ot.id asc;