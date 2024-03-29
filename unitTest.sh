# cd ~/code/indexer-committee
go build
outpath="./testres/"
output="${outpath}$(date +%Y-%m-%d-%H-%M-%S)_unitTests.log"

# go test -v -run ./... 2>&1 >> $output

go test -v -run Test_CatchupStage 2>&1 >> $output
go test -v -run Test_ServiceStage 2>&1 >> $output
go test -v -run Test_Rollingback 2>&1 >> $output
go test -v -run Test_Reorg 2>&1 >> $output
go test -v -run Test_OPI 2>&1 >> $output
go test -v -run Test_APIs 2>&1 >> $output

cat ${output}