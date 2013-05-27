# Gapless

Standalone Apple Push Notification Service with Connection Pool

This package builds into a standalone application which in turns listens to a
[Redis][1] server (using the [gosexy/redis][2] package) for any messages
to push out. Gapless uses a pool to make multiple connections to Apples push
notification servers. This allows for a wicked amount of throughput!

Gapless will retry failed pushes. Internally the app adds a key to the json
obj (`_gapless_RETRYING`) and will retry a total of three times. If the push
has failed three times, we log it as an error and forget about it.

## How to install

Install the app with `go get`. Be sure to add the second *gapless* in the path:

    $ go get github.com/cojac/gapless/gapless

## Usage

Once the app is installed, you can run it like so:

    $ ./gapless path_to_settings.json

That was the easy part. Just ensure the json settings are valid, and the paths
to you certificate files are valid. Then let Gapless run in a background process.
I've included a `gapless.initd` sample file in the example folder for your
reference.

### Sending messages to Gapless

Now, the more interesting part. How do I send push messages out!?!! Well first,
checkout the sample.py file in the example folder. Let's break it down into a
couple of parts.

#### Payload object

You need to create a payload string, which is just a json object... converted
into a string.

##### `token`

    Type: string
    Required: YES
    Default: ---

This is just the device token, as a string... not a hex value.

##### `identifier`

    Type: int
    Required: NO
    Default: 0

If you have a unique identifier (which is an int), I suggest you set it
here. If an error occurs while sending a push, it will log it and include this
identifier. This is helpful when beta tokens make it into production.

##### `expiry`

    Type: int
    Required: NO
    Default: 7200

If your end user has there device turned off, the push message will not be
delivered. This *expiry* value tells the Apples servers how long they should
hold on to this message before discarded it.

You specify how long you want your message to hang out (if undelivered) in
seconds. 7200 seconds == 2 hour. If you are pushing something like sports
scores, or something that changes frequently, you will want to lower this
value. Optionally you can pass 0 (zero) in... this will inform the push
servers to try once and discard it regardless of the delivery status.

##### `data`

    Type: dict
    Required: YES
    Default: ---

This is the actual payload that is passed to Apple. For more information on what
can be in this dict, view [Apple's documentation][3].

#### Sending the payload to Redis

Now that you have a json string (aka payload), you need to post it to Redis so
Gapless can send it on its way. The way to do that is to use [RPUSH][4].

    $ redis> RPUSH my_apns_queue_key "{ json data in string from }"

Gapless pops items from redis on the left, so it's important that you use RPUSH
to add new items to the end of the list. Then if you get backed up,
it will still eventually push everything out.

## Settings

Below are the available settings within Gapless. The headings are the json keys
(strings), and below them are the details pertaining to that key. Check the
json examples if anything is unclear.

### APNS Options

#### `apns_cert_path`

    Type: string
    Required: YES
    Default: ---


This is the file path to your cert.pem file. This can be relative to your
json settings file, or an absolute path.

Please note that this app uses two separate cert files... not a combination
of both the cert and key pems. Also the app assumes you are not using
encrypted certs.

#### `apns_key_path`

    Type: string
    Required: YES
    Default: ---

This is the file path to your key.pem file. This can be relative to your
json settings file, or an absolute path.

Please note that this app uses two separate cert files... not a combination
of both the cert and key pems. Also the app assumes you are not using
encrypted certs.

#### `apns_server`

    Type: string
    Required: YES
    Default: ---

This sets which Apple push endpoint you want to use. Likely you want either;

    `gateway.sandbox.push.apple.com:2195` for beta access
    `gateway.push.apple.com:2195` for production access

Alternatively, if you have a mock push server you can point to that for testing.

### Logging Options

#### `log_successes`

    Type: bool
    Required: NO
    Default: False

This controls the sent message output. If a notification is pushed
successfully, and this is set to *True*, then a message will be logged to
stdout like so:

    `Sent: {"token": "071c128e0114d8f7092843cbbc48c176a5981faede02c1e73a3dfbcd9163c81", "identifier": 154, "data": {"aps": {"sound": "default", "badge": 14, "alert": "You gottest some mail!"}, "acme1": "bar", "acme2": 13524}, "expiry": 7200}.`

Regardless of this value, Gapless will still log errors and warnings to stderr
as you would expect.

### Connection Pool Options

#### `pool_size`

    Type: int
    Required: NO
    Default: 2

This defines how many concurrent connections you want running. This will
likely be the trickiest setting to set. If you have very few notifications
going out, 2 connections will be more than adequate.

I suggest you monitor how your redis queue is doing (use LLEN)... if you find that
it doesn't ever reach zero (or continues to constantly grow), then start
bumping up the number of connections. Something like 10 connections isn't
to far-fetched, but you don't want a ton of idle connections hanging out...
so be conservative at first!


### Redis Options

#### `redis_db`

    Type: int
    Required: NO
    Default: 0

If you have multiple Redis DB's, you can specify which one you want to use.
I found this useful if your source app ever wants to flush the db... just
switch to a separate isolated DB ;)

#### `redis_host`

    Type: string
    Required: NO
    Default: "127.0.0.1"

Defaults to your localhost but if it's on another server, enter the
address here.

#### `redis_port`

    Type: int
    Required: NO
    Default: 6379

Defaults to 6379 which is the Redis default.

#### `redis_queue_key`

    Type: string
    Required: YES
    Default: ---

This is the key we tell Redis to listen for. You will be pushing to this key
from your source app, so choose wisely! If this value is not set, Gapless
will exit with an error.

## Other

Where is the APNS feedback function?!

Not here that's for sure! This is purely an app to get pushes out... fast. You
only need to pull in the feedback once a day. So I suggest you use your favorite
language and do there.

[1]: http://redis.io
[2]: https://github.com/gosexy/redis
[3]: http://developer.apple.com/library/mac/#documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/ApplePushService.html#//apple_ref/doc/uid/TP40008194-CH100-SW15
[4]: http://redis.io/commands/rpush
