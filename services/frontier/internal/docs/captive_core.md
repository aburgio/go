---
title: Using Captive DigitalBits-Core in Frontier
replacement: https://developers.digitalbits.org/docs/frontier-captive-core/
---
## Using Captive DigitalBits-Core in Frontier

Starting with version v1.6.0, Frontier contains an experimental feature to use DigitalBits-Core in the captive mode. In this mode DigitalBits-Core is started as a subprocess of Frontier and streams ledger data over a filesystem pipe. It completely eliminates all issues connected to ledgers missing in DigitalBits-Core's database but requires extra time to initialize the DigitalBits-Core subprocess.

Captive DigitalBits-Core can be used in both reingestion and normal Frontier operations.

### Configuration

To enable captive mode three feature config variables are required:
* `ENABLE_CAPTIVE_CORE_INGESTION=true`,
* `DIGITALBITS_CORE_BINARY_PATH` - defines a path to the `digitalbits-core` binary,
* `CAPTIVE_CORE_CONFIG_APPEND_PATH` - (not required when running `frontier db reingest range`) defines a path to a file to append to the DigitalBits Core configuration file used by captive core. It must, at least, include enough details to define a quorum set. For instance, to connect to the DigitalBits testnet through `core-testnet1.digitalbits.org`:
  ```
  [[HOME_DOMAINS]]
  HOME_DOMAIN="testnet.digitalbits.org"
  QUALITY="MEDIUM"

  [[VALIDATORS]]
  NAME="sdf_testnet_1"
  HOME_DOMAIN="testnet.digitalbits.org"
  PUBLIC_KEY="GDKXE2OZMJIPOSLNA6N6F2BVCI3O777I2OOC4BV7VOYUEHYX7RTRYA7Y"
  ADDRESS="core-testnet1.digitalbits.org"
  ```
  
  The full configuration to be used will be printed out by Frontier when runnign frontier with `--log-level debug`
  
* (optional) `CAPTIVE_CORE_HTTP_PORT` - HTTP port for Captive Core to listen on (0 disables the HTTP server)

### Requirements

* An additional 3GB of RAM,
* Frontier v1.6.0,
* DigitalBits-Core v13.2.0.

### How it works

When using Captive DigitalBits-Core, Frontier runs the `digitalbits-core` binary as a subprocess. Then both processes communicate over filesystem pipe: DigitalBits-Core sends `xdr.LedgerCloseMeta` structs with information about each ledger and Frontier reads it.

The behaviour is slightly different when reingesting old ledgers and when reading recently closed ledgers.

When reingesting, DigitalBits-Core is started in a special `catchup` mode that simply replays the requested range of ledgers. This mode requires an additional 3GB of RAM because all ledger entries are stored in memory - making it extremely fast. This mode only depends on the history archives, so a `digitalbits-core.cfg` file is not required.

When reading recently closed ledgers, DigitalBits-Core is started with a normal `run` command. This requires a persistent database, so extra RAM is not needed, but it makes the initial stage of applying buckets slower than when reingesting. In this case a `digitalbits-core.cfg` file **is** required to configure a quorum set, so that DigitalBits-Core can connect to the DigitalBits network.

### Known Issues

As discussed earlier, Captive DigitalBits-Core provides much better decoupling for Frontier, at the expense of persistence. You should be aware of the following consequences:

* Captive DigitalBits-Core requires a couple of minutes to complete the apply buckets stage every time Frontier in started.
* If Frontier process is terminated, DigitalBits-Core is also terminated.
* Requires extra RAM.

To mitigate this we recommend running multiple ingesting Frontier servers in a single cluster. This allows other ingesting instances to maintain service without interruptions if a Captive DigitalBits-Core is restarted.

### Using Captive Core to reingest the full public network history

In some cases it can be convenient to (re)ingest the full DigitalBits Public
Network history into Frontier (e.g. when running Frontier for the first time).

This process used to take weeks.
However, using multiple Captive Core workers on a high performance
environment (powerful machine on which to run Frontier + powerful Database)
makes this possible in ~1.5 days.


The following instructions assume the reingestion is done on AWS.
However, they should be applicable to any other environment with equivalent
capacity. In the same way, the instructions can be adapted to reingest only
specify parts of the history.

##### Prerequisites

1. An `m5.8xlarge` (32 cores, 64GB of RAM) EC2 instance with at least 200 GB 
   of disk capacity from which to run Frontier.
   This is needed to fit 24 Frontier parallel workers (each with its own
   Captive Core instance). Each Core instance can take up to 3GB of RAM and a
   full core (more on why 24 workers below). If the number of workers is
   increased, you may need a larger machine.
   
2. Frontier version 1.6.0 or newer installed in the machine from (1).

3. [Core](https://github.com/digitalbits/digitalbits-core) version 13.0 or newer installed
   in the machine from (1).

4. A Frontier database, where to reingest the History. Preferably, the
   database should be at least an RDS `r4.8xlarge` instance or better (to take
   full advantage of its IOPS write capacity) and
   should be empty, to minimize storage (Postgres accumulates data during
   usage, which is only deleted when `VACUUM`ed). When using an RDS instance
   with General Purpose SSD storage, the reingestion throughput of the DB
   (namely Write IOPS) is determined by the storage size (3 IOPS per GB).
   With 5TB you get 15K IOPS, which can be saturated with 24 Frontier
   workers. As the DB storage grows,
   the IO capacity will grow along with it. The number of workers (and the
   size of the instance created in (1), should be increased accordingly if
   we want to take advantage of it. To make sure we are minimizing the
   reingestion time, we should look at the RDS _Write IOPS_ CloudWatch graph.
   The graph should ideally always be close to the theoretical limit of
   the DB (3000 IOPS per TB of storage).


##### Reingestion

Once the prerequisites are satisfied, we can spawn two Frontier reingestion
processes in parallel:

 1. One for the first 17 million ledgers (which are almost empty).
 2. Another one for the rest of the history.

This is due to first 17 million ledgers being almost empty whilst the rest
are much more packed. Having a single Frontier instance with enough workers to
saturate the IO capacity of the machine for the first 17 million would kill the
machine when reingesting the rest (during which there is a higher CPU and
memory consumption per woker).

64 workers for (1) and 24 workers for (2) saturates the IO capacity of an RDS
instance with 5TB of General Purpose SSD storage. Again, as the DB storage
grows, a larger number of workers should be considered.

In order to run the reingestion, first set the following environment variables:
```
export DATABASE_URL=postgres://postgres:<password>@<frontier_db_hostanme>:5432/frontier
export APPLY_MIGRATIONS=true
export HISTORY_ARCHIVE_URLS=https://s3-eu-west-1.amazonaws.com/history.digitalbits.org/prd/core-live/core_live_001
export NETWORK_PASSPHRASE="Public Global DigitalBits Network ; September 2015"
export DIGITALBITS_CORE_BINARY_PATH=/usr/bin/digitalbits-core
export ENABLE_CAPTIVE_CORE_INGESTION=true
# Number of ledgers per job sent to the workers
# The larger the job, the better performance from Captive Core's perspective, but, you want to choose a job size which maximizes the time all workers are busy. 
export PARALLEL_JOB_SIZE=100000
# Retries per job
export RETRIES=10
export RETRY_BACKOFF_SECONDS=20
```

If Frontier was previously running, ensure it is stopped. Then, run
the following commands in parallel:

1. `digitalbits-frontier db reingest range --parallel-workers=64 1 16999999`
2. `digitalbits-frontier db reingest range --parallel-workers=24 17000000 <latest_ledger>`

When saturating an RDS instance with 15K IOPS capacity:

(1) should take a few hours to complete.

(2) should take about 1.5 days to complete.


Although there is a retry mechanism, reingestion may fail half-way. Frontier will
print the recommended range to use in order to restart it. 


##### Monitoring reingestion process

This script should help monitor the reingestion process by printing the
 ledger subranges being reingested:
 
```
#!/bin/bash
echo "Current ledger ranges being reingested:"
echo
I=1
for S in $(ps aux | grep digitalbits-core | grep catchup |  awk '{ print $15 }' | sort -n ); do
    printf '%15s' $S
    if [ $(( I %  5 )) = 0 ]; then
	echo
    fi
    I=$(( I + 1))
done
```
 
Ideally we would be using Prometheus metrics for this, but they haven't been
implemented yet.

Here is an example run:

```
Current ledger ranges being reingested:
    99968/99968   199936/99968   299904/99968   399872/99968   499840/99968
   599808/99968   699776/99968   799744/99968   899712/99968   999680/99968
  1099648/99968  1199616/99968  1299584/99968  1399552/99968  1499520/99968
  1599488/99968  1699456/99968  1799424/99968  1899392/99968  1999360/99968
  2099328/99968  2199296/99968  2299264/99968  2399232/99968  2499200/99968
  2599168/99968  2699136/99968  2799104/99968  2899072/99968  2999040/99968
  3099008/99968  3198976/99968  3298944/99968  3398912/99968  3498880/99968
  3598848/99968  3698816/99968  3798784/99968  3898752/99968  3998720/99968
  4098688/99968  4198656/99968  4298624/99968  4398592/99968  4498560/99968
  4598528/99968  4698496/99968  4798464/99968  4898432/99968  4998400/99968
  5098368/99968  5198336/99968  5298304/99968  5398272/99968  5498240/99968
  5598208/99968  5698176/99968  5798144/99968  5898112/99968  5998080/99968
  6098048/99968  6198016/99968  6297984/99968  6397952/99968 17099967/99968
 17199935/99968 17299903/99968 17399871/99968 17499839/99968 17599807/99968
 17699775/99968 17799743/99968 17899711/99968 17999679/99968 18099647/99968
 18199615/99968 18299583/99968 18399551/99968 18499519/99968 18599487/99968
 18699455/99968 18799423/99968 18899391/99968 18999359/99968 19099327/99968
 19199295/99968 19299263/99968 19399231/99968
```
 

