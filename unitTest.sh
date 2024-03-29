cd ~/code/indexer-committee
go build
OUTPATH="./testres/"
OUTPUT="${OUTPATH}$(date +%Y-%m-%d-%H-%M-%S)_unitTests.log"
go test -v -run Test_CatchupStage 2>&1 >> $OUTPUT
go test -v -run Test_ServiceStage 2>&1 >> $OUTPUT
go test -v -run Test_Reorg 2>&1 >> $OUTPUT
go test -v -run Test_OPI 2>&1 >> $OUTPUT