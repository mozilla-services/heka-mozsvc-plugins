# heka_mozsvc_plugins.StatsdClient
mockgen -package=testsupport \
        -destination=testsupport/mock_statsdclient.go \
        github.com/mozilla-services/heka-mozsvc-plugins StatsdClient

# pipeline.PluginHelper
mockgen -package=testsupport \
        -destination=testsupport/mock_pipeline_pluginhelper.go \
        github.com/mozilla-services/heka/pipeline PluginHelper

# pipeline.InputRunner
mockgen -package=testsupport \
        -destination=testsupport/mock_inputrunner.go \
        github.com/mozilla-services/heka/pipeline InputRunner

# pipeline.OutputRunner
mockgen -package=testsupport \
        -destination=testsupport/mock_pipeline.go \
        github.com/mozilla-services/heka/pipeline OutputRunner

# pipeline.DecoderRunner
mockgen -package=testsupport \
        -destination=testsupport/mock_decoderrunner.go \
        github.com/mozilla-services/heka/pipeline DecoderRunner

# aws.Service
mockgen -package=testsupport \
        -destination=testsupport/mock_aws_service.go \
        github.com/crowdmob/goamz/aws AWSService
