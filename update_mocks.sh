# heka_mozsvc_plugins.StatsdClient
mockgen -package=testsupport \
        -destination=testsupport/mock_statsdclient.go \
        github.com/mozilla-services/heka-mozsvc-plugins StatsdClient

# pipeline.PluginHelper
mockgen -package=testsupport \
        -destination=testsupport/mock_pipeline_pluginhelper.go \
        github.com/mozilla-services/heka/pipeline PluginHelper

# pipeline.OutputRunner
mockgen -package=testsupport \
        -destination=testsupport/mock_pipeline_outputrunner.go \
        github.com/mozilla-services/heka/pipeline OutputRunner
