# Usage

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

    helm repo add redis-enforce-expire https://udhos.github.io/redis-enforce-expire

Update files from repo:

    helm repo update

Search redis-enforce-expire:

    $ helm search repo redis-enforce-expire -l --version ">=0.0.0"
    NAME                                     	CHART VERSION	APP VERSION	DESCRIPTION                                       
    redis-enforce-expire/redis-enforce-expire	0.0.4        	0.0.4      	Helm chart to install redis-enforce-expire into...
    redis-enforce-expire/redis-enforce-expire	0.0.3        	0.0.3      	Helm chart to install redis-enforce-expire into...
    redis-enforce-expire/redis-enforce-expire	0.0.2        	0.0.2      	Helm chart to install redis-enforce-expire into...
    redis-enforce-expire/redis-enforce-expire	0.0.1        	0.0.1      	Helm chart to install redis-enforce-expire into...

To install the charts:

    helm install my-redis-enforce-expire redis-enforce-expire/redis-enforce-expire
    #            ^                       ^                    ^
    #            |                       |                     \__________ chart
    #            |                       |
    #            |                        \_______________________________ repo
    #            |
    #             \_______________________________________________________ release (chart instance installed in cluster)

To uninstall the charts:

    helm uninstall my-redis-enforce-expire

# Source

<https://github.com/udhos/redis-enforce-expire>
