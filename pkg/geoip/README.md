# Geoip module

This module is used to get information about the geographical location of an IPv4 or IPv6 address. 


## Usage

To get the location of the system calling this function, use exported method `Fetch` from the package `geoip`

1. use `geoip.Fetch()`:

    This method uses 3 paths of geoip, It starts with first path of `geoipURLs` if any error was produced it continues and tries out the next path, REturnes the default unknown location and the error in case it coulnd't receive correct response from all paths.

## Tests

`geoip_test.go` tests the driver function `getLocation` which is called by the exported function `Fetch`
It mainly tests and validates:
1. Correct response.
2. Wrong response code.
3. Wrong response body.

#### Remark : Tests are computed on all 3 paths of `geoipURLs` to ensure correctness.