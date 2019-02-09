paas-loki-exporter
==================

User haiku:

> Why's my app broken?
>
> The logs are stored in loki.
>
> Thanks paas-loki-exporter!

Reads logs from the [cloud foundry firehose](https://docs.cloudfoundry.org/loggregator/architecture.html#firehose),
writes them to [Loki](https://grafana.com/loki).

Status
------

Untested, not ready for use.

Testing manually
----------------

You can run Loki and Grafana locally with docker-compose, see
[docker-compose.yml](/scripts/docker-compose.yml).

Running the exporter is just a case of setting the right environment variables
and running the code. See [run.sh](/scripts/run.sh). Note that you'll need to
be logged into the cf cli as an administrator for this to work - admin creds
are required to read from the firehose.

