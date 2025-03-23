# benwh - Go package to get data out of a FranklinWH system.

## Overview

The `benwh` package provides a [Go
API](https://pkg.go.dev/zappem.net/pub/net/benwh) for finding and
reading the status of [FranklinWH API
systems](https://www.franklinwh.com/) services over the network. This
package is not in any way official, but seems to work well enough for
its intended purpose.

To use this package, you will need the account email and password as
well as the Site Device ID for your system. The non-password details
are visible from the official [FranklinWH
app](https://www.franklinwh.com/support/articles/detail/how-can-i-download-the-franklinwh-app):
More > Site Devices > Site 1 (etc, look for `SN:`).

```
$ git clone https://github.com/tinkerator/benwh.git
$ cd benwh
$ go run examples/status.go
2024/11/17 13:18:15 unable to read --config="./benwh.config": open ./benwh.config: no such file or directory
exit status 1
$ go run examples/status.go --newlogin
Email: xxx@test.com
Site Device ID (SN): xxxxxxxxxxxxxxxxxxxx
Password: *******
2024/11/17 15:37:51 (kW) Utility    Solar     Gen  A-Gate   House  %Charge
2024/11/17 15:37:51        0.855    0.000   0.000  -0.050   0.805   99.474
```

These values will be stored in a `benwh.config` file for use next
time. The `--config=/path/to/benwh.config` option can be used to
select a different config file location. The file will only be saved
with `--newlogin` if the entered info can be used to obtain a token
from the data server.

By default, the program attempts to connect once and capture a summary
of the device's current state. You can change this behavior with the
`--delay=<time-interval>` command line option, and a number of polled
requests. For example, fetching 10 samples of data collected 1 minute
apart:

```
$ go run examples/status.go --delay=1m --poll=10
2024/11/17 15:38:52 (kW) Utility    Solar     Gen  A-Gate   House  %Charge
2024/11/17 15:38:52        0.905    0.000   0.000  -0.042   0.863   99.474
2024/11/17 15:39:52        0.738    0.000   0.000  -0.038   0.700   99.474
2024/11/17 15:40:53        0.775    0.000   0.000  -0.046   0.729   99.474
2024/11/17 15:41:53        0.727    0.000   0.000  -0.046   0.681   99.474
2024/11/17 15:42:54        0.727    0.000   0.000  -0.039   0.688   99.474
2024/11/17 15:43:54        0.772    0.000   0.000  -0.039   0.733   99.474
2024/11/17 15:44:55        0.766    0.000   0.000  -0.042   0.724   99.474
2024/11/17 15:45:55        0.731    0.000   0.000  -0.038   0.693   99.474
2024/11/17 15:46:56        0.736    0.000   0.000  -0.038   0.698   99.474
2024/11/17 15:47:56        0.755    0.000   0.000  -0.046   0.709   99.474
```

## Credits

Searching online for specs for this system's API failed to turn up
anything that looks like official documentation. What I did find was
some [Python code](https://github.com/richo/franklinwh-python),
written by richö butts, that covered aspects of the MQTT access to the
device. It looks like this is how the official FranklinWH App gets
device status info. There are clearly no guarantees this API will
prove stable.

This Go package started out from the URL endpoints used by that Python
code and staring at the queries it makes when pointing it at `nc
-l`. This present package has just enough functionality to answer
questions like "is the utility power back?" and "how much charge has
the battery got left?".

## License info

The `benwh` package is distributed with the same BSD 3-clause
license as that used by [golang](https://golang.org/LICENSE) itself.

## Reporting bugs and feature requests

The `benwh` package has been developed for a friend who is the happy
owner of one of these systems.  If you find a bug or want to suggest a
feature addition, please use the [bug
tracker](https://github.com/tinkerator/benwh/issues).
