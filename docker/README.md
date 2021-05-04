Quickstart for development with local Cadence server 
====================================

**Prerequisite**: [Docker + Docker compose](https://docs.docker.com/engine/installation/) 

Following steps will bring up the docker container running cadence server
along with all its dependencies (cassandra, prometheus, grafana). Exposes cadence
frontend on port 7933, web on port 8088, and grafana on port 3000.

```
cd $GOPATH/src/github.com/uber/cadence/docker
docker-compose up
```
> Note: Above command will run with `master-auto-setup` image, which is a changing image all the time.
> You can use a released image if you want a stable version. See the below section of "Using a released image".

To update your `master-auto-setup` image to the latest version
```
docker pull ubercadence/server:master-auto-setup

```

* View Cadence-Web at http://localhost:8088  
* View metrics at http://localhost:3000 , with default username/password: admin/admin    
  * Configure Prometheus as datasource: use `http://host.docker.internal:9090` as URL of prometheus.

Using different docker-compose files
-----------------------
By default `docker-compose up` will run with `docker-compose.yml` in this folder.
This compose file is running with Cassandra, with basic visibility, 
using Prometheus for emitting metric, with Grafana access. 


We also provide several other compose files for different features/modes:

* docker-compose-es.yml enables advanced visibility with ElasticSearch 6.x
* docker-compose-es-v7.yml enables advanced visibility with ElasticSearch 7.x
* docker-compose-mysql.yml uses MySQL as persistence storage
* docker-compose-postgres.yml uses PostgreSQL as persistence storage
* docker-compose-statsd.yaml runs with Statsd+Graphite
* docker-compose-multiclusters.yaml runs with 2 cadence clusters

For example:
```
docker-compose -f docker-compose-mysql.yml up
```

Also feel free to make your own to combine the above features.

Using a released image
-----------------------
The above compose files all using master image. It's taking the latest bits on the master branch of this repo.
You may want to use more stable version from our release process.

With every tagged release of the cadence server, there is also a corresponding
docker image that's uploaded to docker hub. In addition, the release will also
contain a **docker.tar.gz** file (docker-compose startup scripts). 

Go [here](https://github.com/uber/cadence/releases/latest) to download a latest **docker.tar.gz** 

Execute the following
commands to start a pre-built image along with all dependencies.

```
tar -xzvf docker.tar.gz
cd docker
docker-compose up
```

DIY: Building an image for any branch and restarting
-----------------------------------------
Replace **YOUR_TAG** and **YOUR_CHECKOUT_BRANCH** in the below command to build:
```
cd $GOPATH/src/github.com/uber/cadence
git checkout YOUR_CHECKOUT_BRANCH
docker build . -t ubercadence/server:YOUR_TAG
```
Or for auto-setup images:
```
cd $GOPATH/src/github.com/uber/cadence
git checkout YOUR_CHECKOUT_BRANCH
docker build . -t ubercadence/server:YOUR_TAG-auto-setup --build-arg TARGET=auto-setup
```
Replace the tag of **image: ubercadence/server** to **YOUR_TAG** in docker-compose.yml .
Then stop service and remove all containers using the below commands.
```
docker-compose down
docker-compose up
```

Note that with `TARGET=auto-setup`, the images will setup all the DB/ElasticSearch schema during startup.
By default, the image will not setup schema if you leave TARGET empty.

Using docker image for production
=========================
In a typical production setting, dependencies (cassandra / statsd server) are
managed / started independently of the cadence-server. To use the container in
a production setting, use the following docker command:

```
docker run -e CASSANDRA_SEEDS=10.x.x.x                  -- csv of cassandra server ipaddrs
    -e CASSANDRA_USER=<username>                        -- Cassandra username
    -e CASSANDRA_PASSWORD=<password>                    -- Cassandra password
    -e KEYSPACE=<keyspace>                              -- Cassandra keyspace
    -e VISIBILITY_KEYSPACE=<visibility_keyspace>        -- Cassandra visibility keyspace, if using basic visibility 
    -e KAFKA_SEEDS=10.x.x.x                             -- Kafka broker seed, if using ElasticSearch + Kafka for advanced visibility feature
    -e ES_SEEDS=10.x.x.x                                -- ElasticSearch seed , if using ElasticSearch + Kafka for advanced visibility feature
    -e RINGPOP_SEEDS=10.x.x.x,10.x.x.x                  -- csv of ipaddrs for gossip bootstrap
    -e STATSD_ENDPOINT=10.x.x.x:8125                    -- statsd server endpoint
    -e NUM_HISTORY_SHARDS=1024                          -- Number of history shards
    -e SERVICES=history,matching                        -- Spinup only the provided services, separated by commas, options are frontend,history,matching and workers
    -e LOG_LEVEL=debug,info                             -- Logging level
    -e DYNAMIC_CONFIG_FILE_PATH=<dynamic_config_file>   -- Dynamic config file to be watched, default to /etc/cadence/config/dynamicconfig/development.yaml, but you can choose /etc/cadence/config/dynamicconfig/development_es.yaml if using ElasticSearch
    ubercadence/server:<tag>
```
Note that each env variable has a default value, so you don't have to specify it if the default works for you. 
For more options to configure the docker, please refer to `config_template.yaml`.

For `<tag>`, use `auto-setup` images only for first initial setup, and use regular ones for production deployment. See the above explanation about `auto-setup`. 

When upgrading, follow the release instrusctions if version upgrades require some configuration or schema changes. 
