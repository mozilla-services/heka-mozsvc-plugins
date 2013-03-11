# heka_mozsvc_plugins.StatsdClient
mockgen -package="testsupport" \
                    -destination="testsupport/mock_statsdclient.go" \
                    github.com/mozilla-services/heka-mozsvc-plugins StatsdClient 
