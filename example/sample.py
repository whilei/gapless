#!/usr/bin/env python
# -*- coding: utf-8 -*-

# See this link for apple json examples:
# http://developer.apple.com/library/mac/#documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/ApplePushService.html#//apple_ref/doc/uid/TP40008194-CH100-SW15

import redis
import json


r = redis.Redis()

for x in range(1, 10):
    payload = {
        "token": "fcd9163c28e0a3cf038c176a598171c128aede02c1eb43bbc4114d8f7097d814",
        "identifier": x,
        "expiry": 7200,
        "data": {
            "aps": {
                "alert": "You got your emails [%d]." % x,
                "badge": x,
                "sound": "default"
            },
            "acme1": "bar",
            "acme2": 42 * x
        }
    }

    payload = json.dumps(payload)
    r.rpush("beta_apns_queue", payload)
