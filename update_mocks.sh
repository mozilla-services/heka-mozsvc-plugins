# heka_mozsvc_plugins.StatsdClient
$GOPATH/bin/mockgen -package="testsupport" \
                    -source="statsdwriter.go" \
                    -destination="testsupport/mock_statsdclient.go" heka_mozsvc_plugins StatsdClient
