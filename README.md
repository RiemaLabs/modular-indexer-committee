# Bug detected:
1. In the `main.go`
return ord.NewQueues(getter, &header, true, catchupHeight+1)
->
return ord.NewQueues(getter, &header, true, catchupHeight)

In `queue.go`
ordTransfer, err := getter.GetOrdTransfers(i)
->
ordTransfer, err := getter.GetOrdTransfers(i+1)