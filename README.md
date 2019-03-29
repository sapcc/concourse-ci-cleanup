# Concourse CI clean-up

This tool removes stale concourse workers when a node in Kubernetes does not exist anymore. Workers need to be named like the Kubernetes node to make this work.

Optionally it allows you to delete Openstack volumes with a specific pattern.

Usage:
```
$ ./concourse-ci-cleanup --help
Usage of ./ci-cleanup:
  -alsologtostderr
    	log to standard error as well as files
  -concourse-password string
    	Use concourse URL [CONCOURSE_PASSWORD]
  -concourse-url string
    	Use concourse URL [CONCOURSE_URL]
  -concourse-user string
    	Use concourse URL [CONCOURSE_USER]
  -context string
    	Use context
  -kubeconfig string
    	Use explicit kubeconfig file
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -os-application-credential-id string
    	Openstack application credential id [OS_APPLICATION_CREDENTIAL_ID]
  -os-application-credential-secret string
    	Openstack application credential secret [OS_APPLICATION_CREDENTIAL_SECRET]
  -os-auth-url string
    	Openstack auth url [OS_AUTH_URL]
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
  -volume-cleanup
    	Cleanup volumes in Openstack [VOLUME_CLEANUP]
  -volume-prefix string
    	Prefix to identify stale workers [VOLUME_PREFIX]
  -worker-prefix string
    	Prefix to identify stale workers [WORKER_PREFIX]
```

## License
This project is licensed under the Apache2 License - see the [LICENSE](LICENSE) file for details
