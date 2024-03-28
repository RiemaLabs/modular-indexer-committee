go test -v -run Test_CatchupStage 2>&1 >> Output.log
go test -v -run Test_ServiceStage 2>&1 >> Output.log
go test -v -run Test_Reorg 2>&1 >> Output.log
go test -v -run Test_OPI 2>&1 >> Output.log