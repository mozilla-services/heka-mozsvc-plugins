# heka_mozsvc_plugins.StatsdClient
$GOPATH/bin/mockgen -package="testsupport" \
                    -source="statsdwriter.go" \
                    -destination="testsupport/mock_statsdclient.go" heka_mozsvc_plugins StatsdClient

# heka.pipeline.WriteRunner
$GOPATH/bin/mockgen -package="heka_mozsvc_plugins" \
    -self_package="heka/testsupport" \
    -destination="mock_write_runner.go" heka/pipeline WriteRunner 


